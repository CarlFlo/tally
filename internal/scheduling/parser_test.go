package scheduling

import (
	"testing"
	"time"
)

func TestParseAcceptsStandardFiveFieldForms(t *testing.T) {
	for _, spec := range []string{
		"0 3 * 3 *",
		"0 3 * JAN,MAR MON-FRI",
		"0,15,30,45 8-17/2 1,15 * 1-5",
		"*/20 */3 * */2 *",
		"*/60 * * * *",
	} {
		if _, err := Parse(spec); err != nil {
			t.Errorf("%q was rejected: %v", spec, err)
		}
	}
}

func TestParsePinsScheduleToUTC(t *testing.T) {
	parsed, err := Parse("0 3 * * *")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 1, 4, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	want := time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)
	if got := parsed.Next(now); !got.Equal(want) {
		t.Fatalf("next run %v, want %v", got, want)
	}
}
