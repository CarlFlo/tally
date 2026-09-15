package jobs

import (
	"context"
	"testing"
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
