package webclient

type FlinkClusterResponse struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Url              string `json:"url"`
	Workspace        string `json:"workspace"`
	Namespace        string `json:"namespace"`
	DeploymentTarget string `json:"deploymentTarget"`
	Uid              string `json:"id"`
}

// FlinkClusterRequest is the body of both the create POST and the update PATCH. Description is a
// pointer and always serialised: the PATCH only applies fields present in the body, so clearing a
// description needs an explicit `"description": null` - an omitted key leaves the old text in place.
type FlinkClusterRequest struct {
	Name             string  `json:"name,omitempty"`
	Description      *string `json:"description"`
	Url              string  `json:"url,omitempty"`
	Workspace        string  `json:"workspace,omitempty"`
	Namespace        string  `json:"namespace,omitempty"`
	DeploymentTarget string  `json:"deploymentTarget,omitempty"`
	ApiToken         string  `json:"apiToken,omitempty"`
}
