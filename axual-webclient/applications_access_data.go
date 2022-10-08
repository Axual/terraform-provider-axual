package webclient

type ApplicationAccessResponse struct {
	AccessType string `json:"accessType"`
	Uid        string `json:"uid"`
	CreatedAt  string `json:"created_at"`
	ModifiedAt string `json:"modified_at"`
	CreatedBy  string `json:"created_by"`
	ModifiedBy string `json:"modified_by"`
	Embedded   struct {
		Application struct {
			ApplicationClass interface{} `json:"applicationClass"`
			Name             string      `json:"name"`
			Type             string      `json:"type"`
			Description      string      `json:"description"`
			ShortName        string      `json:"shortName"`
			Owners           struct {
				Name         string      `json:"name"`
				EmailAddress interface{} `json:"emailAddress"`
				PhoneNumber  interface{} `json:"phoneNumber"`
				Uid          string      `json:"uid"`
				CreatedAt    string      `json:"created_at"`
				ModifiedAt   string      `json:"modified_at"`
				CreatedBy    string      `json:"created_by"`
				ModifiedBy   string      `json:"modified_by"`
			} `json:"owners"`
			Visibility string `json:"visibility"`
			Uid        string `json:"uid"`
			Links      struct {
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"application"`
		Stream struct {
			Name        string `json:"name"`
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
			Uid             string      `json:"uid"`
			Links           struct {
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
				Owners struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"owners"`
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
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		ApplicationAccess struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"applicationAccess"`
		CreateApplicationAccessGrant struct {
			Href string `json:"href"`
		} `json:"createApplicationAccessGrant"`
		Grants struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"grants"`
		Application struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"application"`
		Stream struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"stream"`
	} `json:"_links"`
}

type ApplicationAccessRequest struct {
	Application string `json:"application"`
	Stream      string `json:"stream"`
	AccessType  string `json:"accessType"`
}

type ApplicationAccessList struct {
	Embedded struct {
		ApplicationAccess []ApplicationAccess `json:"application_access"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
	} `json:"_links"`
}

type ApplicationAccess struct {
	AccessType string `json:"accessType"`
	Uid        string `json:"uid"`
	CreatedAt  string `json:"created_at"`
	ModifiedAt string `json:"modified_at"`
	CreatedBy  string `json:"created_by"`
	ModifiedBy string `json:"modified_by"`
	Embedded   struct {
		Grants []struct {
			Comment     interface{} `json:"comment"`
			Environment struct {
				Properties struct {
					SpacetimeMs string `json:"spacetime.ms"`
				} `json:"properties"`
				Name                string `json:"name"`
				ShortName           string `json:"shortName"`
				Description         string `json:"description"`
				Color               string `json:"color"`
				AuthorizationIssuer string `json:"authorizationIssuer"`
				Visibility          string `json:"visibility"`
				RetentionTime       int    `json:"retentionTime"`
				Partitions          int    `json:"partitions"`
				Private             bool   `json:"private"`
				AutoApproved        bool   `json:"autoApproved"`
				Uid                 string `json:"uid"`
				CreatedAt           string `json:"created_at"`
				ModifiedAt          string `json:"modified_at"`
				CreatedBy           string `json:"created_by"`
				ModifiedBy          string `json:"modified_by"`
			} `json:"environment"`
			Status      string      `json:"status"`
			RequestedBy string      `json:"requestedBy"`
			RequestedAt string      `json:"requestedAt"`
			ProcessedAt string      `json:"processedAt"`
			ProcessedBy interface{} `json:"processedBy"`
			Uid         string      `json:"uid"`
			Links       struct {
				Environment struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"environment"`
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"grants"`
		Application struct {
			ApplicationClass interface{} `json:"applicationClass"`
			Name             string      `json:"name"`
			Type             string      `json:"type"`
			Description      string      `json:"description"`
			ShortName        string      `json:"shortName"`
			Owners           struct {
				Name         string      `json:"name"`
				EmailAddress interface{} `json:"emailAddress"`
				PhoneNumber  interface{} `json:"phoneNumber"`
				Uid          string      `json:"uid"`
				CreatedAt    string      `json:"created_at"`
				ModifiedAt   string      `json:"modified_at"`
				CreatedBy    string      `json:"created_by"`
				ModifiedBy   string      `json:"modified_by"`
			} `json:"owners"`
			Visibility string `json:"visibility"`
			Uid        string `json:"uid"`
			Links      struct {
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"application"`
		Stream struct {
			Name        string `json:"name"`
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
			Uid             string      `json:"uid"`
			Links           struct {
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
				Owners struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"owners"`
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
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		ApplicationAccess struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"applicationAccess"`
		CreateApplicationAccessGrant struct {
			Href string `json:"href"`
		} `json:"createApplicationAccessGrant"`
		Grants struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"grants"`
		Application struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"application"`
		Stream struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"stream"`
	} `json:"_links"`
}
