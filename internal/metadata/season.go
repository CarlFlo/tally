package metadata

type Season struct {
	ID           int    `json:"id"`
	Number       int    `json:"number"`
	Name         string `json:"name"`
	EpisodeCount int    `json:"episodeOrder"`
	PremiereDate string `json:"premiereDate"`
	EndDate      string `json:"endDate"`
}
