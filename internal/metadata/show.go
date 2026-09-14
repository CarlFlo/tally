package metadata

type Show struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Summary   string   `json:"summary"`
	Status    string   `json:"status"`
	Premiered string   `json:"premiered"`
	Genres    []string `json:"genres"`
	Runtime   int      `json:"runtime"`
	Updated   int64    `json:"updated"`
	Image     struct {
		Medium string `json:"medium"`
	} `json:"image"`
	Rating struct {
		Average float64 `json:"average"`
	} `json:"rating"`
	Network *struct {
		Name string `json:"name"`
	} `json:"network"`
	WebChannel *struct {
		Name string `json:"name"`
	} `json:"webChannel"`
	Externals struct {
		IMDB string `json:"imdb"`
		TVDB int    `json:"thetvdb"`
	} `json:"externals"`
}

type SearchResult struct {
	Score float64 `json:"score"`
	Show  Show    `json:"show"`
}
