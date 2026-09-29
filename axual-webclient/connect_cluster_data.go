package webclient

// ConnectClusterAuthorizedGroup is a group allowed to deploy to a Kafka Connect cluster.
type ConnectClusterAuthorizedGroup struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// ConnectClusterResponse is the full detail of one registered Kafka Connect cluster, from
// GET /instances/{instance_uid}/clusters/{cluster_uid}/kafka-connects/{id}. AppRole credentials and
// the Connect API password are deliberately not part of this response (ConnectClusterDetailDTO):
// they live only in the Governance Vault.
type ConnectClusterResponse struct {
	Id                       string `json:"id"`
	Name                     string `json:"name"`
	Description              string `json:"description"`
	OwnerGroupId             string `json:"ownerGroupId"`
	OwnerGroupName           string `json:"ownerGroupName"`
	ConnectUrl               string `json:"connectUrl"`
	ConnectApiAuthentication string `json:"connectApiAuthentication"`
	ConnectApiUsername       string `json:"connectApiUsername"`
	// AuthMethod is the Kafka authentication a connector deployed to this cluster must use:
	// "MTLS" or "SASL_SCRAM" (ConnectClusterAuthMethod). Not to be confused with
	// ConnectApiAuthentication, which is how Platform Manager itself talks to the Connect REST API.
	AuthMethod       string                          `json:"authMethod"`
	LogViewerUrl     string                          `json:"logViewerUrl"`
	AuthorizedGroups []ConnectClusterAuthorizedGroup `json:"authorizedGroups"`
}

// ConnectClusterInlineResponse is one entry of the paged list response
// (GET .../kafka-connects), which omits the fields only the single-item GET returns
// (logViewerUrl, connectApiAuthentication/Username, the vault config).
type ConnectClusterInlineResponse struct {
	Id               string                          `json:"id"`
	Name             string                          `json:"name"`
	Description      string                          `json:"description"`
	OwnerGroupId     string                          `json:"ownerGroupId"`
	OwnerGroupName   string                          `json:"ownerGroupName"`
	ConnectUrl       string                          `json:"connectUrl"`
	AuthMethod       string                          `json:"authMethod"`
	AuthorizedGroups []ConnectClusterAuthorizedGroup `json:"authorizedGroups"`
}

type ConnectClusterListResponse struct {
	Embedded struct {
		KafkaConnects []ConnectClusterInlineResponse `json:"kafkaConnects"`
	} `json:"_embedded"`
	Page struct {
		Number        int `json:"number"`
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
	} `json:"page"`
}
