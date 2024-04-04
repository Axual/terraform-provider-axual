package webclient

type ApplicationAccessGrant struct {
	Status      string      `json:"status"`
	RequestedBy string      `json:"requestedBy"`
	ProcessedBy interface{} `json:"processedBy"`
	Comment     string      `json:"comment"`
	Approved    bool        `json:"approved"`
	Pending     bool        `json:"pending"`
	Uid         string      `json:"uid"`
	CreatedAt   string      `json:"created_at"`
	ModifiedAt  string      `json:"modified_at"`
	CreatedBy   string      `json:"created_by"`
	ModifiedBy  string      `json:"modified_by"`
	RequestedAt string      `json:"requested_at"`
	ProcessedAt interface{} `json:"processed_at"`
	Embedded    struct {
		Environment struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			ShortName   string `json:"shortName"`
			Visibility  string `json:"visibility"`
			Color       string `json:"color"`
			Uid         string `json:"uid"`
			Links       struct {
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
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"environment"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		ApplicationAccessGrant struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"applicationAccessGrant"`
		Revoke struct {
			Href string `json:"href"`
		} `json:"revoke"`
		Approve struct {
			Href string `json:"href"`
		} `json:"approve"`
		Cancel struct {
			Href string `json:"href"`
		} `json:"cancel"`
		Deny struct {
			Href string `json:"href"`
		} `json:"deny"`
		Environment struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"environment"`
	} `json:"_links"`
}

type ApplicationAccessGrantRevoke struct {
	Reason      string `json:"reason"`
	Environment string `json:"environment"`
}

type ApplicationAccessGrantRequest struct {
	ApplicationId string `json:"applicationId"`
	StreamId      string `json:"streamId"`
	EnvironmentId string `json:"environmentId"`
	AccessType    string `json:"accessType"`
}

type ApplicationAccessGrantResponse struct {
	Id          string `json:"id"`
	OptLock     int    `json:"optLock"`
	Environment struct {
		Id         string `json:"id"`
		OptLock    int    `json:"optLock"`
		Properties struct {
		} `json:"properties"`
		Name        string `json:"name"`
		ShortName   string `json:"shortName"`
		Description string `json:"description"`
		Color       string `json:"color"`
		Instance    struct {
			Id         string `json:"id"`
			OptLock    int    `json:"optLock"`
			Properties struct {
			} `json:"properties"`
			Name             string `json:"name"`
			Description      string `json:"description"`
			APIURL           string `json:"apiUrl"`
			InstanceClusters []struct {
				Cluster struct {
					Id                  string `json:"id"`
					OptLock             int    `json:"optLock"`
					Name                string `json:"name"`
					Description         string `json:"description"`
					Location            string `json:"location"`
					BillingCloudEnabled bool   `json:"billingCloudEnabled"`
					APIURL              string `json:"apiUrl"`
					ClusterBrowseURL    string `json:"clusterBrowseUrl"`
					BootstrapServers    []struct {
						BootstrapServer string `json:"bootstrapServer"`
					} `json:"bootstrapServers"`
					Tenant       interface{} `json:"tenant"`
					ProviderType interface{} `json:"providerType"`
					Uid          string      `json:"uid"`
				} `json:"cluster"`
				SchemaRegistryUrls string `json:"schemaRegistryUrls"`
			} `json:"instanceClusters"`
			CaCerts []struct {
				Pem       string `json:"pem"`
				ExpiresOn string `json:"expiresOn"`
			} `json:"caCerts"`
			ConnectCerts []struct {
				Pem       string      `json:"pem"`
				ExpiresOn interface{} `json:"expiresOn"`
			} `json:"connectCerts"`
			ShortName                    string `json:"shortName"`
			EnabledAuthenticationMethods []struct {
				Rank      int    `json:"rank"`
				Protocol  string `json:"protocol"`
				Mechanism string `json:"mechanism"`
			} `json:"enabledAuthenticationMethods"`
			SupportTier struct {
				Id          string      `json:"id"`
				OptLock     int         `json:"optLock"`
				Name        string      `json:"name"`
				Description string      `json:"description"`
				Uid         string      `json:"uid"`
				CreatedAt   string      `json:"created_at"`
				ModifiedAt  interface{} `json:"modified_at"`
				CreatedBy   interface{} `json:"created_by"`
				ModifiedBy  interface{} `json:"modified_by"`
			} `json:"supportTier"`
			ConnectEnabled               bool   `json:"connectEnabled"`
			ConnectLoggingSupportEnabled bool   `json:"connectLoggingSupportEnabled"`
			ConnectUrls                  string `json:"connectUrls"`
			GranularBrowsePermission     bool   `json:"granularBrowsePermission"`
			EnvironmentMapping           bool   `json:"environmentMapping"`
			Uid                          string `json:"uid"`
			CreatedAt                    string `json:"created_at"`
			ModifiedAt                   string `json:"modified_at"`
			CreatedBy                    string `json:"created_by"`
			ModifiedBy                   string `json:"modified_by"`
		} `json:"instance"`
		AuthorizationIssuer string `json:"authorizationIssuer"`
		Visibility          string `json:"visibility"`
		RetentionTime       int    `json:"retentionTime"`
		Partitions          int    `json:"partitions"`
		Owners              struct {
			Id      string `json:"id"`
			OptLock int    `json:"optLock"`
			Name    string `json:"name"`
			Members []struct {
				Id           string      `json:"id"`
				OptLock      int         `json:"optLock"`
				FirstName    string      `json:"firstName"`
				LastName     string      `json:"lastName"`
				MiddleName   interface{} `json:"middleName"`
				EmailAddress struct {
					Email string `json:"email"`
				} `json:"emailAddress"`
				PhoneNumber interface{} `json:"phoneNumber"`
				Roles       []struct {
					Name string `json:"name"`
				} `json:"roles"`
				FullName   string      `json:"fullName"`
				Uid        string      `json:"uid"`
				CreatedAt  string      `json:"created_at"`
				ModifiedAt string      `json:"modified_at"`
				CreatedBy  interface{} `json:"created_by"`
				ModifiedBy interface{} `json:"modified_by"`
			} `json:"members"`
			EmailAddress struct {
				Email string `json:"email"`
			} `json:"emailAddress"`
			PhoneNumber string `json:"phoneNumber"`
			Uid         string `json:"uid"`
			CreatedAt   string `json:"created_at"`
			ModifiedAt  string `json:"modified_at"`
			CreatedBy   string `json:"created_by"`
			ModifiedBy  string `json:"modified_by"`
		} `json:"owners"`
		Private      bool   `json:"private"`
		AutoApproved bool   `json:"autoApproved"`
		Uid          string `json:"uid"`
		CreatedAt    string `json:"created_at"`
		ModifiedAt   string `json:"modified_at"`
		CreatedBy    string `json:"created_by"`
		ModifiedBy   string `json:"modified_by"`
	} `json:"environment"`
	Status      string      `json:"status"`
	RequestedBy string      `json:"requestedBy"`
	ProcessedBy interface{} `json:"processedBy"`
	Comment     interface{} `json:"comment"`
	Pending     bool        `json:"pending"`
	Approved    bool        `json:"approved"`
	Uid         string      `json:"uid"`
	CreatedAt   string      `json:"created_at"`
	ModifiedAt  string      `json:"modified_at"`
	CreatedBy   string      `json:"created_by"`
	ModifiedBy  string      `json:"modified_by"`
	RequestedAt string      `json:"requested_at"`
	ProcessedAt string      `json:"processed_at"`
}

type GetApplicationAccessGrantsByAttributeResponse struct {
	Embedded struct {
		ApplicationAccessGrantResponses []struct {
			AccessType string `json:"accessType"`
			Uid        string `json:"uid"`
			Comment    string `json:"comment"`
			Embedded   struct {
				Environment struct {
					Properties struct {
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
					Uid                 string `json:"uid"`
					AutoApproved        bool   `json:"autoApproved"`
					CreatedAt           string `json:"created_at"`
					ModifiedAt          string `json:"modified_at"`
					CreatedBy           string `json:"created_by"`
					ModifiedBy          string `json:"modified_by"`
				} `json:"environment"`
				Application struct {
					Name             string      `json:"name"`
					ShortName        string      `json:"shortName"`
					Description      string      `json:"description"`
					ApplicationType  string      `json:"applicationType"`
					Type             string      `json:"type"`
					ApplicationClass interface{} `json:"applicationClass"`
					Visibility       string      `json:"visibility"`
					Owners           struct {
						Name         string `json:"name"`
						EmailAddress struct {
							Email string `json:"email"`
						} `json:"emailAddress"`
						PhoneNumber interface{} `json:"phoneNumber"`
						Uid         string      `json:"uid"`
						CreatedAt   string      `json:"created_at"`
						ModifiedAt  string      `json:"modified_at"`
						CreatedBy   string      `json:"created_by"`
						ModifiedBy  string      `json:"modified_by"`
						Embedded    struct {
							Members []struct {
								Uid          string `json:"uid"`
								EmailAddress struct {
									Email string `json:"email"`
								} `json:"emailAddress"`
								FirstName   string      `json:"firstName"`
								LastName    string      `json:"lastName"`
								PhoneNumber interface{} `json:"phoneNumber"`
								MiddleName  string      `json:"middleName"`
								Links       struct {
									Tenant struct {
										Href  string `json:"href"`
										Title string `json:"title"`
									} `json:"tenant"`
									Self struct {
										Href      string `json:"href"`
										Templated bool   `json:"templated"`
										Title     string `json:"title"`
									} `json:"self"`
								} `json:"_links"`
							} `json:"members"`
						} `json:"_embedded"`
						Links struct {
							Edit struct {
								Href  string `json:"href"`
								Title string `json:"title"`
							} `json:"edit"`
							Members []struct {
								Href      string `json:"href"`
								Templated bool   `json:"templated"`
								Title     string `json:"title"`
							} `json:"members"`
						} `json:"_links"`
					} `json:"owners"`
					AllApplicationIds []string `json:"allApplicationIds"`
					Uid               string   `json:"uid"`
					CreatedAt         string   `json:"created_at"`
					ModifiedAt        string   `json:"modified_at"`
					CreatedBy         string   `json:"created_by"`
					ModifiedBy        string   `json:"modified_by"`
					ApplicationID     string   `json:"applicationId"`
				} `json:"application"`
				Stream struct {
					Properties struct {
					} `json:"properties"`
					Name            string `json:"name"`
					Description     string `json:"description"`
					KeyType         string `json:"keyType"`
					ValueType       string `json:"valueType"`
					RetentionPolicy string `json:"retentionPolicy"`
					Uid             string `json:"uid"`
					CreatedAt       string `json:"created_at"`
					ModifiedAt      string `json:"modified_at"`
					CreatedBy       string `json:"created_by"`
					ModifiedBy      string `json:"modified_by"`
				} `json:"stream"`
			} `json:"_embedded"`
			AppConfigured    bool   `json:"appConfigured"`
			StreamConfigured bool   `json:"streamConfigured"`
			Status           string `json:"status"`
			RequestedByUser  struct {
				FirstName    string `json:"firstName"`
				LastName     string `json:"lastName"`
				MiddleName   string `json:"middleName"`
				EmailAddress struct {
					Email string `json:"email"`
				} `json:"emailAddress"`
				PhoneNumber interface{} `json:"phoneNumber"`
				Roles       []struct {
					Name string `json:"name"`
				} `json:"roles"`
				FullName   string      `json:"fullName"`
				Uid        string      `json:"uid"`
				CreatedAt  string      `json:"created_at"`
				ModifiedAt string      `json:"modified_at"`
				CreatedBy  interface{} `json:"created_by"`
				ModifiedBy string      `json:"modified_by"`
			} `json:"requestedByUser"`
			ProcessedByUser struct {
				FirstName    string `json:"firstName"`
				LastName     string `json:"lastName"`
				MiddleName   string `json:"middleName"`
				EmailAddress struct {
					Email string `json:"email"`
				} `json:"emailAddress"`
				PhoneNumber interface{} `json:"phoneNumber"`
				Roles       []struct {
					Name string `json:"name"`
				} `json:"roles"`
				FullName   string      `json:"fullName"`
				Uid        string      `json:"uid"`
				CreatedAt  string      `json:"created_at"`
				ModifiedAt string      `json:"modified_at"`
				CreatedBy  interface{} `json:"created_by"`
				ModifiedBy string      `json:"modified_by"`
			} `json:"processedByUser"`
			ProcessedAt string `json:"processedAt"`
			RequestedAt string `json:"requestedAt"`
			Links       struct {
				Revoke struct {
					Href string `json:"href"`
				} `json:"revoke"`
			} `json:"_links"`
		} `json:"applicationAccessGrantResponses"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
	} `json:"_links"`
	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}