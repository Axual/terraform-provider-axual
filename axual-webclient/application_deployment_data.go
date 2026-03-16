package webclient

type ApplicationDeploymentCreateResponse string

type ApplicationDeploymentUpdateResponse interface{}
type Config struct {
	ConfigKey   string `json:"configKey"`
	ConfigValue string `json:"configValue"`
}
type ApplicationDeploymentResponse struct {
	Configs  []Config `json:"configs"`
	State    string   `json:"state"`
	Uid      string   `json:"uid"`
	Embedded struct {
		Application struct {
			ShortName       string `json:"shortName"`
			ApplicationType string `json:"applicationType"`
			Uid             string `json:"uid"`
		} `json:"application"`
		Environment struct {
			ShortName string `json:"shortName"`
			Uid       string `json:"uid"`
		} `json:"environment"`
	} `json:"_embedded"`
}

type ApplicationDeploymentCreateRequest struct {
	Application string            `json:"application"`
	Environment string            `json:"environment"`
	Configs     map[string]string `json:"configs"`
}

type ApplicationDeploymentUpdateRequest struct {
	Configs map[string]string `json:"configs"`
}

type ApplicationDeploymentOperationRequest struct {
	Action string `json:"action"`
}

type ApplicationDeploymentFindByApplicationAndEnvironmentResponse struct {
	Embedded struct {
		ApplicationDeploymentResponses []ApplicationDeploymentResponse `json:"application_deployments"`
	} `json:"_embedded"`
}

type ApplicationDeploymentStatusResponse struct {
	ConnectorState struct {
		State string `json:"state"`
	} `json:"connectorState"`
	KsmlStatus struct {
		Status string `json:"status"`
	} `json:"ksmlStatus"`
}
