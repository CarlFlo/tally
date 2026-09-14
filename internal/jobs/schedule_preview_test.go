package jobs

import (
	"strings"
	"testing"
	"time"

	"github.com/CarlFlo/mediaManager/internal/scheduling"
)

func TestPreviewScheduleUsesSchedulerDialect(t *testing.T) {
	now := time.Date(2026, time.December, 31, 4, 0, 0, 0, time.UTC)
	preview, err := PreviewSchedule("0 3 1 1 1", now)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Description == "" || preview.Timezone != "UTC" || len(preview.NextRuns) != 3 {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	first := time.Unix(preview.NextRuns[0], 0).UTC()
	if first.Before(now) || first.Hour() != 3 || first.Minute() != 0 {
		t.Fatalf("preview did not use scheduler next occurrence: %v", first)
	}
}

func TestScheduleSemanticsAndUTCPreviews(t *testing.T) {
	cases := []struct {
		name, spec string
		now, want  time.Time
	}{
		{"month", "0 3 * 3 *", time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 1, 3, 0, 0, 0, time.UTC)},
		{"weekday", "0 3 * * 1", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 5, 3, 0, 0, 0, time.UTC)},
		{"day or weekday", "0 3 1 1 1", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)},
		{"ranges lists and steps", "*/15 8-9 * 1,3 1-5", time.Date(2026, 1, 1, 7, 59, 0, 0, time.UTC), time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)},
		{"sixty minute compatibility", "*/60 * * * *", time.Date(2026, 1, 1, 0, 15, 0, 0, time.FixedZone("test", 2*60*60)), time.Date(2025, 12, 31, 23, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			preview, err := PreviewSchedule(tc.spec, tc.now)
			if err != nil {
				t.Fatal(err)
			}
			got := time.Unix(preview.NextRuns[0], 0).UTC()
			if !got.Equal(tc.want) {
				t.Fatalf("first run %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseScheduleRequiresFiveFields(t *testing.T) {
	if _, err := scheduling.Parse("@daily"); err == nil || !strings.Contains(err.Error(), "five-field") {
		t.Fatalf("expected five-field error, got %v", err)
	}
	if _, err := scheduling.Parse("0 3 * 1 1"); err != nil {
		t.Fatalf("valid schedule rejected: %v", err)
	}
	if _, err := scheduling.Parse("60 * * * *"); err == nil || !strings.Contains(err.Error(), "maximum (59)") {
		t.Fatalf("expected clear minute validation error, got %v", err)
	}
}
