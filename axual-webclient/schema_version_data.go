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
	Id         string `json:"uid"`
	Version    string `json:"version"`
	SchemaBody string `json:"schemaBody"`
}

type SchemaVersionRequest struct {
	Schema      string `json:"schema"`
	Version     string `json:"version"`
	Description string `json:"description"`
}
