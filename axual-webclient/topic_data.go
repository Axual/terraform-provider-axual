package webclient

type TopicResponse struct {
	Properties      map[string]interface{} `json:"properties"`
	Name            string                 `json:"name"`
	Description     interface{}            `json:"description"`
	KeyType         string                 `json:"keyType"`
	ValueType       string                 `json:"valueType"`
	RetentionPolicy string                 `json:"retentionPolicy"`
	Uid             string                 `json:"uid"`
	Embedded        struct {
		KeySchema struct {
			Name string `json:"name"`
			Uid  string `json:"uid"`
		} `json:"keySchema"`
		ValueSchema struct {
			Name string `json:"name"`
			Uid  string `json:"uid"`
		} `json:"valueSchema"`
		Owners struct {
			Name string `json:"name"`
			Uid  string `json:"uid"`
		} `json:"owners"`
		Viewers []struct {
			Name string `json:"name"`
			Uid  string `json:"uid"`
		} `json:"viewers,omitempty"`
	} `json:"_embedded"`
}

type TopicRequest struct {
	Name            string                 `json:"name,omitempty"`
	Description     string                 `json:"description"`
	KeyType         string                 `json:"keyType,omitempty"`
	KeySchema       string                 `json:"keySchema"`
	ValueType       string                 `json:"valueType,omitempty"`
	ValueSchema     string                 `json:"valueSchema"`
	Owners          string                 `json:"owners,omitempty"`
	Viewers         []string               `json:"viewers"`
	RetentionPolicy string                 `json:"retentionPolicy,omitempty"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
}

type TopicsByNameResponse struct {
	Embedded struct {
		Topics []struct {
			Name string `json:"name"`
			Uid  string `json:"uid"`
		} `json:"streams"`
	} `json:"_embedded"`
}
