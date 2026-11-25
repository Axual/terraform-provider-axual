package webclient

// KsmlRuntimeType is the `type` of a KSML runtime on /runtimes.
const KsmlRuntimeType = "KSML"

// KsmlRuntimeConfig is the `config` object of a runtime of type KSML.
type KsmlRuntimeConfig struct {
	Url string `json:"url"`
	// Send a list, never nil: [] is no schema registries on a create, and empties the stored list on an update.
	SchemaRegistries []SchemaRegistryConfig `json:"schemaRegistries"`
}

type KsmlRuntimeResponse struct {
	Uid              string            `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	OwnerGroupId     string            `json:"ownerGroupId"`
	AuthorizedGroups []AuthorizedGroup `json:"authorizedGroups"`
	Config           KsmlRuntimeConfig `json:"config"`
}

// KsmlRuntimeCreateRequest is the body of POST /runtimes. A description and authorized groups without a
// value are left out, and Platform Manager then stores none.
type KsmlRuntimeCreateRequest struct {
	Type               string            `json:"type"`
	Name               string            `json:"name"`
	Description        *string           `json:"description,omitempty"`
	InstanceId         string            `json:"instanceId"`
	ClusterId          string            `json:"clusterId"`
	OwnerGroupId       string            `json:"ownerGroupId"`
	AuthorizedGroupIds *[]string         `json:"authorizedGroupIds,omitempty"`
	Config             KsmlRuntimeConfig `json:"config"`
}

// KsmlRuntimeUpdateRequest is the body of PATCH /runtimes/{id}, which only applies the fields in the body.
// Description is always serialised: clearing it needs an explicit null, an omitted key leaves the old value
// alone. The name, type, instance and cluster of a runtime are fixed and not sent.
type KsmlRuntimeUpdateRequest struct {
	Description  *string `json:"description"`
	OwnerGroupId string  `json:"ownerGroupId,omitempty"`
	// Omitted keeps the stored groups, an empty array clears them. The API rejects an explicit null.
	AuthorizedGroupIds *[]string         `json:"authorizedGroupIds,omitempty"`
	Config             KsmlRuntimeConfig `json:"config"`
}
