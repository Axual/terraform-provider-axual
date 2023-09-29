package webclient

type ApplicationResponse struct {
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
			Members interface{} `json:"members"` //This can't be []struct because if group has 1 member, it returns an object and not an array
		} `json:"_links"`
	} `json:"owners"`
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
	ApplicationType string `json:"applicationType"`
	ApplicationId   string `json:"applicationId"`
	Name            string `json:"name"`
	ShortName       string `json:"shortName"`
	Owners          string `json:"owners"`
	Type            string `json:"type"`
	Visibility      string `json:"visibility"`
	Description     string `json:"description"`
}

type ApplicationByNameResponse struct {
	Embedded struct {
		Applications []struct {
			Name             string `json:"name"`
			Visibility       string `json:"visibility"`
			Description      string `json:"description"`
			ShortName        string `json:"shortName"`
			Type             string `json:"type"`
			ApplicationClass string `json:"applicationClass"`
			CreatedAt        string `json:"created_at"`
			CreatedBy        string `json:"created_by"`
			ModifiedAt       string `json:"modified_at"`
			ModifiedBy       string `json:"modified_by"`
			Uid              string `json:"uid"`
			Owners           struct {
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
				Application struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"application"`
			} `json:"_links"`
		} `json:"applications"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
	} `json:"_links"`
}
