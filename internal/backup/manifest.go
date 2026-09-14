package backup

type Manifest struct {
	Format  int               `json:"format"`
	Schema  int               `json:"schema"`
	Created string            `json:"created"`
	Files   map[string]string `json:"files"`
}
