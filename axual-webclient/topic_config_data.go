package webclient

type TopicConfigResponse struct {
	Properties    map[string]interface{} `json:"properties"`
	RetentionTime int                    `json:"retentionTime"`
	Partitions    int                    `json:"partitions"`
	Uid           string                 `json:"uid"`
	CreatedAt     string                 `json:"created_at"`
	ModifiedAt    string                 `json:"modified_at"`
	CreatedBy     string                 `json:"created_by"`
	ModifiedBy    string                 `json:"modified_by"`
	Embedded      struct {
		Environment struct {
			Visibility  string `json:"visibility"`
			ShortName   string `json:"shortName"`
			Description string `json:"description"`
			Color       string `json:"color"`
			Name        string `json:"name"`
			Uid         string `json:"uid"`
			Links       struct {
				Owners struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"owners"`
				Instance struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"instance"`
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"environment"`
		Topic struct {
			Description string `json:"description"`
			Owners      struct {
				Name         string      `json:"name"`
				EmailAddress interface{} `json:"emailAddress"`
				PhoneNumber  interface{} `json:"phoneNumber"`
				Uid          string      `json:"uid"`
				CreatedAt    string      `json:"created_at"`
				ModifiedAt   string      `json:"modified_at"`
				CreatedBy    string      `json:"created_by"`
				ModifiedBy   string      `json:"modified_by"`
			} `json:"owners"`
			RetentionPolicy string      `json:"retentionPolicy"`
			Integrity       interface{} `json:"integrity"`
			Confidentiality interface{} `json:"confidentiality"`
			Name            string      `json:"name"`
			Uid             string      `json:"uid"`
			Links           struct {
				Owners struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"owners"`
				Integrity struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"integrity"`
				Confidentiality struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"confidentiality"`
				KeySchema struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"keySchema"`
				ValueSchema struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"valueSchema"`
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"topic"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		TopicConfig struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"topicConfig"`
		Edit struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"edit"`
		Delete struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"delete"`
		Environment struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"environment"`
		KeySchemaVersion struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"keySchemaVersion"`
		Topic struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"topic"`
		ValueSchemaVersion struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"valueSchemaVersion"`
	} `json:"_links"`
}

type TopicConfigRequest struct {
	Partitions    int                    `json:"partitions,omitempty"`
	RetentionTime int                    `json:"retentionTime,omitempty"`
	Topic        string                 `json:"topic,omitempty"`
	Environment   string                 `json:"environment,omitempty"`
	Properties    map[string]interface{} `json:"properties,omitempty"`
}
