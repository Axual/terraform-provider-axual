package webclient

type UsersResponse struct {
	Embedded struct {
		Users []struct {
			UID          string `json:"uid"`
			EmailAddress struct {
				Email string `json:"email"`
			} `json:"emailAddress"`
			Lastname    string      `json:"lastName"`
			MiddleName  interface{} `json:"middleName"`
			PhoneNumber interface{} `json:"phoneNumber"`
			Firstname   string      `json:"firstName"`
		} `json:"users"`
	} `json:"_embedded"`
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
	Uid string `json:"uid"`
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
