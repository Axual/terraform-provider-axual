package webclient

type EnvironmentsResponse struct {
	Embedded struct {
		Environments []struct {
			ShortName string `json:"shortName"`
			Uid       string `json:"uid"`
		} `json:"environments"`
	} `json:"_embedded"`
}

type EnvironmentResponse struct {
	Properties          map[string]interface{} `json:"properties"`
	Settings            map[string]interface{} `json:"settings"`
	Name                string                 `json:"name"`
	ShortName           string                 `json:"shortName"`
	Description         string                 `json:"description"`
	Color               string                 `json:"color"`
	AuthorizationIssuer string                 `json:"authorizationIssuer"`
	Visibility          string                 `json:"visibility"`
	RetentionTime       int                    `json:"retentionTime"`
	Partitions          int                    `json:"partitions"`
	Private             bool                   `json:"private"`
	AutoApproved        bool                   `json:"autoApproved"`
	Uid                 string                 `json:"uid"`

	Embedded struct {
		Instance struct {
			Name string `json:"name"`
			Uid  string `json:"uid"`
		} `json:"instance"`
		Owners struct {
			Name string `json:"name"`
			Uid  string `json:"uid"`
		} `json:"owners"`
		Viewers []struct {
			Uid string `json:"uid"`
		} `json:"viewers,omitempty"`
	} `json:"_embedded"`
}

type EnvironmentRequest struct {
	Name                string                 `json:"name,omitempty"`
	ShortName           string                 `json:"shortName,omitempty"`
	Description         interface{}            `json:"description,omitempty"`
	Color               string                 `json:"color,omitempty"`
	RetentionTime       int                    `json:"retentionTime,omitempty"`
	Partitions          int                    `json:"partitions,omitempty"`
	AuthorizationIssuer string                 `json:"authorizationIssuer,omitempty"`
	Visibility          string                 `json:"visibility,omitempty"`
	Instance            string                 `json:"instance,omitempty"`
	Owners              string                 `json:"owners,omitempty"`
	Viewers             []string               `json:"viewers"`
	Properties          map[string]interface{} `json:"properties,omitempty"`
	Settings            map[string]interface{} `json:"settings"`
}
