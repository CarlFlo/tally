package metadata

type Episode struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Season   int    `json:"season"`
	Number   int    `json:"number"`
	Summary  string `json:"summary"`
	Airdate  string `json:"airdate"`
	Airstamp string `json:"airstamp"`
	Runtime  int    `json:"runtime"`
	Type     string `json:"type"`
}
