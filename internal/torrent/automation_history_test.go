package torrent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
)

func TestAutomationRunLifecycleIsImmutableAfterFinish(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := AutomationStore{DB: db}

	runID, err := store.StartRun(ctx, AutomationRun{
		ShowID:           "show-1",
		EpisodeID:        "episode-1",
		ShowName:         "Example Show",
		Season:           1,
		Episode:          2,
		Query:            "Example Show S01E02",
		SettingsSnapshot: json.RawMessage(`{"preferred_quality":"1080p","min_seeders":5}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.AppendDecision(ctx, runID, DecisionStep{Stage: "search", Summary: "12 candidates returned"}); err != nil {
		t.Fatal(err)
	}
	assessment := ReleaseAssessment{Confidence: ConfidenceHigh, Verification: VerificationVerified, InfoHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	if err = store.FinishRun(ctx, runID, RunDownloaded, assessment, "Example.Show.S01E02.1080p.WEB-DL-GROUP"); err != nil {
		t.Fatal(err)
	}
	if err = store.AppendDecision(ctx, runID, DecisionStep{Stage: "late", Summary: "must not change history"}); err == nil {
		t.Fatal("completed run accepted a new decision step")
	}
	if err = store.FinishRun(ctx, runID, RunFailed, ReleaseAssessment{}, ""); err == nil {
		t.Fatal("completed run was rewritten")
	}

	run, err := store.GetRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunDownloaded || run.Confidence != ConfidenceHigh || run.Verification != VerificationVerified || len(run.DecisionLog) != 1 {
		t.Fatalf("unexpected completed run: %+v", run)
	}
	if run.SelectedInfoHash != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || run.EngineVersion != DecisionEngineVersion {
		t.Fatalf("missing immutable decision identity: %+v", run)
	}
}

func TestMarkBadPreservesRunAndBlocksSelectedHash(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.ExecContext(ctx, "INSERT INTO profiles(id,display_name,avatar,created_at,locale,auth_method) VALUES('profile-a','Alex','mint',1,'en','none')"); err != nil {
		t.Fatal(err)
	}
	store := AutomationStore{DB: db}
	runID, err := store.StartRun(ctx, AutomationRun{ShowID: "show-1", EpisodeID: "episode-1", ShowName: "Example Show", Season: 1, Episode: 2, Query: "Example Show S01E02"})
	if err != nil {
		t.Fatal(err)
	}
	hash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err = store.FinishRun(ctx, runID, RunDownloaded, ReleaseAssessment{Confidence: ConfidenceHigh, Verification: VerificationVerified, InfoHash: hash}, "Example.Release"); err != nil {
		t.Fatal(err)
	}
	if err = store.MarkBad(ctx, runID, "profile-a", "wrong_episode", "release was mislabeled"); err != nil {
		t.Fatal(err)
	}
	if err = store.MarkBad(ctx, runID, "profile-a", "wrong_episode", "duplicate"); err == nil {
		t.Fatal("run accepted duplicate feedback")
	}
	blocked, err := store.IsBadInfoHash(ctx, hash)
	if err != nil || !blocked {
		t.Fatalf("selected bad hash was not blocked: %v %v", blocked, err)
	}
	run, err := store.GetRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunDownloaded || run.Feedback == nil || run.Feedback.Reason != "wrong_episode" {
		t.Fatalf("feedback did not remain separate from the original decision: %+v", run)
	}
}

func TestShowAutomationPolicyIsGlobalAndDefaultsCleanly(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.ExecContext(ctx, "INSERT INTO shows(id,name) VALUES('show-1','Example Show')"); err != nil {
		t.Fatal(err)
	}
	store := AutomationStore{DB: db}
	policy, err := store.ShowPolicy(ctx, "show-1")
	if err != nil || policy != "default" {
		t.Fatalf("unexpected default policy: %q %v", policy, err)
	}
	if err = store.SetShowPolicy(ctx, "show-1", "auto"); err != nil {
		t.Fatal(err)
	}
	policy, err = store.ShowPolicy(ctx, "show-1")
	if err != nil || policy != "auto" {
		t.Fatalf("global show policy not saved: %q %v", policy, err)
	}
	if err = store.SetShowPolicy(ctx, "show-1", "default"); err != nil {
		t.Fatal(err)
	}
	policy, err = store.ShowPolicy(ctx, "show-1")
	if err != nil || policy != "default" {
		t.Fatalf("default policy did not remove override: %q %v", policy, err)
	}
}
