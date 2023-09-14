package webclient

type TopicResponse struct {
	Properties      map[string]interface{} `json:"properties"`
	Name            string                 `json:"name"`
	Description     interface{}            `json:"description"`
	KeyType         string                 `json:"keyType"`
	KeySchema       string                 `json:"keySchema"`
	ValueType       string                 `json:"valueType"`
	ValueSchema     string                 `json:"valueSchema"`
	RetentionPolicy string                 `json:"retentionPolicy"`
	Uid             string                 `json:"uid"`
	CreatedAt       string                 `json:"created_at"`
	ModifiedAt      string                 `json:"modified_at"`
	CreatedBy       string                 `json:"created_by"`
	ModifiedBy      string                 `json:"modified_by"`
	Embedded        struct {
		ValueSchema struct {
			Description string `json:"description"`
			Name        string `json:"name"`
			Uid         string `json:"uid"`
			Links       struct {
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"valueSchema"`
		Owners struct {
			EmailAddress interface{} `json:"emailAddress"`
			PhoneNumber  interface{} `json:"phoneNumber"`
			Name         string      `json:"name"`
			Uid          string      `json:"uid"`
			Links        struct {
				Members struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"members"`
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"owners"`
		KeySchema struct {
			Description string `json:"description"`
			Name        string `json:"name"`
			Uid         string `json:"uid"`
			Links       struct {
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"keySchema"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		Topic struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"topic"`
		Edit struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"edit"`
		Delete struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"delete"`
		Confidentiality struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"confidentiality"`
		ValueSchema struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"valueSchema"`
		Owners struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"owners"`
		KeySchema struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"keySchema"`
		Integrity struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"integrity"`
	} `json:"_links"`
}

type TopicRequest struct {
	Name            string                 `json:"name,omitempty"`
	Description     string                 `json:"description"`
	KeyType         string                 `json:"keyType,omitempty"`
	KeySchema       string                 `json:"keySchema,omitempty"`
	ValueType       string                 `json:"valueType,omitempty"`
	ValueSchema     string                 `json:"valueSchema,omitempty"`
	Owners          string                 `json:"owners,omitempty"`
	RetentionPolicy string                 `json:"retentionPolicy,omitempty"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
}
