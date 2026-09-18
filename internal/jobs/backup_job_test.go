package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
)

type recordingBackup struct {
	kinds []string
}

func (b *recordingBackup) Create(_ context.Context, kind string) (string, error) {
	b.kinds = append(b.kinds, kind)
	return kind + ".zip", nil
}

func TestBackupKindFollowsInvocationSource(t *testing.T) {
	tests := []struct {
		trigger string
		want    string
	}{
		{trigger: "manual_backup", want: "manual"},
		{trigger: "manual_refresh", want: "auto"},
		{trigger: "scheduled_refresh", want: "auto"},
	}
	for _, test := range tests {
		t.Run(test.trigger, func(t *testing.T) {
			recorder := &recordingBackup{}
			service := &Service{Backup: recorder}
			if _, err := service.runBackup(context.Background(), test.trigger); err != nil {
				t.Fatal(err)
			}
			if len(recorder.kinds) != 1 || recorder.kinds[0] != test.want {
				t.Fatalf("backup kind = %v, want %q", recorder.kinds, test.want)
			}
		})
	}
}

func TestSuccessfulBackupPublishesBackupChange(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := New(
		ctx,
		db,
		config.Config{JobConcurrency: 1, JobRuntime: time.Second},
		nil,
		nil,
		&recordingBackup{},
	)
	defer service.Stop(context.Background())

	changes := make(chan []string, 8)
	service.OnChange = func(_ string, resources ...string) {
		copied := append([]string(nil), resources...)
		select {
		case changes <- copied:
		default:
		}
	}

	if _, err = service.Trigger("backup", "manual_backup", ""); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case resources := <-changes:
			for _, resource := range resources {
				if resource == "backups" {
					return
				}
			}
		case <-deadline:
			t.Fatal("successful backup did not publish a backups change")
		}
	}
}

func TestTriggerAndWaitReturnsAfterBackupCompletion(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	recorder := &recordingBackup{}
	service := New(ctx, db, config.Config{JobConcurrency: 1, JobRuntime: time.Second}, nil, nil, recorder)
	defer service.Stop(context.Background())

	id, err := service.TriggerAndWait(ctx, "backup", "manual_backup", "")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" || len(recorder.kinds) != 1 || recorder.kinds[0] != "manual" {
		t.Fatalf("backup did not finish before return: id=%q kinds=%v", id, recorder.kinds)
	}
	var status string
	if err = db.QueryRow("SELECT status FROM job_runs WHERE id=?", id).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "success" {
		t.Fatalf("job status=%q, want success", status)
	}
}
