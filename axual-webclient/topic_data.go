package webclient

type TopicResponse struct {
	Properties      map[string]interface{} `json:"properties"`
	Name            string                 `json:"name"`
	Description     interface{}            `json:"description"`
	KeyType         string                 `json:"keyType"`
	ValueType       string                 `json:"valueType"`
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
				Members []struct {
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
		Viewers []struct {
			Name         string      `json:"name"`
			EmailAddress interface{} `json:"emailAddress"`
			PhoneNumber  interface{} `json:"phoneNumber"`
			Uid          string      `json:"uid"`
			CreatedAt    string      `json:"created_at"`
			CreatedBy    string      `json:"created_by"`
			ModifiedAt   string      `json:"modified_at"`
			ModifiedBy   string      `json:"modified_by"`
			Links        struct {
				Managers struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"managers"`
				Members []struct {
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
		} `json:"viewers,omitempty"`
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
		Viewers struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"viewers"`
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
			Name            string `json:"name"`
			Description     string `json:"description"`
			RetentionPolicy string `json:"retentionPolicy"`
			Integrity       string `json:"integrity"`
			Confidentiality string `json:"confidentiality"`
			CreatedAt       string `json:"created_at"`
			CreatedBy       string `json:"created_by"`
			ModifiedAt      string `json:"modified_at"`
			ModifiedBy      string `json:"modified_by"`
			Uid             string `json:"uid"`
			Owners          struct {
				Name         string `json:"name"`
				EmailAddress struct {
					Email string `json:"email"`
				} `json:"emailAddress"`
				PhoneNumber string `json:"phoneNumber"`
				Uid         string `json:"uid"`
			} `json:"owners"`
			Links struct {
				Self struct {
					Href  string `json:"href"`
					Title string `json:"title"`
				} `json:"self"`
				Stream struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"stream"`
			} `json:"_links"`
		} `json:"streams"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
	} `json:"_links"`
}
