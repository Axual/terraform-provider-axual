package webclient

type TopicConfigResponse struct {
	Properties         map[string]interface{} `json:"properties"`
	RetentionTime      int                    `json:"retentionTime"`
	Partitions         int                    `json:"partitions"`
	Uid                string                 `json:"uid"`
	CreatedAt          string                 `json:"created_at"`
	ModifiedAt         string                 `json:"modified_at"`
	CreatedBy          string                 `json:"created_by"`
	ModifiedBy         string                 `json:"modified_by"`
	KeySchemaVersion   string                 `json:"key_schema_version"`
	ValueSchemaVersion string                 `json:"value_schema_version"`
	Embedded           struct {
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
		Stream struct {
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
		} `json:"stream"`
		KeySchemaVersion struct {
			Id       string `json:"schemaVersionUid"`
			SchemaId string `json:"schemaUid"`
			Version  string `json:"version"`
		}
		ValueSchemaVersion struct {
			Id       string `json:"schemaVersionUid"`
			SchemaId string `json:"schemaUid"`
			Version  string `json:"version"`
		}
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		StreamConfig struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"streamConfig"`
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
		Stream struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"stream"`
		ValueSchemaVersion struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"valueSchemaVersion"`
	} `json:"_links"`
}

type TopicConfigRequest struct {
	Partitions         int                    `json:"partitions,omitempty"`
	RetentionTime      int                    `json:"retentionTime,omitempty"`
	Stream             string                 `json:"stream,omitempty"`
	Environment        string                 `json:"environment,omitempty"`
	Properties         map[string]interface{} `json:"properties,omitempty"`
	KeySchemaVersion   string                 `json:"keySchemaVersion,omitempty"`
	ValueSchemaVersion string                 `json:"valueSchemaVersion,omitempty"`
}

type PermissionRequest struct {
	Type   string   `json:"type"`
	Groups []string `json:"groups,omitempty"`
	Users  []string `json:"users,omitempty"`
}

type PermissionResponse struct {
	Uid          string `json:"uid"`
	FirstName    string `json:"firstName,omitempty"`
	LastName     string `json:"lastName,omitempty"`
	MiddleName   string `json:"middleName,omitempty"`
	EmailAddress Email  `json:"emailAddress"`
	Name         string `json:"name,omitempty"`
	Type         string `json:"type"`
}

type Email struct {
	Email string `json:"email"`
}
