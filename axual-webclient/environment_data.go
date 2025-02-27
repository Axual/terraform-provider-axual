package webclient

type EnvironmentsResponse struct {
	Embedded struct {
		Environments []struct {
			Name        string `json:"name"`
			Visibility  string `json:"visibility"`
			Description string `json:"description"`
			ShortName   string `json:"shortName"`
			Color       string `json:"color"`
			CreatedAt   string `json:"created_at"`
			CreatedBy   string `json:"created_by"`
			ModifiedAt  string `json:"modified_at"`
			ModifiedBy  string `json:"modified_by"`
			Uid         string `json:"uid"`
			Links       struct {
				Self struct {
					Href  string `json:"href"`
					Title string `json:"title"`
				} `json:"self"`
				Environment struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"environment"`
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
			} `json:"_links"`
		} `json:"environments"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"self"`
		Profile struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"profile"`
		Search struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"search"`
		Create struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"create"`
	} `json:"_links"`
	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

type EnvironmentResponse struct {
	Properties          map[string]interface{} `json:"properties"`
	Settings            map[string]interface{} `json:"settings"`
	Name                string                 `json:"name"`
	ShortName           string                 `json:"shortName"`
	Description         string                 `json:"description"`
	Color               string                 `json:"color"`
	AuthorizationIssuer string                 `json:"authorizationIssuer"`
	Visibility          string                 `json:"visibility"`
	RetentionTime       int                    `json:"retentionTime"`
	Partitions          int                    `json:"partitions"`
	Private             bool                   `json:"private"`
	AutoApproved        bool                   `json:"autoApproved"`
	Uid                 string                 `json:"uid"`
	CreatedAt           string                 `json:"created_at"`
	ModifiedAt          string                 `json:"modified_at"`
	CreatedBy           string                 `json:"created_by"`
	ModifiedBy          string                 `json:"modified_by"`
	Embedded            struct {
		Instance struct {
			Name             string `json:"name"`
			Description      string `json:"description"`
			InstanceClusters []struct {
				Cluster struct {
					Name                string `json:"name"`
					Description         string `json:"description"`
					Location            string `json:"location"`
					BillingCloudEnabled bool   `json:"billingCloudEnabled"`
					ApiUrl              string `json:"apiUrl"`
					ClusterBrowseUrl    string `json:"clusterBrowseUrl"`
					BootstrapServers    []struct {
						BootstrapServer string `json:"bootstrapServer"`
					} `json:"bootstrapServers"`
					Uid string `json:"uid"`
				} `json:"cluster"`
				SchemaRegistryUrls string `json:"schemaRegistryUrls"`
			} `json:"instanceClusters"`
			Uid   string `json:"uid"`
			Links struct {
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
				SupportTier struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
				} `json:"supportTier"`
			} `json:"_links"`
		} `json:"instance"`
		Owners struct {
			Name         string      `json:"name"`
			EmailAddress interface{} `json:"emailAddress"`
			PhoneNumber  interface{} `json:"phoneNumber"`
			Uid          string      `json:"uid"`
			Links        struct {
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
				Members []struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"members"`
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
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		Environment struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"environment"`
		Edit struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"edit"`
		Synchronize struct {
			Href string `json:"href"`
		} `json:"synchronize"`
		Delete struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"delete"`
		Instance struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"instance"`
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
	} `json:"_links"`
}

type EnvironmentRequest struct {
	Name                string                 `json:"name,omitempty"`
	ShortName           string                 `json:"shortName,omitempty"`
	Description         interface{}            `json:"description,omitempty"`
	Color               string                 `json:"color,omitempty"`
	RetentionTime       int                    `json:"retentionTime,omitempty"`
	Partitions          int                    `json:"partitions,omitempty"`
	AuthorizationIssuer string                 `json:"authorizationIssuer,omitempty"`
	Visibility          string                 `json:"visibility,omitempty"`
	Instance            string                 `json:"instance,omitempty"`
	Owners              string                 `json:"owners,omitempty"`
	Viewers             []string               `json:"viewers"`
	Properties          map[string]interface{} `json:"properties,omitempty"`
	Settings            map[string]interface{} `json:"settings"`
}
