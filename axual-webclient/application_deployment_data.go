package webclient

type ApplicationDeploymentCreateResponse string

type ApplicationDeploymentUpdateResponse interface{}
type Config struct {
	ConfigKey   string `json:"configKey"`
	ConfigValue string `json:"configValue"`
}
type ApplicationDeploymentResponse struct {
	Configs    []Config `json:"configs"`
	State      string   `json:"state"`
	Uid        string   `json:"uid"`
	CreatedAt  string   `json:"created_at"`
	ModifiedAt string   `json:"modified_at"`
	CreatedBy  string   `json:"created_by"`
	ModifiedBy string   `json:"modified_by"`
	Embedded   struct {
		Application struct {
			Visibility       string `json:"visibility"`
			ApplicationClass string `json:"applicationClass"`
			Name             string `json:"name"`
			Type             string `json:"type"`
			ShortName        string `json:"shortName"`
			Description      string `json:"description"`
			Owners           struct {
				Name         string `json:"name"`
				EmailAddress struct {
					Email string `json:"email"`
				} `json:"emailAddress"`
				PhoneNumber string      `json:"phoneNumber"`
				Properties  interface{} `json:"properties"`
				Uid         string      `json:"uid"`
				CreatedAt   string      `json:"created_at"`
				ModifiedAt  string      `json:"modified_at"`
				CreatedBy   string      `json:"created_by"`
				ModifiedBy  string      `json:"modified_by"`
			} `json:"owners"`
			Uid        string `json:"uid"`
			CreatedAt  string `json:"created_at"`
			ModifiedAt string `json:"modified_at"`
			CreatedBy  string `json:"created_by"`
			ModifiedBy string `json:"modified_by"`
			Links      struct {
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
			} `json:"_links"`
		} `json:"application"`
		Environment struct {
			Visibility  string `json:"visibility"`
			Name        string `json:"name"`
			ShortName   string `json:"shortName"`
			Description string `json:"description"`
			Color       string `json:"color"`
			Uid         string `json:"uid"`
			CreatedAt   string `json:"created_at"`
			ModifiedAt  string `json:"modified_at"`
			CreatedBy   string `json:"created_by"`
			ModifiedBy  string `json:"modified_by"`
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
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		ApplicationDeployment struct {
			Href string `json:"href"`
		} `json:"applicationDeployment"`
		Edit struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"edit"`
		Delete struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"delete"`
		Application struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"application"`
		Environment struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"environment"`
	} `json:"_links"`
}

type ApplicationDeploymentCreateRequest struct {
	Application string            `json:"application"`
	Environment string            `json:"environment"`
	Configs     map[string]string `json:"configs"`
}

type ApplicationDeploymentUpdateRequest struct {
	Configs map[string]string `json:"configs"`
}

type ApplicationDeploymentOperationRequest struct {
	Action string `json:"action"`
}

type ApplicationDeploymentFindByApplicationAndEnvironmentResponse struct {
	Embedded struct {
		ApplicationDeploymentResponses []ApplicationDeploymentResponse `json:"application_deployments"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
	} `json:"_links"`
}

type ApplicationDeploymentStatusResponse struct {
	ConnectorState struct {
		State    string  `json:"state"`
		WorkerId *string `json:"workerId"` // Pointer to string for nullable field
		Trace    *string `json:"trace"`    // Pointer to string because a nullable field
	} `json:"connectorState"`
	TaskStates *[]struct { // Pointer to a slice of structs for handling `null`
		Id       int    `json:"id"`
		Status   string `json:"status"`
		WorkerId string `json:"workerId"`
		Trace    string `json:"trace"`
	} `json:"taskStates"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		RestartTask *struct { // Pointer to struct for handling cases where it might be null
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
		} `json:"restartTask"`
	} `json:"_links"`
}
