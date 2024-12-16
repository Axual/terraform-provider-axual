package webclient

type SchemaType struct {
	SchemaId    string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OptLock     int64  `json:"optLock"`
	Uid         string `json:"uid"`
	CreatedAt   string `json:"created_at"`
	ModifiedAt  string `json:"modified_at"`
	CreatedBy   string `json:"created_by"`
	ModifiedBy  string `json:"modified_by"`
	Owners      *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"owners"`
}

type GetSchemaVersionResponse struct {
	Id                string     `json:"id"`
	Version           string     `json:"version"`
	SchemaBody        string     `json:"schemaBody"`
	Schema            SchemaType `json:"schema"`
	CreatedByFullName string     `json:"createdByFullName"`
	CreatedAt         string     `json:"createdAt"`
	ModifiedAt        string     `json:"modifiedAt"`
}

type CreateSchemaVersionResponse struct {
	Id       string `json:"schemaVersionUid"`
	SchemaId string `json:"schemaUid"`
	Version  string `json:"version"`
	FullName string `json:"fullName"`
	Owners   *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"owners"`
}

type ValidateSchemaVersionResponse struct {
	Schema   string   `json:"schema"`
	Versions []string `json:"version"`
	FullName string   `json:"fullName"`
}

type ValidateSchemaVersionRequest struct {
	Schema string `json:"schema"`
}

type SchemaVersionResponse struct {
	Id       string `json:"schemaVersionUid"`
	SchemaId string `json:"schemaUid"`
	Version  string `json:"version"`
	FullName string `json:"fullName"`
	Owners   *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"owners"`
}

type SchemaVersionRequest struct {
	Schema      string  `json:"schema"`
	Version     string  `json:"version"`
	Description string  `json:"description"`
	Owners      *string `json:"owners"`
}

type GetSchemaVersionsResponse struct {
	Embedded struct {
		SchemaVersion []struct {
			Version    string `json:"version"`
			SchemaBody string `json:"schemaBody"`
			Uid        string `json:"uid"`
			Embedded   struct {
				Schema struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					Uid         string `json:"uid"`
					ModifiedBy  string `json:"modified_by"`
					CreatedAt   string `json:"created_at"`
					CreatedBy   string `json:"created_by"`
					ModifiedAt  string `json:"modified_at"`
					Owners      *struct {
						UID  string `json:"uid"`
						Name string `json:"name"`
					} `json:"owners"`
					Links struct {
						Self struct {
							Href      string `json:"href"`
							Templated bool   `json:"templated"`
							Title     string `json:"title"`
						} `json:"self"`
					} `json:"_links"`
				} `json:"schema"`
			} `json:"_embedded"`
		} `json:"schema_versions"`
	} `json:"_embedded"`
}
type GetSchemaByNameResponse struct {
	Embedded struct {
		Schemas []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Uid         string `json:"uid"`
			ModifiedBy  string `json:"modified_by"`
			CreatedAt   string `json:"created_at"`
			CreatedBy   string `json:"created_by"`
			ModifiedAt  string `json:"modified_at"`
			Links       struct {
				Self struct {
					Href  string `json:"href"`
					Title string `json:"title"`
				} `json:"self"`
				Schema struct {
					Href      string `json:"href"`
					Templated bool   `json:"templated"`
					Title     string `json:"title"`
				} `json:"schema"`
			} `json:"_links"`
		} `json:"schemas"`
	} `json:"_embedded"`
	Links struct {
		Self struct {
			Href  string `json:"href"`
			Title string `json:"title"`
		} `json:"self"`
	} `json:"_links"`
	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}
