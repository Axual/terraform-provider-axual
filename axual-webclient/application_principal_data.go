package webclient

type ApplicationPrincipalCreateResponse string
type ApplicationPrincipalUpdateResponse interface{}

type ApplicationPrincipalResponse struct {
	Principal          string      `json:"principal"`
	ApplicationPem     string      `json:"applicationPem"`
	PrincipalChain     string      `json:"principalChain"`
	PrivateKeyPem      interface{} `json:"privateKeyPem"`
	PrivateKeyUploaded interface{} `json:"privateKeyUploaded"`
	ExpiresOn          string      `json:"expiresOn"`
	Uid                string      `json:"uid"`
	CreatedAt          string      `json:"created_at"`
	ModifiedAt         string      `json:"modified_at"`
	CreatedBy          string      `json:"created_by"`
	ModifiedBy         string      `json:"modified_by"`
	Embedded           struct {
		Application struct {
			Visibility       string      `json:"visibility"`
			ApplicationClass interface{} `json:"applicationClass"`
			Name             string      `json:"name"`
			Type             string      `json:"type"`
			ShortName        string      `json:"shortName"`
			Description      interface{} `json:"description"`
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
			} `json:"owners"`
			Uid   string `json:"uid"`
			Links struct {
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
		ApplicationPrincipal struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"applicationPrincipal"`
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

type ApplicationPrincipalRequest struct {
	Principal   string `json:"principal"`
	PrivateKey  string `json:"privateKey,omitempty"`
	Application string `json:"application"`
	Environment string `json:"environment"`
	Custom      bool   `json:"custom,omitempty"`
}
type ApplicationPrincipalUpdateRequest struct {
	Principal string `json:"principal"`
}
