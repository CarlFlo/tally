package torrent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/providers"
	"github.com/CarlFlo/tally/internal/settings"
)

type AutomationClientSource interface {
	Current(context.Context) (DownloadClient, error)
}

type AutomationService struct {
	DB       *database.Store
	Control  providers.Requester
	Clients  AutomationClientSource
	Now      func() time.Time
	OnChange func(profile, resource string)
}

type automationEpisode struct {
	ID, ShowID, ShowName, Premiered, Airstamp, Policy string
	Season, Episode, Runtime                           int
}

type automationCapabilities struct {
	Search     settings.Search
	Downloads  settings.Torrent
	Automation settings.TorrentAutomation
}

type automationCandidate struct {
	Result      SearchResult
	Assessment  ReleaseAssessment
	SizeProfile SizeProfileEvaluation
	ReleaseAge  ReleaseAgeEvaluation
}

func (s *AutomationService) Run(ctx context.Context) (int, error) {
	if s.DB == nil || s.Control == nil || s.Clients == nil {
		return 0, fmt.Errorf("torrent automation is not configured")
	}
	caps, err := s.capabilities(ctx)
	if err != nil {
		return 0, err
	}
	processed := 0
	if caps.Downloads.Enabled {
		verified, verifyErr := s.processPendingMagnetVerifications(ctx)
		if verifyErr != nil {
			return processed, verifyErr
		}
		processed += verified
	}
	if !caps.Automation.Enabled || !caps.Search.Enabled || !caps.Search.Configured() || !caps.Downloads.Enabled {
		return processed, nil
	}
	now := s.now()
	episodes, err := s.dueEpisodes(ctx, now, caps.Automation)
	if err != nil {
		return 0, err
	}
	for _, episode := range episodes {
		if err = ctx.Err(); err != nil {
			return processed, err
		}
		ready, err := s.retryReady(ctx, episode.ID, now, caps.Automation.RetryWindowHours)
		if err != nil {
			return processed, err
		}
		if !ready {
			continue
		}
		claimed, err := s.runEpisode(ctx, episode, caps)
		if err != nil {
			return processed, err
		}
		if claimed {
			processed++
		}
	}
	_ = (AutomationStore{DB: s.DB}).PruneRuns(ctx, now.AddDate(0, 0, -90))
	return processed, nil
}


const pendingMagnetVerificationMaxAge = 24 * time.Hour

func (s *AutomationService) processPendingMagnetVerifications(ctx context.Context) (int, error) {
	store := AutomationStore{DB: s.DB}
	pending, err := store.PendingMagnetVerifications(ctx, 50)
	if err != nil || len(pending) == 0 {
		return 0, err
	}
	client, err := s.Clients.Current(ctx)
	if err != nil {
		return 0, err
	}
	inspector, ok := client.(ResolvedFileClient)
	if !ok {
		return 0, fmt.Errorf("the configured torrent client cannot inspect magnet file metadata")
	}
	snapshot, err := client.Downloads(ctx, TallyCategory)
	if err != nil {
		return 0, err
	}
	existing := make(map[string]Download, len(snapshot.Torrents))
	for _, item := range snapshot.Torrents {
		existing[strings.ToLower(item.Hash)] = item
	}

	completed := 0
	now := s.now()
	for _, item := range pending {
		if err = ctx.Err(); err != nil {
			return completed, err
		}
		download, exists := existing[strings.ToLower(item.InfoHash)]
		if !exists {
			if now.Sub(time.Unix(item.StartedAt, 0)) >= pendingMagnetVerificationMaxAge {
				if err = store.FinishMagnetVerification(ctx, item.RunID, "unavailable", ReleaseAssessment{
					Confidence: ConfidenceHigh, Verification: VerificationUnverified, InfoHash: item.InfoHash,
				}, SizeProfileEvaluation{}, "torrent no longer exists in the Tally qBittorrent category"); err != nil {
					return completed, err
				}
				completed++
				s.publishChange()
			} else {
				_ = store.RecordMagnetVerificationAttempt(ctx, item.RunID, "torrent not visible in the Tally qBittorrent category yet")
			}
			continue
		}
		files, fileErr := inspector.ResolvedFiles(ctx, item.InfoHash)
		if fileErr != nil {
			if now.Sub(time.Unix(item.StartedAt, 0)) >= pendingMagnetVerificationMaxAge {
				if err = store.FinishMagnetVerification(ctx, item.RunID, "unavailable", ReleaseAssessment{
					Confidence: ConfidenceHigh, Verification: VerificationUnverified, InfoHash: item.InfoHash,
				}, SizeProfileEvaluation{}, fileErr.Error()); err != nil {
					return completed, err
				}
				completed++
				s.publishChange()
			} else {
				_ = store.RecordMagnetVerificationAttempt(ctx, item.RunID, fileErr.Error())
			}
			continue
		}
		if len(files) == 0 {
			if now.Sub(time.Unix(item.StartedAt, 0)) >= pendingMagnetVerificationMaxAge {
				if err = store.FinishMagnetVerification(ctx, item.RunID, "unavailable", ReleaseAssessment{
					Confidence: ConfidenceHigh, Verification: VerificationUnverified, InfoHash: item.InfoHash,
				}, SizeProfileEvaluation{}, "magnet metadata was not resolved within 24 hours"); err != nil {
					return completed, err
				}
				completed++
				s.publishChange()
			} else {
				_ = store.RecordMagnetVerificationAttempt(ctx, item.RunID, "")
			}
			continue
		}

		var submission struct {
			Automation       settings.TorrentAutomation `json:"automation"`
			ShowMediaProfile ShowMediaProfile            `json:"show_media_profile"`
			RuntimeMinutes   int                         `json:"runtime_minutes"`
		}
		if err = json.Unmarshal(item.SettingsSnapshot, &submission); err != nil {
			return completed, fmt.Errorf("stored magnet verification settings are invalid: %w", err)
		}
		profile := submission.ShowMediaProfile.Effective
		if profile != MediaProfileAnimated {
			profile = MediaProfileLive
		}
		target := EpisodeTarget{ShowTitle: item.ShowName, Season: item.Season, Episode: item.Episode}
		base := ReleaseAssessment{
			Confidence: ConfidenceHigh, Verification: VerificationUnverified,
			Parsed: ParseReleaseName(item.SelectedName), InfoHash: item.InfoHash,
		}
		resolvedName := strings.TrimSpace(download.Name)
		assessment, verifyErr := VerifyResolvedFilesForTarget(base, item.InfoHash, resolvedName, files, target)
		if verifyErr == nil && assessment.Confidence != ConfidenceHigh {
			assessment.HardRejections = appendReason(assessment.HardRejections, AssessmentReason{Code: ReasonAmbiguousIdentity, Detail: "post_magnet_payload"})
			assessment.Confidence = ConfidenceRejected
			verifyErr = verificationError(assessment)
		}
		sizeProfile := SizeProfileEvaluation{}
		if assessment.Payload != nil {
			sizeProfile = EvaluateSizeProfile(assessment.Payload.TotalSize, submission.RuntimeMinutes, submission.Automation, profile)
			if sizeProfile.Known && !sizeProfile.InActiveRange {
				assessment.HardRejections = appendReason(assessment.HardRejections, AssessmentReason{Code: ReasonSizeOutOfRange})
				assessment.Confidence = ConfidenceRejected
				verifyErr = verificationError(assessment)
			}
		}
		if verifyErr != nil || assessment.Rejected() {
			_ = client.Stop(ctx, item.InfoHash)
			if removeErr := client.Remove(ctx, item.InfoHash, true); removeErr != nil {
				_ = store.RecordMagnetVerificationAttempt(ctx, item.RunID, "payload rejected but qBittorrent removal failed: "+removeErr.Error())
				return completed, removeErr
			}
			if err = store.RejectMagnetVerification(ctx, item.RunID, assessment, sizeProfile, verifyErr.Error()); err != nil {
				return completed, err
			}
			completed++
			s.publishChange()
			continue
		}
		if err = store.FinishMagnetVerification(ctx, item.RunID, "verified", assessment, sizeProfile, ""); err != nil {
			return completed, err
		}
		completed++
		s.publishChange()
	}
	return completed, nil
}

func (s *AutomationService) runEpisode(ctx context.Context, episode automationEpisode, caps automationCapabilities) (bool, error) {
	target, err := s.episodeTarget(ctx, episode)
	if err != nil {
		return false, err
	}
	store := AutomationStore{DB: s.DB}
	mediaProfile, err := store.ShowMediaProfile(ctx, episode.ShowID)
	if err != nil {
		return false, err
	}
	query := fmt.Sprintf("%s S%02dE%02d", episode.ShowName, episode.Season, episode.Episode)
	snapshot, _ := json.Marshal(map[string]any{
		"automation":         caps.Automation,
		"show_policy":        episode.Policy,
		"show_media_profile": mediaProfile,
		"runtime_minutes":    episode.Runtime,
		"search_enabled":     caps.Search.Enabled,
		"downloads_enabled":  caps.Downloads.Enabled,
	})
	runID, err := store.StartRun(ctx, AutomationRun{
		ShowID: episode.ShowID, EpisodeID: episode.ID, ShowName: episode.ShowName,
		Season: episode.Season, Episode: episode.Episode, Query: query,
		SettingsSnapshot: snapshot,
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return false, nil
		}
		return false, err
	}
	started := s.now()
	finish := func(status AutomationRunStatus, assessment ReleaseAssessment, name string) error {
		return store.FinishRun(ctx, runID, status, assessment, name)
	}

	provider := &Jackett{Control: s.Control, BaseURL: caps.Search.BaseURL, APIKey: caps.Search.APIKey}
	searchStarted := s.now()
	// Keep discovery broad here. Automation-specific rejection happens below so
	// Previous Runs can explain exactly why Jackett candidates were removed.
	results, err := provider.Search(ctx, SearchQuery{Query: query})
	if err != nil {
		_ = store.AppendDecision(ctx, runID, DecisionStep{Stage: "search", Status: "failed", Summary: "Jackett search failed", DurationMS: elapsedMS(searchStarted, s.now())})
		_ = finish(RunFailed, ReleaseAssessment{}, "")
		return true, err
	}
	_ = store.AppendDecision(ctx, runID, DecisionStep{
		Stage: "search", Status: "success", Summary: fmt.Sprintf("%d candidates returned by Jackett", len(results)),
		Data: map[string]any{"candidate_count": len(results)}, DurationMS: elapsedMS(searchStarted, s.now()),
	})

	valid := make([]automationCandidate, 0, len(results))
	rejected, magnetOnly, previouslyBad, unusable := 0, 0, 0, 0
	belowSeeders, keywordFiltered, groupFiltered, uploaderFiltered, nonHigh, sizeRateFiltered, releaseDelayFiltered := 0, 0, 0, 0, 0, 0, 0
	for _, result := range results {
		if result.Seeders < caps.Automation.MinSeeders {
			belowSeeders++
			continue
		}
		if !automationKeywordsMatch(result.Name, caps.Automation) {
			keywordFiltered++
			continue
		}
		assessment := EvaluateSearchCandidate(result, target)
		if assessment.Rejected() {
			rejected++
			continue
		}
		if !automationGroupAllowed(assessment.Parsed.Group, caps.Automation) {
			groupFiltered++
			continue
		}
		if !automationUploaderAllowed(result.Uploader, caps.Automation) {
			uploaderFiltered++
			continue
		}
		knownHash := result.InfoHash
		if knownHash == "" {
			knownHash = MagnetInfoHash(result.Magnet)
		}
		if knownHash != "" {
			bad, checkErr := store.IsBadInfoHash(ctx, knownHash)
			if checkErr != nil {
				_ = finish(RunFailed, ReleaseAssessment{}, "")
				return true, checkErr
			}
			if bad {
				previouslyBad++
				continue
			}
		}
		if assessment.Confidence != ConfidenceHigh {
			nonHigh++
			continue
		}
		releaseAge := EvaluateReleaseAge(result.Published, s.now(), time.Duration(caps.Automation.ReleaseDelayMinutes)*time.Minute)
		sizeProfile := EvaluateSizeProfile(result.Size, episode.Runtime, caps.Automation, mediaProfile.Effective)
		if sizeProfile.Known && !sizeProfile.InActiveRange {
			sizeRateFiltered++
			continue
		}
		if result.URL == "" && !ValidMagnet(result.Magnet) {
			unusable++
			continue
		}
		if result.URL == "" && ValidMagnet(result.Magnet) {
			magnetOnly++
		}
		valid = append(valid, automationCandidate{Result: result, Assessment: assessment, SizeProfile: sizeProfile, ReleaseAge: releaseAge})
	}
	if !ReleaseDelayWindowOpen(candidateReleaseAges(valid)) {
		ready := valid[:0]
		for _, candidate := range valid {
			if candidate.ReleaseAge.Known {
				releaseDelayFiltered++
				continue
			}
			ready = append(ready, candidate)
		}
		valid = ready
	}
	rankAutomationCandidates(valid, caps.Automation)
	if len(valid) > caps.Automation.MaxCandidates {
		valid = valid[:caps.Automation.MaxCandidates]
	}
	_ = store.AppendDecision(ctx, runID, DecisionStep{
		Stage: "filter", Status: "success", Summary: fmt.Sprintf("%d candidates remained for selection", len(valid)),
		Data: map[string]any{
			"rejected": rejected, "below_min_seeders": belowSeeders, "keyword_filtered": keywordFiltered,
			"group_filtered": groupFiltered, "uploader_filtered": uploaderFiltered, "non_high_confidence": nonHigh,
			"size_rate_filtered": sizeRateFiltered, "release_delay_filtered": releaseDelayFiltered, "magnet_only": magnetOnly, "unusable": unusable, "previously_bad": previouslyBad,
			"shortlisted": len(valid), "candidates": candidateAuditRows(valid, caps.Automation),
		},
	})

	for index, candidate := range valid {
		if err = ctx.Err(); err != nil {
			background := context.Background()
			_ = store.AppendDecision(background, runID, DecisionStep{Stage: "decision", Status: "cancelled", Summary: "Automation run was cancelled"})
			_ = store.FinishRun(background, runID, RunCancelled, ReleaseAssessment{}, "")
			return true, err
		}

		assessment := candidate.Assessment
		assessment.InfoHash = candidate.Result.InfoHash
		if assessment.InfoHash == "" {
			assessment.InfoHash = MagnetInfoHash(candidate.Result.Magnet)
		}
		var torrentData []byte
		useTorrent := false
		inspectionStarted := s.now()

		if candidate.Result.URL != "" {
			data, fetchErr := provider.FetchTorrent(ctx, candidate.Result.URL)
			if fetchErr == nil {
				verificationCandidate := candidate.Result
				if verificationCandidate.InfoHash == "" {
					verificationCandidate.InfoHash = MagnetInfoHash(verificationCandidate.Magnet)
				}
				verified, verifyErr := VerifyTorrentForTarget(candidate.Assessment, verificationCandidate, data, &target)
				assessment = verified
				if verifyErr == nil && assessment.InfoHash != "" {
					bad, checkErr := store.IsBadInfoHash(ctx, assessment.InfoHash)
					if checkErr != nil {
						_ = finish(RunFailed, assessment, candidate.Result.Name)
						return true, checkErr
					}
					if bad {
						assessment = RejectKnownBadInfoHash(assessment)
						verifyErr = verificationError(assessment)
					}
				}
				if verifyErr == nil && assessment.EligibleForAutomaticDownload() {
					verifiedSize := candidate.SizeProfile
					if assessment.Payload != nil {
						verifiedSize = EvaluateSizeProfile(assessment.Payload.TotalSize, episode.Runtime, caps.Automation, mediaProfile.Effective)
					}
					if verifiedSize.Known && !verifiedSize.InActiveRange {
						_ = store.AppendDecision(ctx, runID, DecisionStep{
							Stage: "inspection", Status: "rejected", Summary: "Verified torrent size fell outside the active MB/min range",
							Data: map[string]any{"rank": index + 1, "name": candidate.Result.Name, "size_profile": verifiedSize, "payload": assessment.Payload},
							DurationMS: elapsedMS(inspectionStarted, s.now()),
						})
						continue
					}
					candidate.SizeProfile = verifiedSize
					torrentData = data
					useTorrent = true
					_ = store.AppendDecision(ctx, runID, DecisionStep{
						Stage: "inspection", Status: "success", Summary: "Candidate payload verified from Jackett torrent metadata",
						Data: map[string]any{
							"rank": index + 1, "name": candidate.Result.Name, "infohash": assessment.InfoHash,
							"confidence": assessment.Confidence, "verification": assessment.Verification, "payload": assessment.Payload,
							"size_profile": candidate.SizeProfile,
						}, DurationMS: elapsedMS(inspectionStarted, s.now()),
					})
				} else if !hasReason(assessment.Reasons, ReasonTorrentMetadataFailed) || !ValidMagnet(candidate.Result.Magnet) {
					_ = store.AppendDecision(ctx, runID, DecisionStep{
						Stage: "inspection", Status: "rejected", Summary: "Candidate failed torrent payload inspection",
						Data: map[string]any{
							"rank": index + 1, "name": candidate.Result.Name, "confidence": assessment.Confidence,
							"verification": assessment.Verification, "hard_rejections": assessment.HardRejections, "payload": assessment.Payload,
							"size_profile": candidate.SizeProfile,
						}, DurationMS: elapsedMS(inspectionStarted, s.now()),
					})
					continue
				}
			} else if !ValidMagnet(candidate.Result.Magnet) {
				_ = store.AppendDecision(ctx, runID, DecisionStep{
					Stage: "inspection", Status: "rejected", Summary: "Jackett torrent metadata could not be fetched and no magnet fallback is available",
					Data: map[string]any{"rank": index + 1, "name": candidate.Result.Name, "reason": "metadata_fetch_failed"},
					DurationMS: elapsedMS(inspectionStarted, s.now()),
				})
				continue
			}
		}

		if !useTorrent {
			if !ValidMagnet(candidate.Result.Magnet) || assessment.Confidence != ConfidenceHigh || assessment.Rejected() {
				continue
			}
			assessment.Verification = VerificationUnverified
			if assessment.InfoHash == "" {
				assessment.InfoHash = MagnetInfoHash(candidate.Result.Magnet)
			}
			if normalizeInfoHash(assessment.InfoHash) == "" {
				continue
			}
			if assessment.InfoHash != "" {
				bad, checkErr := store.IsBadInfoHash(ctx, assessment.InfoHash)
				if checkErr != nil {
					_ = finish(RunFailed, assessment, candidate.Result.Name)
					return true, checkErr
				}
				if bad {
					continue
				}
			}
			_ = store.AppendDecision(ctx, runID, DecisionStep{
				Stage: "inspection", Status: "metadata_only", Summary: "Using High-confidence magnet fallback without payload verification",
				Data: map[string]any{
					"rank": index + 1, "name": candidate.Result.Name, "infohash": assessment.InfoHash,
					"confidence": assessment.Confidence, "verification": assessment.Verification, "size_profile": candidate.SizeProfile,
				}, DurationMS: elapsedMS(inspectionStarted, s.now()),
			})
		}

		latest, capabilityErr := s.capabilities(ctx)
		if capabilityErr != nil {
			_ = finish(RunFailed, assessment, candidate.Result.Name)
			return true, capabilityErr
		}
		if !latest.Automation.Enabled || !latest.Search.Enabled || !latest.Downloads.Enabled {
			_ = store.AppendDecision(ctx, runID, DecisionStep{Stage: "decision", Status: "skipped", Summary: "Automation or torrent capability was disabled before submission"})
			if err = finish(RunSkipped, assessment, candidate.Result.Name); err != nil {
				return true, err
			}
			s.publishChange()
			return true, nil
		}
		client, clientErr := s.Clients.Current(ctx)
		if clientErr != nil {
			_ = store.AppendDecision(ctx, runID, DecisionStep{Stage: "download", Status: "failed", Summary: "Torrent client is not configured"})
			_ = finish(RunFailed, assessment, candidate.Result.Name)
			return true, clientErr
		}
		if !useTorrent {
			if _, ok := client.(ResolvedFileClient); !ok {
				_ = store.AppendDecision(ctx, runID, DecisionStep{Stage: "inspection", Status: "rejected", Summary: "Torrent client cannot provide post-magnet file verification"})
				continue
			}
		}
		downloadStarted := s.now()
		submissionType := "magnet"
		if useTorrent {
			submissionType = "torrent"
			clientErr = client.AddTorrent(ctx, torrentData)
		} else {
			clientErr = client.AddMagnet(ctx, candidate.Result.Magnet)
		}
		if clientErr != nil && assessment.InfoHash != "" {
			if clientSnapshot, reconcileErr := client.Downloads(ctx, TallyCategory); reconcileErr == nil && snapshotHasHash(clientSnapshot, assessment.InfoHash) {
				clientErr = nil
			}
		}
		if clientErr != nil {
			_ = store.AppendDecision(ctx, runID, DecisionStep{
				Stage: "download", Status: "failed", Summary: "Torrent client submission failed",
				Data: map[string]any{"name": candidate.Result.Name, "infohash": assessment.InfoHash, "submission_type": submissionType},
				DurationMS: elapsedMS(downloadStarted, s.now()),
			})
			_ = finish(RunFailed, assessment, candidate.Result.Name)
			return true, clientErr
		}
		_ = store.AppendDecision(ctx, runID, DecisionStep{
			Stage: "decision", Status: "selected", Summary: "Best suitable candidate selected",
			Data: map[string]any{
				"name": candidate.Result.Name, "confidence": assessment.Confidence, "verification": assessment.Verification,
				"submission_type": submissionType, "size_profile": candidate.SizeProfile,
				"preferences": AutomationPreferenceSignals(candidate.Result, candidate.Assessment.Parsed, caps.Automation),
			},
		})
		_ = store.AppendDecision(ctx, runID, DecisionStep{
			Stage: "download", Status: "success", Summary: "Sent to torrent client successfully",
			Data: map[string]any{"infohash": assessment.InfoHash, "submission_type": submissionType}, DurationMS: elapsedMS(downloadStarted, s.now()),
		})
		if useTorrent {
			err = finish(RunDownloaded, assessment, candidate.Result.Name)
		} else {
			err = store.FinishMagnetRun(ctx, runID, assessment, candidate.Result.Name)
		}
		if err != nil {
			if !useTorrent {
				_ = client.Stop(ctx, assessment.InfoHash)
				_ = client.Remove(ctx, assessment.InfoHash, true)
			}
			return true, err
		}
		s.publishChange()
		return true, nil
	}

	_ = store.AppendDecision(ctx, runID, DecisionStep{
		Stage: "decision", Status: "no_verified_candidate", Summary: "No suitable High-confidence candidate was available for automatic download",
		Data: map[string]any{"elapsed_ms": elapsedMS(started, s.now())},
	})
	if err = finish(RunNoVerifiedCandidate, ReleaseAssessment{Verification: VerificationUnverified}, ""); err != nil {
		return true, err
	}
	s.publishChange()
	return true, nil
}

func (s *AutomationService) capabilities(ctx context.Context) (automationCapabilities, error) {
	store := settings.Store{DB: s.DB}
	var out automationCapabilities
	if _, err := store.Load(ctx, "search", &out.Search); err != nil {
		return out, err
	}
	out.Search = out.Search.Effective()
	if _, err := store.Load(ctx, "torrent", &out.Downloads); err != nil {
		return out, err
	}
	if _, err := store.Load(ctx, "torrent_automation", &out.Automation); err != nil {
		return out, err
	}
	out.Automation = out.Automation.Effective()
	return out, nil
}

func (s *AutomationService) dueEpisodes(ctx context.Context, now time.Time, config settings.TorrentAutomation) ([]automationEpisode, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT DISTINCT e.id,e.show_id,s.name,s.premiered,e.season,e.number,e.airstamp,COALESCE(NULLIF(e.runtime,0),NULLIF(s.runtime,0),0),COALESCE(p.policy,'default')
		FROM episodes e
		JOIN shows s ON s.id=e.show_id
		JOIN profile_shows f ON f.show_id=e.show_id
		LEFT JOIN torrent_show_policy p ON p.show_id=e.show_id
		WHERE e.airstamp<>'' AND e.number>0
		AND COALESCE(p.policy,'default') IN ('default','auto')
		AND NOT EXISTS(
			SELECT 1
			FROM torrent_automation_runs r
			LEFT JOIN torrent_magnet_verifications m ON m.run_id=r.id
			WHERE r.episode_id=e.id
			AND (
				r.status='running'
				OR (r.status='downloaded' AND COALESCE(m.status,'')<>'rejected')
			)
		)
		ORDER BY e.airstamp ASC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]automationEpisode, 0)
	delay := time.Duration(config.ReleaseDelayMinutes) * time.Minute
	window := time.Duration(config.RetryWindowHours) * time.Hour
	for rows.Next() {
		var episode automationEpisode
		if err = rows.Scan(&episode.ID, &episode.ShowID, &episode.ShowName, &episode.Premiered, &episode.Season, &episode.Episode, &episode.Airstamp, &episode.Runtime, &episode.Policy); err != nil {
			return nil, err
		}
		aired, parseErr := time.Parse(time.RFC3339, episode.Airstamp)
		if parseErr != nil {
			continue
		}
		age := now.Sub(aired)
		if age >= delay && age <= window {
			out = append(out, episode)
		}
	}
	return out, rows.Err()
}

func (s *AutomationService) retryReady(ctx context.Context, episodeID string, now time.Time, retryWindowHours int) (bool, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT CASE
			WHEN r.status='downloaded' AND m.status='rejected' THEN COALESCE(m.completed_at,r.started_at)
			ELSE r.started_at
		END AS retry_at
		FROM torrent_automation_runs r
		LEFT JOIN torrent_magnet_verifications m ON m.run_id=r.id
		WHERE r.episode_id=?
		AND (
			r.status NOT IN ('running','downloaded')
			OR (r.status='downloaded' AND m.status='rejected')
		)
		AND CASE
			WHEN r.status='downloaded' AND m.status='rejected' THEN COALESCE(m.completed_at,r.started_at)
			ELSE r.started_at
		END >=?
		ORDER BY retry_at DESC`, episodeID, now.Add(-time.Duration(retryWindowHours)*time.Hour).Unix())
	if err != nil {
		return false, err
	}
	defer rows.Close()
	attempts := 0
	var latest int64
	for rows.Next() {
		var started int64
		if err = rows.Scan(&started); err != nil {
			return false, err
		}
		if attempts == 0 {
			latest = started
		}
		attempts++
	}
	if err = rows.Err(); err != nil {
		return false, err
	}
	if attempts == 0 {
		return true, nil
	}
	var delay time.Duration
	switch attempts {
	case 1:
		delay = 30 * time.Minute
	case 2:
		delay = 2 * time.Hour
	default:
		delay = 6 * time.Hour
	}
	return now.Sub(time.Unix(latest, 0)) >= delay, nil
}

func (s *AutomationService) episodeTarget(ctx context.Context, episode automationEpisode) (EpisodeTarget, error) {
	target := EpisodeTarget{ShowTitle: episode.ShowName, Season: episode.Season, Episode: episode.Episode}
	if len(episode.Premiered) >= 4 {
		target.Year, _ = strconv.Atoi(episode.Premiered[:4])
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT provider,external_id FROM external_ids WHERE kind='show' AND internal_id=?", episode.ShowID)
	if err != nil {
		return target, err
	}
	defer rows.Close()
	for rows.Next() {
		var provider, id string
		if err = rows.Scan(&provider, &id); err != nil {
			return target, err
		}
		switch strings.ToLower(provider) {
		case "tvmaze":
			target.TVMazeID = id
		case "tvdb":
			target.TVDBID = id
		case "tmdb":
			target.TMDBID = id
		case "imdb":
			target.IMDBID = id
		}
	}
	return target, rows.Err()
}

func candidateReleaseAges(items []automationCandidate) []ReleaseAgeEvaluation {
	ages := make([]ReleaseAgeEvaluation, 0, len(items))
	for _, item := range items {
		ages = append(ages, item.ReleaseAge)
	}
	return ages
}

func rankAutomationCandidates(items []automationCandidate, config settings.TorrentAutomation) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		leftTrust := automationPreferenceScore(left.Result, left.Assessment.Parsed, config)
		rightTrust := automationPreferenceScore(right.Result, right.Assessment.Parsed, config)
		if leftTrust != rightTrust {
			return leftTrust > rightTrust
		}
		leftQuality := automationQualityScore(left.Assessment.Parsed.Resolution, config.PreferredQuality)
		rightQuality := automationQualityScore(right.Assessment.Parsed.Resolution, config.PreferredQuality)
		if leftQuality != rightQuality {
			return leftQuality > rightQuality
		}
		if left.SizeProfile.Known && right.SizeProfile.Known && left.SizeProfile.ActiveScore != right.SizeProfile.ActiveScore {
			return left.SizeProfile.ActiveScore > right.SizeProfile.ActiveScore
		}
		if left.Result.Seeders != right.Result.Seeders {
			return left.Result.Seeders > right.Result.Seeders
		}
		if config.PreferSmaller && left.Result.Size > 0 && right.Result.Size > 0 && left.Result.Size != right.Result.Size {
			return left.Result.Size < right.Result.Size
		}
		return left.Result.Name < right.Result.Name
	})
}

func automationQualityScore(resolution, preferred string) int {
	if preferred != settings.TorrentQualityBest && resolution == preferred {
		return 100
	}
	if preferred != settings.TorrentQualityBest && resolution != "" {
		return 10
	}
	switch resolution {
	case settings.TorrentQuality2160:
		return 40
	case settings.TorrentQuality1080:
		return 30
	case settings.TorrentQuality720:
		return 20
	default:
		return 0
	}
}

func candidateAuditRows(items []automationCandidate, config settings.TorrentAutomation) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for index, item := range items {
		rows = append(rows, map[string]any{
			"rank": index + 1, "name": item.Result.Name, "provider": item.Result.Provider,
			"uploader": item.Result.Uploader, "seeders": item.Result.Seeders, "size": item.Result.Size,
			"confidence": item.Assessment.Confidence, "parsed": item.Assessment.Parsed,
			"preferences": AutomationPreferenceSignals(item.Result, item.Assessment.Parsed, config),
			"size_profile": item.SizeProfile, "release_age": item.ReleaseAge,
		})
	}
	return rows
}

func snapshotHasHash(snapshot DownloadSnapshot, hash string) bool {
	for _, item := range snapshot.Torrents {
		if strings.EqualFold(item.Hash, hash) {
			return true
		}
	}
	return false
}

func elapsedMS(start, end time.Time) int64 {
	if end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}

func (s *AutomationService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *AutomationService) publishChange() {
	if s.OnChange != nil {
		s.OnChange("", "torrent-automation-runs")
	}
}
