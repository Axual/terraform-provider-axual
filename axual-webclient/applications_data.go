package webclient

type ApplicationResponse struct {
	Name             string `json:"name"`
	ShortName        string `json:"shortName"`
	Description      string `json:"description"`
	ApplicationType  string `json:"applicationType"`
	Type             string `json:"type"`
	ApplicationClass string `json:"applicationClass"`
	Visibility       string `json:"visibility"`
	Owners           struct {
		Name string `json:"name"`
		Uid  string `json:"uid"`
	} `json:"owners"`
	Embedded struct {
		Viewers []struct {
			Name string `json:"name"`
			Uid  string `json:"uid"`
		} `json:"viewers,omitempty"`
	} `json:"_embedded"`
	Uid           string `json:"uid"`
	ApplicationId string `json:"applicationId"`
}

type ApplicationRequest struct {
	ApplicationType  string   `json:"applicationType"`
	ApplicationId    string   `json:"applicationId"`
	Name             string   `json:"name"`
	ShortName        string   `json:"shortName"`
	Owners           string   `json:"owners"`
	Viewers          []string `json:"viewers"`
	Type             string   `json:"type,omitempty"`
	ApplicationClass string   `json:"applicationClass,omitempty"`
	Visibility       string   `json:"visibility"`
	Description      string   `json:"description"`
}
