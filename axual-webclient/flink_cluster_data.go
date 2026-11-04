package webclient

// SchemaRegistryConfig is one schema registry endpoint a Flink Cluster reads schemas from. Flink
// SQL jobs over AVRO topics need it: the platform injects the url into the generated table DDL.
type SchemaRegistryConfig struct {
	Type string `json:"type"`
	Url  string `json:"url"`
}

// FlinkClusterAuthorizedGroup is a Group that may deploy Flink SQL to a Flink Cluster.
type FlinkClusterAuthorizedGroup struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type FlinkClusterResponse struct {
	SchemaRegistries []SchemaRegistryConfig `json:"schemaRegistries"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Url              string                 `json:"url"`
	Workspace        string                 `json:"workspace"`
	Namespace        string                 `json:"namespace"`
	DeploymentTarget string                 `json:"deploymentTarget"`
	Uid              string                 `json:"id"`
	// OwnerGroupId is empty for a Flink Cluster registered before owners existed.
	OwnerGroupId     string                        `json:"ownerGroupId"`
	OwnerGroupName   string                        `json:"ownerGroupName"`
	AuthorizedGroups []FlinkClusterAuthorizedGroup `json:"authorizedGroups"`
}

// FlinkClusterRequest is the body of both the create POST and the update PATCH. Description and
// SchemaRegistries are pointers and always serialised: the PATCH only applies fields present in the
// body, so clearing either one needs an explicit null - an omitted key leaves the old value alone.
type FlinkClusterRequest struct {
	Name             string  `json:"name,omitempty"`
	Description      *string `json:"description"`
	Url              string  `json:"url,omitempty"`
	Workspace        string  `json:"workspace,omitempty"`
	Namespace        string  `json:"namespace,omitempty"`
	DeploymentTarget string  `json:"deploymentTarget,omitempty"`
	ApiToken         string  `json:"apiToken,omitempty"`

	SchemaRegistries *[]SchemaRegistryConfig `json:"schemaRegistries"`

	OwnerGroupId string `json:"ownerGroupId,omitempty"`
	// AuthorizedGroupIds is left out to keep the stored groups, and sent as an empty array to clear
	// them. The API rejects an explicit null, so it must never be serialised as one.
	AuthorizedGroupIds *[]string `json:"authorizedGroupIds,omitempty"`
}
