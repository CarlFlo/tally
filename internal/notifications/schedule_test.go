package notifications

import (
	"testing"
	"time"
)

func TestDailyDeliveryRespectsTimezoneAndDST(t *testing.T) {
	zone, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ release, clock, want string }{
		{"2026-01-02T07:00:00Z", "09:00", "2026-01-02T08:00:00Z"},
		{"2026-01-02T09:00:00Z", "09:00", "2026-01-03T08:00:00Z"},
		{"2026-03-29T00:00:00Z", "02:30", "2026-03-29T01:00:00Z"},
		{"2026-10-25T00:00:00Z", "02:30", "2026-10-25T00:30:00Z"},
	} {
		at, _ := time.Parse(time.RFC3339, tc.release)
		if got := DeliveryAt(at, tc.clock, zone).UTC().Format(time.RFC3339); got != tc.want {
			t.Errorf("%s %s: got %s want %s", tc.release, tc.clock, got, tc.want)
		}
	}
}
