package notifications

import "time"

// DeliveryAt uses the selected local calendar day, including DST transitions.
// During a skipped wall-clock hour, deliver at the first existing minute after it.
// During a repeated hour, choose the first occurrence.
func DeliveryAt(released time.Time, clock string, location *time.Location) time.Time {
	parsed, _ := time.Parse("15:04", clock)
	local := released.In(location)
	for day := 0; day < 3; day++ {
		date := time.Date(local.Year(), local.Month(), local.Day()+day, 0, 0, 0, 0, location)
		end := date.AddDate(0, 0, 1)
		for candidate := date; candidate.Before(end); candidate = candidate.Add(time.Minute) {
			t := candidate.In(location)
			if t.Hour()*60+t.Minute() >= parsed.Hour()*60+parsed.Minute() {
				if !candidate.Before(released) {
					return candidate
				}
				break
			}
		}
	}
	return released.Add(24 * time.Hour)
}
