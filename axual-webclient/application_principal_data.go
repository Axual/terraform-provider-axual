package webclient

type ApplicationPrincipalCreateResponse string
type ApplicationPrincipalUpdateResponse interface{}

type ApplicationPrincipalResponse struct {
	Uid      string `json:"uid"`
	Principal          string      `json:"principal"`
	ApplicationPem     string      `json:"applicationPem"`
	Type               string      `json:"type"`
	Embedded struct {
		Application struct {
			ShortName string `json:"shortName"`
			Uid       string `json:"uid"`
		} `json:"application"`
		Environment struct {
			ShortName string `json:"shortName"`
			Uid       string `json:"uid"`
		} `json:"environment"`
	} `json:"_embedded"`
}

type ApplicationPrincipalRequest struct {
	Principal   string `json:"principal"`
	PrivateKey  string `json:"privateKey,omitempty"`
	Application string `json:"application"`
	Environment string `json:"environment"`
	Custom      bool   `json:"custom,omitempty"`
}
type ApplicationPrincipalUpdateRequest struct {
	Principal string `json:"principal"`
}

type ApplicationPrincipalFindByApplicationAndEnvironmentResponse struct {
	Embedded struct {
		ApplicationPrincipalResponses []interface{} `json:"application_principals"`
	} `json:"_embedded"`
}
