package webclient

type GroupResponse struct {
	Name         string `json:"name"`
	EmailAddress struct {
		Email string `json:"email"`
	} `json:"emailAddress"`
	PhoneNumber interface{} `json:"phoneNumber"`
	Uid         string      `json:"uid"`
	Embedded    struct {
		Managers []struct {
			Uid string `json:"uid"`
		} `json:"managers"`
		Members []struct {
			Uid string `json:"uid"`
		} `json:"members"`
	} `json:"_embedded"`
}

type GroupRequest struct {
	Name         string      `json:"name,omitempty"`
	EmailAddress interface{} `json:"emailAddress"`
	PhoneNumber  interface{} `json:"phoneNumber"`
	Members      []string    `json:"members"`
	Managers     []string    `json:"managers"`
}

type GetGroupByNameResponse struct {
	Embedded struct {
		Groups []struct {
			Uid string `json:"uid"`
		} `json:"groups"`
	} `json:"_embedded"`
}
