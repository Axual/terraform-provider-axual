package webclient

type ApplicationCredentialResponse struct {
	AuthData struct {
		Password string `json:"password"`
		Provider string `json:"provider"`
		Clusters string `json:"clusters"`
		Username string `json:"username"`
	} `json:"authData"`
}

type ApplicationCredentialResponseList []ApplicationCredentialResponse

type ApplicationCredentialCreateRequest struct {
	ApplicationId string `json:"applicationId"`
	EnvironmentId string `json:"environmentId"`
	Target        string `json:"target"`
}

type ApplicationCredentialDeleteRequest struct {
	ApplicationId string     `json:"applicationId"`
	EnvironmentId string     `json:"environmentId"`
	Target        string     `json:"target"`
	Configs       NameConfig `json:"configs"`
}

type NameConfig struct {
	Username string `json:"username"`
}

type ApplicationCredentialFindByApplicationAndEnvironmentResponseList []ApplicationCredentialFindByApplicationAndEnvironmentResponse
type ApplicationCredentialFindByApplicationAndEnvironmentResponse struct {
	ID          string          `json:"id"`
	Application ApplicationInfo `json:"application"`
	Environment EnvironmentInfo `json:"environment"`
	Username    string          `json:"username"`
	Types       []AuthType      `json:"types"`
	Description string          `json:"description"`
	Metadata    Metadata        `json:"metadata"`
}

type Metadata struct {
	Clusters string `json:"clusters"`
}

type ApplicationInfo struct {
	ID string `json:"id"`
}

type EnvironmentInfo struct {
	ID string `json:"id"`
}

type AuthType struct {
	Type string `json:"type"`
}
