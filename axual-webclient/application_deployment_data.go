package webclient

// ApplicationDeploymentCreateResponse holds what the POST /application_deployments response
// exposes about the created deployment. The API returns no body, so the Uid comes from the
// response's Location header and is empty when that header is absent.
type ApplicationDeploymentCreateResponse struct {
	Uid string
}

type ApplicationDeploymentUpdateResponse interface{}

// Link is one HAL `_links` entry.
type Link struct {
	Href      string `json:"href"`
	Title     string `json:"title,omitempty"`
	Templated bool   `json:"templated,omitempty"`
}

// Links are the HAL `_links` of a response. The API advertises the actions that are valid for
// a deployment's current state as links, so the presence of a rel is the authority on whether
// that action can be taken - the same rel names are used for every application type.
type Links map[string]Link

// Link relations advertised by the Application Deployment endpoints.
const (
	RelStop   = "stop"
	RelStart  = "start"
	RelDelete = "delete"
)

func (l Links) Has(rel string) bool {
	_, ok := l[rel]
	return ok
}

type Config struct {
	ConfigKey   string `json:"configKey"`
	ConfigValue string `json:"configValue"`
}
type ApplicationDeploymentResponse struct {
	Links    Links    `json:"_links"`
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

// ApplicationDeploymentUpdateRequest carries no targetId on purpose: the deployment target is a
// create-only field, guarded by the Platform Manager, and is only sent by
// ApplicationDeploymentCreateRequest.
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
	FlinkStatus struct {
		Status string `json:"status"`
	} `json:"flinkStatus"`
	Links Links `json:"_links"`
}
