package webclient

// type SchemasResponse struct {
// 	Embedded struct {
// 		Schemas []any  `json:"schemas"`
// 	} `json:"_embedded"`
// 	Links struct {
// 		Self struct {
// 			Href      string `json:"href"`
// 			Templated bool   `json:"templated"`
// 			Title     string `json:"title"`
// 		} `json:"self"`
// 	} `json:"_links"`
// 	Page struct {
// 		Size          int `json:"size"`
// 		Totalelements int `json:"totalElements"`
// 		Totalpages    int `json:"totalPages"`
// 		Number        int `json:"number"`
// 	} `json:"page"`
// }

type SchemaVersionCreateResponse struct {
	FullName    string      `json:"fullName"`
	Version  string  `json:"version"`
	SchemaVersionUid string `json:"schemaVersionUid"`
	SchemaUid        string      `json:"schemaUid"`
	// CreatedAt  string      `json:"created_at"`
	// ModifiedAt string      `json:"modified_at"`
	// CreatedBy  string `json:"created_by"`
	// ModifiedBy string      `json:"modified_by"`
	// Links      struct {
	// 	Self struct {
	// 		Href  string `json:"href"`
	// 		Title string `json:"title"`
	// 	} `json:"self"`
	// 	Scheme struct {
	// 		Href      string `json:"href"`
	// 		Templated bool   `json:"templated"`
	// 		Title     string `json:"title"`
	// 	} `json:"scheme"`
	// } `json:"_links"`
}

type SchemaVersionRequest struct {
	Schema    string     `json:"schema"`
	Version string			`json:"version"`
	Description   string     `json:"description"`
}

