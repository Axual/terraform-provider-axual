package webclient

type KsmlRuntimeResponse struct {
	Uid              string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Url              string                 `json:"url"`
	SchemaRegistries []SchemaRegistryConfig `json:"schemaRegistries"`
	OwnerGroupId     string                 `json:"ownerGroupId"`
	AuthorizedGroups []AuthorizedGroup      `json:"authorizedGroups"`
}

// KsmlRuntimeRequest is the body of both the create POST and the update PATCH. Description and
// SchemaRegistries are pointers and always serialised: the PATCH only applies fields present in the
// body, so clearing either one needs an explicit null - an omitted key leaves the old value alone.
type KsmlRuntimeRequest struct {
	Name        string  `json:"name,omitempty"`
	Description *string `json:"description"`
	Url         string  `json:"url,omitempty"`

	SchemaRegistries *[]SchemaRegistryConfig `json:"schemaRegistries"`

	OwnerGroupId string `json:"ownerGroupId,omitempty"`
	// Omitted keeps the stored groups, an empty array clears them. The API rejects an explicit null.
	AuthorizedGroupIds *[]string `json:"authorizedGroupIds,omitempty"`
}
