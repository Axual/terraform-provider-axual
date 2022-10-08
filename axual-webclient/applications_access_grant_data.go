package webclient

type ApplicationAccessGrant struct {
	Status      string      `json:"status"`
	RequestedBy string      `json:"requestedBy"`
	ProcessedBy interface{} `json:"processedBy"`
	Comment     interface{} `json:"comment"`
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
