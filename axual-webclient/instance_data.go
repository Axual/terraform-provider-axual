package webclient

type InstanceResponse struct {
	Name                            string                         `json:"name"`
	ShortName                       string                         `json:"shortName"`
	Description                     string                         `json:"description"`
	Uid                             string                         `json:"uid"`
	Properties                      map[string]string              `json:"properties"`
	InstanceClusters                []InstanceClusterResponse      `json:"instanceClusters"`
	CaCerts                         []InstanceCertificate          `json:"caCerts"`
	EnabledAuthenticationMethods    []InstanceAuthenticationMethod `json:"enabledAuthenticationMethods"`
	ConnectEnabled                  bool                           `json:"connectEnabled"`
	FlinkEnabled                    bool                           `json:"flinkEnabled"`
	ConnectUrls                     string                         `json:"connectUrls"`
	GranularBrowsePermission        bool                           `json:"granularBrowsePermission"`
	TopicOwnerBrowsePermission      bool                           `json:"topicOwnerBrowsePermission"`
	SelfApproval                    bool                           `json:"selfApproval"`
	EnvironmentMapping              bool                           `json:"environmentMapping"`
	CertificateReusePreventionLevel string                         `json:"certificateReusePreventionLevel"`
}

// InstanceClusterResponse is one cluster of an Instance. The cluster is embedded in full.
type InstanceClusterResponse struct {
	Cluster            ClusterResponse `json:"cluster"`
	SchemaRegistryType string          `json:"schemaRegistryType"`
}

type InstanceCertificate struct {
	Pem string `json:"pem"`
}

// InstanceAuthenticationMethod is one way clients may authenticate to the Instance's clusters. The
// response also has a rank, which Platform Manager copies from the tenant.
type InstanceAuthenticationMethod struct {
	Protocol  string `json:"protocol"`
	Mechanism string `json:"mechanism"`
}

// InstanceClusterLink is one entry of instanceClusters in a request: the cluster's URL.
type InstanceClusterLink struct {
	Cluster string `json:"cluster"`
}

// InstanceRequest is the body of the create POST and the update PATCH. The PATCH only changes a
// field that is present, and instanceClusters, caCerts, properties and the authentication methods
// replace the stored value as a whole. Description and connectUrls are always sent, so a removed
// value is cleared with an explicit null.
type InstanceRequest struct {
	Name                            string                         `json:"name,omitempty"`
	ShortName                       string                         `json:"shortName,omitempty"`
	Description                     *string                        `json:"description"`
	InstanceClusters                []InstanceClusterLink          `json:"instanceClusters"`
	CaCerts                         []InstanceCertificate          `json:"caCerts"`
	Properties                      map[string]string              `json:"properties"`
	EnabledAuthenticationMethods    []InstanceAuthenticationMethod `json:"enabledAuthenticationMethods,omitempty"`
	ConnectEnabled                  bool                           `json:"connectEnabled"`
	FlinkEnabled                    bool                           `json:"flinkEnabled"`
	ConnectUrls                     *string                        `json:"connectUrls"`
	GranularBrowsePermission        bool                           `json:"granularBrowsePermission"`
	TopicOwnerBrowsePermission      bool                           `json:"topicOwnerBrowsePermission"`
	EnvironmentMapping              bool                           `json:"environmentMapping"`
	SelfApproval                    bool                           `json:"selfApproval"`
	CertificateReusePreventionLevel string                         `json:"certificateReusePreventionLevel,omitempty"`
}
