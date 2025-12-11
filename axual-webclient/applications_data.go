package webclient

type ApplicationResponse struct {
	Name             string `json:"name"`
	ShortName        string `json:"shortName"`
	Description      string `json:"description"`
	ApplicationType  string `json:"applicationType"`
	Type             string `json:"type"`
	ApplicationClass string `json:"applicationClass"`
	Visibility       string `json:"visibility"`
	Owners           struct {
		Name         string `json:"name"`
		EmailAddress struct {
			Email string `json:"email"`
		} `json:"emailAddress"`
		PhoneNumber string `json:"phoneNumber"`
		Uid         string `json:"uid"`
		CreatedAt   string `json:"created_at"`
		ModifiedAt  string `json:"modified_at"`
		CreatedBy   string `json:"created_by"`
		ModifiedBy  string `json:"modified_by"`
		Embedded    struct {
			Members []struct {
				FirstName    string `json:"firstName"`
				LastName     string `json:"lastName"`
				EmailAddress struct {
					Email string `json:"email"`
				} `json:"emailAddress"`
				Uid         string      `json:"uid"`
				PhoneNumber interface{} `json:"phoneNumber"`
				MiddleName  *string     `json:"middleName"`
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
			Delete struct {
				Href  string `json:"href"`
				Title string `json:"title"`
			} `json:"delete"`
			Members []struct {
				Href      string `json:"href"`
				Templated bool   `json:"templated"`
				Title     string `json:"title"`
			} `json:"members"`
		} `json:"_links"`
	} `json:"owners"`
	Embedded struct {
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
	AllApplicationIds []string `json:"allApplicationIds"`
	Uid               string   `json:"uid"`
	CreatedAt         string   `json:"created_at"`
	ModifiedAt        string   `json:"modified_at"`
	CreatedBy         string   `json:"created_by"`
	ModifiedBy        string   `json:"modified_by"`
	ApplicationId     string   `json:"applicationId"`
	Links             struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		Application struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"application"`
		Viewers struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"viewers"`
		Edit struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"edit"`
		CreateApplicationPrincipal struct {
			Href string `json:"href"`
		} `json:"createApplicationPrincipal"`
		CreateApplicationAccess struct {
			Href string `json:"href"`
		} `json:"createApplicationAccess"`
		Delete struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"delete"`
	} `json:"_links"`
}

type ApplicationRequest struct {
	ApplicationType  string   `json:"applicationType"`
	ApplicationId    string   `json:"applicationId"`
	Name             string   `json:"name"`
	ShortName        string   `json:"shortName"`
	Owners           string   `json:"owners"`
	Viewers          []string `json:"viewers"`
	Type             string   `json:"type"`
	ApplicationClass string   `json:"applicationClass,omitempty"`
	Visibility       string   `json:"visibility"`
	Description      string   `json:"description"`
}
