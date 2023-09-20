package webclient

type GroupsResponse struct {
	Embedded struct {
		Groups []struct {
			Name         string  `json:"name"`
			PhoneNumber  *string `json:"phoneNumber"`
			EmailAddress *struct {
				Email string `json:"email"`
			} `json:"emailAddress"`
			Uid   string `json:"uid"`
			Links struct {
				Self struct {
					Href  string `json:"href"`
					Title string `json:"title"`
				} `json:"self"`
				Group struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"group"`
				Members struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"members"`
			} `json:"_links"`
		} `json:"groups"`
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

type GroupResponse struct {
	Name         string      `json:"name"`
	EmailAddress interface{} `json:"emailAddress"`
	PhoneNumber  interface{} `json:"phoneNumber"`
	Uid          string      `json:"uid"`
	CreatedAt    string      `json:"created_at"`
	ModifiedAt   string      `json:"modified_at"`
	CreatedBy    string      `json:"created_by"`
	ModifiedBy   string      `json:"modified_by"`
	Embedded     struct {
		Members []struct {
			Uid          string `json:"uid"`
			EmailAddress struct {
				Email string `json:"email"`
			} `json:"emailAddress"`
			LastName    string      `json:"lastName"`
			MiddleName  interface{} `json:"middleName"`
			PhoneNumber interface{} `json:"phoneNumber"`
			FirstName   string      `json:"firstName"`
			Links       struct {
				Self struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"self"`
				Tenant struct {
					Href  string `json:"href"`
					Title string `json:"title"`
				} `json:"tenant"`
			} `json:"_links"`
		} `json:"members"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		Group struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"group"`
		Edit struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"edit"`
		Delete struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"delete"`
		Members struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"members"`
	} `json:"_links"`
}

type GroupRequest struct {
	Name         string      `json:"name,omitempty"`
	EmailAddress interface{} `json:"emailAddress,omitempty"`
	PhoneNumber  interface{} `json:"phoneNumber,omitempty"`
	Members      []string    `json:"members,omitempty"`
}

type GroupByNameResponse struct {
	Embedded struct {
		Groups []struct {
			Name         string `json:"name"`
			EmailAddress struct {
				Email string `json:"email"`
			} `json:"emailAddress"`
			PhoneNumber interface{} `json:"phoneNumber"`
			Uid         string      `json:"uid"`
			CreatedBy   string      `json:"created_by"`
			CreatedAt   string      `json:"created_at"`
			ModifiedAt  string      `json:"modified_at"`
			ModifiedBy  string      `json:"modified_by"`
			Links       struct {
				Self struct {
					Href  string `json:"href"`
					Title string `json:"title"`
				} `json:"self"`
				Group struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"group"`
				Members struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"members"`
			} `json:"_links"`
		} `json:"groups"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		Create struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"create"`
	} `json:"_links"`
	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}