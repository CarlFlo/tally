package torrent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/CarlFlo/tally/internal/database"
)

const DecisionEngineVersion = "4"

type AutomationRunStatus string

const (
	RunRunning             AutomationRunStatus = "running"
	RunDownloaded          AutomationRunStatus = "downloaded"
	RunNoVerifiedCandidate AutomationRunStatus = "no_verified_candidate"
	RunRejected            AutomationRunStatus = "rejected"
	RunFailed              AutomationRunStatus = "failed"
	RunSkipped             AutomationRunStatus = "skipped"
	RunCancelled           AutomationRunStatus = "cancelled"
)

type DecisionStep struct {
	Stage      string         `json:"stage"`
	Status     string         `json:"status,omitempty"`
	Summary    string         `json:"summary,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
	OccurredAt int64          `json:"occurred_at"`
	DurationMS int64          `json:"duration_ms,omitempty"`
}

type MagnetVerification struct {
	RunID         string                 `json:"run_id"`
	InfoHash      string                 `json:"infohash"`
	Status        string                 `json:"status"`
	Attempts      int                    `json:"attempts"`
	LastCheckedAt int64                  `json:"last_checked_at,omitempty"`
	CompletedAt   *int64                 `json:"completed_at,omitempty"`
	Assessment    *ReleaseAssessment     `json:"assessment,omitempty"`
	SizeProfile   *SizeProfileEvaluation `json:"size_profile,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

type PendingMagnetVerification struct {
	RunID            string
	ShowID           string
	EpisodeID        string
	ShowName         string
	Season           int
	Episode          int
	SelectedName     string
	InfoHash         string
	SettingsSnapshot json.RawMessage
	Attempts         int
	StartedAt        int64
}

type AutomationRun struct {
	ID               string              `json:"id"`
	ShowID           string              `json:"show_id"`
	EpisodeID        string              `json:"episode_id"`
	ShowName         string              `json:"show_name"`
	Season           int                 `json:"season"`
	Episode          int                 `json:"episode"`
	Query            string              `json:"query"`
	Status           AutomationRunStatus `json:"status"`
	Confidence       Confidence          `json:"confidence,omitempty"`
	Verification     VerificationState   `json:"verification,omitempty"`
	SelectedName     string              `json:"selected_name,omitempty"`
	SelectedInfoHash string              `json:"selected_infohash,omitempty"`
	SettingsSnapshot json.RawMessage     `json:"settings_snapshot"`
	DecisionLog      []DecisionStep      `json:"decision_log"`
	EngineVersion    string              `json:"engine_version"`
	StartedAt        int64               `json:"started_at"`
	EndedAt          *int64              `json:"ended_at,omitempty"`
	DurationMS       int64               `json:"duration_ms"`
	Feedback         *AutomationFeedback `json:"feedback,omitempty"`
	PostVerification *MagnetVerification `json:"post_verification,omitempty"`
}

type AutomationFeedback struct {
	ProfileID string `json:"profile_id,omitempty"`
	Reason    string `json:"reason"`
	Note      string `json:"note,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

type AutomationStore struct{ DB *database.Store }

const (
	MediaProfileAuto     = "auto"
	MediaProfileLive     = "live"
	MediaProfileAnimated = "animated"
)

type ShowMediaProfile struct {
	Mode      string `json:"mode"`
	Detected  string `json:"detected"`
	Effective string `json:"effective"`
}

func detectShowMediaProfile(showType, genresJSON string) string {
	animated := func(value string) bool {
		value = strings.ToLower(strings.TrimSpace(value))
		return strings.Contains(value, "animation") || strings.Contains(value, "anime") || strings.Contains(value, "cartoon")
	}
	if animated(showType) {
		return MediaProfileAnimated
	}
	var genres []string
	if json.Unmarshal([]byte(genresJSON), &genres) == nil {
		for _, genre := range genres {
			if animated(genre) {
				return MediaProfileAnimated
			}
		}
	}
	return MediaProfileLive
}

func (s AutomationStore) ShowMediaProfile(ctx context.Context, showID string) (ShowMediaProfile, error) {
	var showType, genres, override string
	err := s.DB.QueryRowContext(ctx, `SELECT s.show_type,s.genres,COALESCE(p.profile,'')
		FROM shows s LEFT JOIN torrent_show_media_profile p ON p.show_id=s.id WHERE s.id=?`, showID).Scan(&showType, &genres, &override)
	if err != nil {
		return ShowMediaProfile{}, err
	}
	detected := detectShowMediaProfile(showType, genres)
	mode := MediaProfileAuto
	effective := detected
	if override == MediaProfileLive || override == MediaProfileAnimated {
		mode = override
		effective = override
	}
	return ShowMediaProfile{Mode: mode, Detected: detected, Effective: effective}, nil
}

func (s AutomationStore) SetShowMediaProfile(ctx context.Context, showID, mode string) error {
	switch mode {
	case MediaProfileAuto:
		_, err := s.DB.ExecContext(ctx, "DELETE FROM torrent_show_media_profile WHERE show_id=?", showID)
		return err
	case MediaProfileLive, MediaProfileAnimated:
	default:
		return fmt.Errorf("invalid show media profile")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO torrent_show_media_profile(show_id,profile,updated_at) VALUES(?,?,?)
		ON CONFLICT(show_id) DO UPDATE SET profile=excluded.profile,updated_at=excluded.updated_at`, showID, mode, time.Now().Unix())
	return err
}

func (s AutomationStore) QueueMagnetVerification(ctx context.Context, runID, infohash string) error {
	infohash = normalizeInfoHash(infohash)
	if runID == "" || infohash == "" {
		return fmt.Errorf("magnet verification requires a run and infohash")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO torrent_magnet_verifications(run_id,infohash,status)
		VALUES(?,?,'pending') ON CONFLICT(run_id) DO NOTHING`, runID, infohash)
	return err
}

func (s AutomationStore) PendingMagnetVerifications(ctx context.Context, limit int) ([]PendingMagnetVerification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT r.id,r.show_id,r.episode_id,r.show_name,r.season,r.episode,r.selected_name,
		m.infohash,r.settings_snapshot,m.attempts,r.started_at
		FROM torrent_magnet_verifications m
		JOIN torrent_automation_runs r ON r.id=m.run_id
		WHERE m.status='pending'
		ORDER BY r.started_at ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PendingMagnetVerification, 0)
	for rows.Next() {
		var item PendingMagnetVerification
		var snapshot string
		if err = rows.Scan(&item.RunID, &item.ShowID, &item.EpisodeID, &item.ShowName, &item.Season, &item.Episode,
			&item.SelectedName, &item.InfoHash, &snapshot, &item.Attempts, &item.StartedAt); err != nil {
			return nil, err
		}
		item.SettingsSnapshot = json.RawMessage(snapshot)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s AutomationStore) RecordMagnetVerificationAttempt(ctx context.Context, runID, message string) error {
	if len(message) > 500 {
		message = message[:500]
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE torrent_magnet_verifications
		SET attempts=attempts+1,last_checked_at=?,error=?
		WHERE run_id=? AND status='pending'`, time.Now().Unix(), message, runID)
	return err
}

func (s AutomationStore) FinishMagnetVerification(ctx context.Context, runID, status string, assessment ReleaseAssessment, sizeProfile SizeProfileEvaluation, message string) error {
	if status != "verified" && status != "rejected" && status != "unavailable" {
		return fmt.Errorf("invalid magnet verification status")
	}
	if len(message) > 500 {
		message = message[:500]
	}
	assessmentRaw, err := json.Marshal(assessment)
	if err != nil {
		return err
	}
	sizeRaw, err := json.Marshal(sizeProfile)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	res, err := s.DB.ExecContext(ctx, `UPDATE torrent_magnet_verifications
		SET status=?,attempts=attempts+1,last_checked_at=?,completed_at=?,assessment=?,size_profile=?,error=?
		WHERE run_id=? AND status='pending'`, status, now, now, string(assessmentRaw), string(sizeRaw), message, runID)
	if err != nil {
		return err
	}
	changed, _ := res.RowsAffected()
	if changed != 1 {
		return fmt.Errorf("magnet verification is not pending")
	}
	return nil
}

func (s AutomationStore) RejectMagnetVerification(ctx context.Context, runID string, assessment ReleaseAssessment, sizeProfile SizeProfileEvaluation, message string) error {
	if len(message) > 500 {
		message = message[:500]
	}
	assessmentRaw, err := json.Marshal(assessment)
	if err != nil {
		return err
	}
	sizeRaw, err := json.Marshal(sizeProfile)
	if err != nil {
		return err
	}
	infohash := normalizeInfoHash(assessment.InfoHash)
	if infohash == "" {
		return fmt.Errorf("rejected magnet verification has no valid infohash")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `UPDATE torrent_magnet_verifications
		SET status='rejected',attempts=attempts+1,last_checked_at=?,completed_at=?,assessment=?,size_profile=?,error=?
		WHERE run_id=? AND status='pending'`, now, now, string(assessmentRaw), string(sizeRaw), message, runID)
	if err != nil {
		return err
	}
	changed, _ := res.RowsAffected()
	if changed != 1 {
		return fmt.Errorf("magnet verification is not pending")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO torrent_bad_hashes(infohash,reason,source_run_id,marked_by,created_at)
		VALUES(?,'post_magnet_verification',?,NULL,?) ON CONFLICT(infohash) DO NOTHING`, infohash, runID, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s AutomationStore) BlockInfoHash(ctx context.Context, infohash, reason, sourceRunID string) error {
	infohash = normalizeInfoHash(infohash)
	if infohash == "" {
		return fmt.Errorf("invalid infohash")
	}
	if len(reason) > 200 {
		reason = reason[:200]
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO torrent_bad_hashes(infohash,reason,source_run_id,marked_by,created_at)
		VALUES(?,?,?,NULL,?) ON CONFLICT(infohash) DO NOTHING`, infohash, reason, sourceRunID, time.Now().Unix())
	return err
}

func scanMagnetVerification(
	infohash, status sql.NullString,
	attempts sql.NullInt64,
	lastChecked, completed sql.NullInt64,
	assessmentRaw, sizeRaw, verificationError sql.NullString,
) (*MagnetVerification, error) {
	if !status.Valid {
		return nil, nil
	}
	out := &MagnetVerification{
		InfoHash: infohash.String, Status: status.String, Attempts: int(attempts.Int64),
		LastCheckedAt: lastChecked.Int64, Error: verificationError.String,
	}
	if completed.Valid {
		value := completed.Int64
		out.CompletedAt = &value
	}
	if assessmentRaw.Valid && assessmentRaw.String != "" && assessmentRaw.String != "{}" {
		var assessment ReleaseAssessment
		if err := json.Unmarshal([]byte(assessmentRaw.String), &assessment); err != nil {
			return nil, fmt.Errorf("stored magnet verification assessment is invalid: %w", err)
		}
		out.Assessment = &assessment
	}
	if sizeRaw.Valid && sizeRaw.String != "" && sizeRaw.String != "{}" {
		var sizeProfile SizeProfileEvaluation
		if err := json.Unmarshal([]byte(sizeRaw.String), &sizeProfile); err != nil {
			return nil, fmt.Errorf("stored magnet verification size profile is invalid: %w", err)
		}
		out.SizeProfile = &sizeProfile
	}
	return out, nil
}

func (s AutomationStore) StartRun(ctx context.Context, run AutomationRun) (string, error) {
	if run.ID == "" {
		run.ID = database.ID()
	}
	if run.EngineVersion == "" {
		run.EngineVersion = DecisionEngineVersion
	}
	if run.StartedAt == 0 {
		run.StartedAt = time.Now().Unix()
	}
	if run.Status == "" {
		run.Status = RunRunning
	}
	if run.Status != RunRunning || run.ShowID == "" || run.EpisodeID == "" || run.ShowName == "" || run.Query == "" {
		return "", fmt.Errorf("automation run is missing required identity")
	}
	settingsSnapshot := run.SettingsSnapshot
	if len(settingsSnapshot) == 0 {
		settingsSnapshot = json.RawMessage(`{}`)
	}
	if !json.Valid(settingsSnapshot) {
		return "", fmt.Errorf("automation settings snapshot is invalid JSON")
	}
	decisionLog, err := json.Marshal(run.DecisionLog)
	if err != nil {
		return "", err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO torrent_automation_runs(
		id,show_id,episode_id,show_name,season,episode,query,status,settings_snapshot,decision_log,engine_version,started_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, run.ID, run.ShowID, run.EpisodeID, run.ShowName, run.Season, run.Episode, run.Query, run.Status, string(settingsSnapshot), string(decisionLog), run.EngineVersion, run.StartedAt)
	return run.ID, err
}

func (s AutomationStore) AppendDecision(ctx context.Context, runID string, step DecisionStep) error {
	if step.Stage == "" {
		return fmt.Errorf("decision step stage is required")
	}
	if step.OccurredAt == 0 {
		step.OccurredAt = time.Now().Unix()
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var raw string
	if err = tx.QueryRowContext(ctx, "SELECT decision_log FROM torrent_automation_runs WHERE id=? AND status='running'", runID).Scan(&raw); err != nil {
		return err
	}
	var steps []DecisionStep
	if err = json.Unmarshal([]byte(raw), &steps); err != nil {
		return err
	}
	steps = append(steps, step)
	updated, err := json.Marshal(steps)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE torrent_automation_runs SET decision_log=? WHERE id=? AND status='running'", string(updated), runID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s AutomationStore) FinishMagnetRun(ctx context.Context, runID string, assessment ReleaseAssessment, selectedName string) error {
	infohash := normalizeInfoHash(assessment.InfoHash)
	if infohash == "" {
		return fmt.Errorf("magnet run has no valid infohash")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ended := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `UPDATE torrent_automation_runs
		SET status=?,confidence=?,verification=?,selected_name=?,selected_infohash=?,ended_at=?,duration_ms=MAX(0,(?-started_at)*1000)
		WHERE id=? AND status='running'`, RunDownloaded, assessment.Confidence, assessment.Verification, selectedName, infohash, ended, ended, runID)
	if err != nil {
		return err
	}
	changed, _ := res.RowsAffected()
	if changed != 1 {
		return fmt.Errorf("automation run is not running")
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO torrent_magnet_verifications(run_id,infohash,status)
		VALUES(?,?,'pending')`, runID, infohash); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO torrent_episode_downloads(infohash,episode_id,created_at)
		SELECT ?,r.episode_id,? FROM torrent_automation_runs r JOIN episodes e ON e.id=r.episode_id WHERE r.id=?
		ON CONFLICT(infohash) DO UPDATE SET episode_id=excluded.episode_id`, infohash, ended, runID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s AutomationStore) FinishRun(ctx context.Context, runID string, status AutomationRunStatus, assessment ReleaseAssessment, selectedName string) error {
	if status == RunRunning || !validTerminalRunStatus(status) {
		return fmt.Errorf("invalid terminal automation status")
	}
	ended := time.Now().Unix()
	infohash := strings.ToLower(strings.TrimSpace(assessment.InfoHash))
	if status == RunDownloaded && normalizeInfoHash(infohash) == "" {
		return fmt.Errorf("downloaded automation run has no valid infohash")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE torrent_automation_runs
		SET status=?,confidence=?,verification=?,selected_name=?,selected_infohash=?,ended_at=?,duration_ms=MAX(0,(?-started_at)*1000)
		WHERE id=? AND status='running'`, status, assessment.Confidence, assessment.Verification, selectedName, infohash, ended, ended, runID)
	if err != nil {
		return err
	}
	changed, _ := res.RowsAffected()
	if changed != 1 {
		return fmt.Errorf("automation run is not running")
	}
	if status == RunDownloaded {
		if _, err = tx.ExecContext(ctx, `INSERT INTO torrent_episode_downloads(infohash,episode_id,created_at)
			SELECT ?,r.episode_id,? FROM torrent_automation_runs r JOIN episodes e ON e.id=r.episode_id WHERE r.id=?
			ON CONFLICT(infohash) DO UPDATE SET episode_id=excluded.episode_id`, infohash, ended, runID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func validTerminalRunStatus(status AutomationRunStatus) bool {
	switch status {
	case RunDownloaded, RunNoVerifiedCandidate, RunRejected, RunFailed, RunSkipped, RunCancelled:
		return true
	default:
		return false
	}
}

func (s AutomationStore) ListRuns(ctx context.Context, limit int) ([]AutomationRun, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT r.id,r.show_id,r.episode_id,r.show_name,r.season,r.episode,r.query,r.status,
		r.confidence,r.verification,r.selected_name,r.selected_infohash,r.settings_snapshot,r.decision_log,r.engine_version,
		r.started_at,r.ended_at,r.duration_ms,f.profile_id,f.reason,f.note,f.created_at,
		m.infohash,m.status,m.attempts,m.last_checked_at,m.completed_at,m.assessment,m.size_profile,m.error
		FROM torrent_automation_runs r
		LEFT JOIN torrent_automation_feedback f ON f.run_id=r.id
		LEFT JOIN torrent_magnet_verifications m ON m.run_id=r.id
		ORDER BY r.started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AutomationRun, 0)
	for rows.Next() {
		run, err := scanAutomationRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

func (s AutomationStore) GetRun(ctx context.Context, id string) (AutomationRun, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT r.id,r.show_id,r.episode_id,r.show_name,r.season,r.episode,r.query,r.status,
		r.confidence,r.verification,r.selected_name,r.selected_infohash,r.settings_snapshot,r.decision_log,r.engine_version,
		r.started_at,r.ended_at,r.duration_ms,f.profile_id,f.reason,f.note,f.created_at,
		m.infohash,m.status,m.attempts,m.last_checked_at,m.completed_at,m.assessment,m.size_profile,m.error
		FROM torrent_automation_runs r
		LEFT JOIN torrent_automation_feedback f ON f.run_id=r.id
		LEFT JOIN torrent_magnet_verifications m ON m.run_id=r.id WHERE r.id=?`, id)
	return scanAutomationRun(row)
}

type automationRunScanner interface{ Scan(...any) error }

func scanAutomationRun(scanner automationRunScanner) (AutomationRun, error) {
	var run AutomationRun
	var settingsRaw, decisionsRaw string
	var ended sql.NullInt64
	var feedbackProfile, feedbackReason, feedbackNote sql.NullString
	var feedbackCreated sql.NullInt64
	var magnetInfoHash, magnetStatus, magnetAssessment, magnetSize, magnetError sql.NullString
	var magnetAttempts, magnetLastChecked, magnetCompleted sql.NullInt64
	if err := scanner.Scan(&run.ID, &run.ShowID, &run.EpisodeID, &run.ShowName, &run.Season, &run.Episode, &run.Query, &run.Status,
		&run.Confidence, &run.Verification, &run.SelectedName, &run.SelectedInfoHash, &settingsRaw, &decisionsRaw, &run.EngineVersion,
		&run.StartedAt, &ended, &run.DurationMS, &feedbackProfile, &feedbackReason, &feedbackNote, &feedbackCreated,
		&magnetInfoHash, &magnetStatus, &magnetAttempts, &magnetLastChecked, &magnetCompleted, &magnetAssessment, &magnetSize, &magnetError); err != nil {
		return AutomationRun{}, err
	}
	if ended.Valid {
		run.EndedAt = &ended.Int64
	}
	run.SettingsSnapshot = json.RawMessage(settingsRaw)
	if !json.Valid(run.SettingsSnapshot) {
		return AutomationRun{}, fmt.Errorf("stored automation settings snapshot is invalid")
	}
	if err := json.Unmarshal([]byte(decisionsRaw), &run.DecisionLog); err != nil {
		return AutomationRun{}, fmt.Errorf("stored automation decision log is invalid: %w", err)
	}
	if feedbackReason.Valid {
		run.Feedback = &AutomationFeedback{ProfileID: feedbackProfile.String, Reason: feedbackReason.String, Note: feedbackNote.String, CreatedAt: feedbackCreated.Int64}
	}
	verification, err := scanMagnetVerification(magnetInfoHash, magnetStatus, magnetAttempts, magnetLastChecked, magnetCompleted, magnetAssessment, magnetSize, magnetError)
	if err != nil {
		return AutomationRun{}, err
	}
	if verification != nil {
		verification.RunID = run.ID
		run.PostVerification = verification
	}
	return run, nil
}

var hexInfoHash = regexp.MustCompile(`^(?:[a-f0-9]{40}|[a-f0-9]{64})$`)

func normalizeInfoHash(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if !hexInfoHash.MatchString(value) {
		return ""
	}
	return value
}

func (s AutomationStore) MarkBad(ctx context.Context, runID, profileID, reason, note string) error {
	if !validFeedbackReason(reason) {
		return fmt.Errorf("invalid bad-run reason")
	}
	if len(note) > 500 {
		return fmt.Errorf("bad-run note is too long")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var infohash string
	var status AutomationRunStatus
	if err = tx.QueryRowContext(ctx, "SELECT selected_infohash,status FROM torrent_automation_runs WHERE id=?", runID).Scan(&infohash, &status); err != nil {
		return err
	}
	if status == RunRunning {
		return fmt.Errorf("a running automation attempt cannot be marked bad")
	}
	created := time.Now().Unix()
	if _, err = tx.ExecContext(ctx, "INSERT INTO torrent_automation_feedback(run_id,profile_id,reason,note,created_at) VALUES(?,?,?,?,?)", runID, profileID, reason, note, created); err != nil {
		return err
	}
	if normalized := normalizeInfoHash(infohash); normalized != "" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO torrent_bad_hashes(infohash,reason,source_run_id,marked_by,created_at)
			VALUES(?,?,?,?,?) ON CONFLICT(infohash) DO NOTHING`, normalized, reason, runID, profileID, created); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func validFeedbackReason(reason string) bool {
	switch reason {
	case "wrong_show", "wrong_episode", "wrong_language", "poor_quality", "corrupt", "suspicious_files", "other":
		return true
	default:
		return false
	}
}

func (s AutomationStore) IsBadInfoHash(ctx context.Context, infohash string) (bool, error) {
	infohash = normalizeInfoHash(infohash)
	if infohash == "" {
		return false, nil
	}
	var exists int
	err := s.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM torrent_bad_hashes WHERE infohash=?)", infohash).Scan(&exists)
	return exists == 1, err
}

func (s AutomationStore) ShowPolicy(ctx context.Context, showID string) (string, error) {
	var policy string
	err := s.DB.QueryRowContext(ctx, "SELECT policy FROM torrent_show_policy WHERE show_id=?", showID).Scan(&policy)
	if errors.Is(err, sql.ErrNoRows) {
		return "default", nil
	}
	return policy, err
}

func (s AutomationStore) SetShowPolicy(ctx context.Context, showID, policy string) error {
	switch policy {
	case "default", "auto", "never":
	default:
		return fmt.Errorf("invalid show automation policy")
	}
	if policy == "default" {
		_, err := s.DB.ExecContext(ctx, "DELETE FROM torrent_show_policy WHERE show_id=?", showID)
		return err
	}
	res, err := s.DB.ExecContext(ctx, `INSERT INTO torrent_show_policy(show_id,policy,updated_at) VALUES(?,?,?)
		ON CONFLICT(show_id) DO UPDATE SET policy=excluded.policy,updated_at=excluded.updated_at`, showID, policy, time.Now().Unix())
	if err != nil {
		return err
	}
	changed, _ := res.RowsAffected()
	if changed == 0 {
		return fmt.Errorf("show automation policy was not saved")
	}
	return nil
}

func (s AutomationStore) PruneRuns(ctx context.Context, before time.Time) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM torrent_automation_runs WHERE started_at<? AND status<>'running'", before.Unix())
	return err
}
