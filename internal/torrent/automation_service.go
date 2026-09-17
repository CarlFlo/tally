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
	Season, Episode                                    int
}

type automationCapabilities struct {
	Search     settings.Search
	Downloads  settings.Torrent
	Automation settings.TorrentAutomation
}

type automationCandidate struct {
	Result     SearchResult
	Assessment ReleaseAssessment
}

func (s *AutomationService) Run(ctx context.Context) (int, error) {
	if s.DB == nil || s.Control == nil || s.Clients == nil {
		return 0, fmt.Errorf("torrent automation is not configured")
	}
	caps, err := s.capabilities(ctx)
	if err != nil {
		return 0, err
	}
	if !caps.Automation.Enabled || !caps.Search.Enabled || !caps.Search.Configured() || !caps.Downloads.Enabled {
		return 0, nil
	}
	now := s.now()
	episodes, err := s.dueEpisodes(ctx, now, caps.Automation)
	if err != nil {
		return 0, err
	}
	processed := 0
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

func (s *AutomationService) runEpisode(ctx context.Context, episode automationEpisode, caps automationCapabilities) (bool, error) {
	target, err := s.episodeTarget(ctx, episode)
	if err != nil {
		return false, err
	}
	query := fmt.Sprintf("%s S%02dE%02d", episode.ShowName, episode.Season, episode.Episode)
	snapshot, _ := json.Marshal(map[string]any{
		"automation":        caps.Automation,
		"show_policy":       episode.Policy,
		"search_enabled":    caps.Search.Enabled,
		"downloads_enabled": caps.Downloads.Enabled,
	})
	store := AutomationStore{DB: s.DB}
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
	results, err := provider.Search(ctx, SearchQuery{Query: query, MinSeeders: caps.Automation.MinSeeders})
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
	rejected, magnetOnly, previouslyBad := 0, 0, 0
	for _, result := range results {
		assessment := EvaluateSearchCandidate(result, target)
		if assessment.Rejected() {
			rejected++
			continue
		}
		if result.InfoHash != "" {
			bad, checkErr := store.IsBadInfoHash(ctx, result.InfoHash)
			if checkErr != nil {
				_ = finish(RunFailed, ReleaseAssessment{}, "")
				return true, checkErr
			}
			if bad {
				previouslyBad++
				continue
			}
		}
		if result.URL == "" {
			magnetOnly++
			continue
		}
		if assessment.Confidence != ConfidenceHigh {
			continue
		}
		valid = append(valid, automationCandidate{Result: result, Assessment: assessment})
	}
	rankAutomationCandidates(valid, caps.Automation)
	if len(valid) > caps.Automation.MaxCandidates {
		valid = valid[:caps.Automation.MaxCandidates]
	}
	_ = store.AppendDecision(ctx, runID, DecisionStep{
		Stage: "filter", Status: "success", Summary: fmt.Sprintf("%d candidates remained for verification", len(valid)),
		Data: map[string]any{
			"rejected": rejected, "magnet_only": magnetOnly, "previously_bad": previouslyBad,
			"shortlisted": len(valid), "candidates": candidateAuditRows(valid),
		},
	})

	for index, candidate := range valid {
		if err = ctx.Err(); err != nil {
			background := context.Background()
			_ = store.AppendDecision(background, runID, DecisionStep{Stage: "decision", Status: "cancelled", Summary: "Automation run was cancelled"})
			_ = store.FinishRun(background, runID, RunCancelled, ReleaseAssessment{}, "")
			return true, err
		}
		verifyStarted := s.now()
		data, fetchErr := provider.FetchTorrent(ctx, candidate.Result.URL)
		if fetchErr != nil {
			_ = store.AppendDecision(ctx, runID, DecisionStep{
				Stage: "inspection", Status: "rejected", Summary: "Candidate metadata could not be fetched",
				Data: map[string]any{"rank": index + 1, "name": candidate.Result.Name, "reason": "metadata_fetch_failed"},
				DurationMS: elapsedMS(verifyStarted, s.now()),
			})
			continue
		}
		assessment, verifyErr := VerifyTorrentForTarget(candidate.Assessment, candidate.Result, data, &target)
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
		if verifyErr != nil || !assessment.EligibleForAutomaticDownload() {
			_ = store.AppendDecision(ctx, runID, DecisionStep{
				Stage: "inspection", Status: "rejected", Summary: "Candidate failed verified inspection",
				Data: map[string]any{
					"rank": index + 1, "name": candidate.Result.Name,
					"confidence": assessment.Confidence, "verification": assessment.Verification,
					"hard_rejections": assessment.HardRejections, "payload": assessment.Payload,
				}, DurationMS: elapsedMS(verifyStarted, s.now()),
			})
			continue
		}
		_ = store.AppendDecision(ctx, runID, DecisionStep{
			Stage: "inspection", Status: "success", Summary: "Candidate payload verified",
			Data: map[string]any{
				"rank": index + 1, "name": candidate.Result.Name, "infohash": assessment.InfoHash,
				"confidence": assessment.Confidence, "verification": assessment.Verification, "payload": assessment.Payload,
			}, DurationMS: elapsedMS(verifyStarted, s.now()),
		})

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
		downloadStarted := s.now()
		clientErr = client.AddTorrent(ctx, data)
		if clientErr != nil && assessment.InfoHash != "" {
			if clientSnapshot, reconcileErr := client.Downloads(ctx, TallyCategory); reconcileErr == nil && snapshotHasHash(clientSnapshot, assessment.InfoHash) {
				clientErr = nil
			}
		}
		if clientErr != nil {
			_ = store.AppendDecision(ctx, runID, DecisionStep{
				Stage: "download", Status: "failed", Summary: "qBittorrent submission failed",
				Data: map[string]any{"name": candidate.Result.Name, "infohash": assessment.InfoHash},
				DurationMS: elapsedMS(downloadStarted, s.now()),
			})
			_ = finish(RunFailed, assessment, candidate.Result.Name)
			return true, clientErr
		}
		_ = store.AppendDecision(ctx, runID, DecisionStep{
			Stage: "decision", Status: "selected", Summary: "Best verified candidate selected",
			Data: map[string]any{"name": candidate.Result.Name, "confidence": assessment.Confidence, "verification": assessment.Verification},
		})
		_ = store.AppendDecision(ctx, runID, DecisionStep{
			Stage: "download", Status: "success", Summary: "Sent to qBittorrent successfully",
			Data: map[string]any{"infohash": assessment.InfoHash}, DurationMS: elapsedMS(downloadStarted, s.now()),
		})
		if err = finish(RunDownloaded, assessment, candidate.Result.Name); err != nil {
			return true, err
		}
		s.publishChange()
		return true, nil
	}

	_ = store.AppendDecision(ctx, runID, DecisionStep{
		Stage: "decision", Status: "no_verified_candidate", Summary: "No verified high-confidence candidate was suitable for automatic download",
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
	rows, err := s.DB.QueryContext(ctx, `SELECT DISTINCT e.id,e.show_id,s.name,s.premiered,e.season,e.number,e.airstamp,COALESCE(p.policy,'default')
		FROM episodes e
		JOIN shows s ON s.id=e.show_id
		JOIN profile_shows f ON f.show_id=e.show_id
		LEFT JOIN torrent_show_policy p ON p.show_id=e.show_id
		WHERE e.airstamp<>'' AND e.number>0
		AND COALESCE(p.policy,'default') IN ('default','auto')
		AND NOT EXISTS(SELECT 1 FROM torrent_automation_runs r WHERE r.episode_id=e.id AND r.status IN ('running','downloaded'))
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
		if err = rows.Scan(&episode.ID, &episode.ShowID, &episode.ShowName, &episode.Premiered, &episode.Season, &episode.Episode, &episode.Airstamp, &episode.Policy); err != nil {
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
	rows, err := s.DB.QueryContext(ctx, `SELECT started_at FROM torrent_automation_runs
		WHERE episode_id=? AND status NOT IN ('running','downloaded') AND started_at>=?
		ORDER BY started_at DESC`, episodeID, now.Add(-time.Duration(retryWindowHours)*time.Hour).Unix())
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

func rankAutomationCandidates(items []automationCandidate, config settings.TorrentAutomation) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		leftQuality := automationQualityScore(left.Assessment.Parsed.Resolution, config.PreferredQuality)
		rightQuality := automationQualityScore(right.Assessment.Parsed.Resolution, config.PreferredQuality)
		if leftQuality != rightQuality {
			return leftQuality > rightQuality
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

func candidateAuditRows(items []automationCandidate) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for index, item := range items {
		rows = append(rows, map[string]any{
			"rank": index + 1, "name": item.Result.Name, "provider": item.Result.Provider,
			"seeders": item.Result.Seeders, "size": item.Result.Size,
			"confidence": item.Assessment.Confidence, "parsed": item.Assessment.Parsed,
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
