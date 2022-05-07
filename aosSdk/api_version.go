package aosSdk

const (
	apiUrlVersion = "/api/version"
)

type VersionResponse struct {
	Major   string `json:"major"`
	Version string `json:"version"`
	Build   string `json:"build"`
	Minor   string `json:"minor"`
}

func (o Client) getVersion() (*VersionResponse, error) {
	var response VersionResponse
	return &response, o.talkToAos(&talkToAosIn{
		method:        httpMethodGet,
		url:           apiUrlVersion,
		fromServerPtr: &response,
	})
}