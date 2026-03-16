package webclient

type TopicConfigResponse struct {
	Properties         map[string]interface{} `json:"properties"`
	RetentionTime      int                    `json:"retentionTime"`
	Partitions         int                    `json:"partitions"`
	Uid                string                 `json:"uid"`
	KeySchemaVersion   string                 `json:"key_schema_version"`
	ValueSchemaVersion string                 `json:"value_schema_version"`
	Embedded           struct {
		Environment struct {
			ShortName string `json:"shortName"`
			Uid       string `json:"uid"`
		} `json:"environment"`
		Stream struct {
			Name string `json:"name"`
			Uid  string `json:"uid"`
		} `json:"stream"`
	} `json:"_embedded"`
}

type TopicConfigRequest struct {
	Partitions         int                    `json:"partitions,omitempty"`
	RetentionTime      int                    `json:"retentionTime,omitempty"`
	Stream             string                 `json:"stream,omitempty"`
	Environment        string                 `json:"environment,omitempty"`
	Properties         map[string]interface{} `json:"properties,omitempty"`
	KeySchemaVersion   string                 `json:"keySchemaVersion,omitempty"`
	ValueSchemaVersion string                 `json:"valueSchemaVersion,omitempty"`
	Force              bool                   `json:"force,omitempty"`
}

type PermissionRequest struct {
	Type   string   `json:"type"`
	Groups []string `json:"groups,omitempty"`
	Users  []string `json:"users,omitempty"`
}

type PermissionResponse struct {
	Uid  string `json:"uid"`
	Type string `json:"type"`
}
