package api

import (
	"time"

	"github.com/CarlFlo/mediaManager/internal/jobs"
)

func addSchedulePreviews(rows []map[string]any, timezone string) {
	for _, row := range rows {
		spec, _ := row["schedule"].(string)
		preview, err := jobs.PreviewSchedule(spec, time.Now(), timezone)
		if err != nil {
			continue
		}
		row["preview"] = preview
		row["description"] = preview.Description
		row["next_runs"] = preview.NextRuns
		row["timezone"] = preview.Timezone
	}
}
