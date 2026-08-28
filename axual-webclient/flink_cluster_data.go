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

type FlinkClusterRequest struct {
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	Url              string `json:"url,omitempty"`
	Workspace        string `json:"workspace,omitempty"`
	Namespace        string `json:"namespace,omitempty"`
	DeploymentTarget string `json:"deploymentTarget,omitempty"`
	ApiToken         string `json:"apiToken,omitempty"`
}
