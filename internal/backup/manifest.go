package backup

type Manifest struct {
	Format     int               `json:"format"`
	Schema     int               `json:"schema"`
	AppVersion string            `json:"app_version,omitempty"`
	Created    string            `json:"created"`
	Files      map[string]string `json:"files"`
}
