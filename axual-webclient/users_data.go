package webclient

type UsersResponse struct {
	Embedded struct {
		Users []struct {
			UID          string `json:"uid"`
			Emailaddress struct {
				Email string `json:"email"`
			} `json:"emailAddress"`
			Lastname    string      `json:"lastName"`
			Middlename  interface{} `json:"middleName"`
			Phonenumber interface{} `json:"phoneNumber"`
			Firstname   string      `json:"firstName"`
			Links       struct {
				Self struct {
					Href  string `json:"href"`
					Title string `json:"title"`
				} `json:"self"`
				User struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"user"`
				Tenant struct {
					Href  string `json:"href"`
					Title string `json:"title"`
				} `json:"tenant"`
			} `json:"_links"`
		} `json:"users"`
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
		Totalelements int `json:"totalElements"`
		Totalpages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

type UserResponse struct {
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
	Uid        string      `json:"uid"`
	CreatedAt  string      `json:"created_at"`
	ModifiedAt string      `json:"modified_at"`
	CreatedBy  interface{} `json:"created_by"`
	ModifiedBy string      `json:"modified_by"`
	Links      struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
		User struct {
			Href      string `json:"href"`
			Templated bool   `json:"templated"`
			Title     string `json:"title"`
		} `json:"user"`
		Edit struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"edit"`
		AssignRoles struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"assign_roles"`
		Delete struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"delete"`
		Tenant struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"tenant"`
	} `json:"_links"`
}

type UserRequest struct {
	FirstName    string     `json:"firstName,omitempty"`
	LastName     string     `json:"lastName,omitempty"`
	MiddleName   string     `json:"middleName"`
	EmailAddress string     `json:"emailAddress,omitempty"`
	PhoneNumber  string     `json:"phoneNumber,omitempty"`
	Roles        []UserRole `json:"roles,omitempty"`
}

type UserRequestWithPass struct {
	UserRequest
	Password string `json:"password"`
}

type UserRole struct {
	Name string `json:"name"`
}
