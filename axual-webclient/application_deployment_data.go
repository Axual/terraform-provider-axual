package webclient

// ApplicationDeploymentCreateResponse holds what the POST /application_deployments response
// exposes about the created deployment. The API returns no body, so the Uid comes from the
// response's Location header and is empty when that header is absent.
type ApplicationDeploymentCreateResponse struct {
	Uid string
}

type ApplicationDeploymentUpdateResponse interface{}
type Config struct {
	ConfigKey   string `json:"configKey"`
	ConfigValue string `json:"configValue"`
}
type ApplicationDeploymentResponse struct {
	Configs  []Config `json:"configs"`
	State    string   `json:"state"`
	Uid      string   `json:"uid"`
	TargetId string   `json:"targetId,omitempty"`
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
	TargetId    string            `json:"targetId,omitempty"`
}

type ApplicationDeploymentUpdateRequest struct {
	Configs  map[string]string `json:"configs"`
	TargetId string            `json:"targetId,omitempty"`
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
	FlinkStatus struct {
		Status string `json:"status"`
	} `json:"flinkStatus"`
}
