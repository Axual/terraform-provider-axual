package webclient

type InstanceResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ShortName   string `json:"shortName"`
	Uid         string `json:"uid"`
}

type InstancesResponseByAttributes struct {
	Embedded struct {
		Instances []struct {
			Links struct {
				Self struct {
					Href  string `json:"href"`
					Title string `json:"title"`
				} `json:"self"`
				SupportTier struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"supportTier"`
			} `json:"_links"`
			Name        string `json:"name"`
			ShortName   string `json:"shortName"`
			Description string `json:"description"`
			Uid         string `json:"uid"`
			CreatedBy   string `json:"created_by"`
			ModifiedAt  string `json:"modified_at"`
			ModifiedBy  string `json:"modified_by"`
			CreatedAt   string `json:"created_at"`
		} `json:"instances"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
	} `json:"_links"`
}
