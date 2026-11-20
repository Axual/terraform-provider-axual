package webclient

// Keys of a Cluster's providerInfo. Platform Manager stores this map in Vault, not in its database,
// and answers the secret keys (private key and passwords) as "".
const (
	ProviderInfoProviderType         = "Axl-Provider-Type"
	ProviderInfoBootstrapServers     = "Axl-Bootstrap-Server-ApacheKafka"
	ProviderInfoBrokerProtocol       = "Axl-Broker-Protocol-Type-ApacheKafka"
	ProviderInfoBrokerSecurity       = "Axl-Broker-Security-Protocol-Type-ApacheKafka"
	ProviderInfoBrokerCertificate    = "Axl-Broker-Cert-ApacheKafka"
	ProviderInfoBrokerPrivateKey     = "Axl-Broker-Private-Key-ApacheKafka"
	ProviderInfoBrokerUsername       = "Axl-Broker-Username-ApacheKafka"
	ProviderInfoBrokerPassword       = "Axl-Broker-Password-ApacheKafka"
	ProviderInfoBrokerTruststore     = "Axl-Broker-Truststore-ApacheKafka"
	ProviderInfoBrokerCAValidation   = "Axl-Broker-CA-Validation-ApacheKafka"
	ProviderInfoStrimziProvisionerId = "Axl-Strimzi-Provisioner-Username"
	ProviderInfoStrimziProvisionerPw = "Axl-Strimzi-Provisioner-Password"
)

// Keys of a Cluster's settings map.
const (
	ClusterSettingSslAuthMode           = "sslAuthMode"
	ClusterSettingOauthPrincipalsPrefix = "oauthPrincipalsPrefix"
	ClusterSettingStrimziClusterName    = "strimziClusterName"
	ClusterSettingStrimziNamespace      = "strimziNamespace"
	ClusterSettingStrimziProvisionerUrl = "strimziProvisionerUrl"
)

type ClusterResponse struct {
	Uid                           string            `json:"uid"`
	Name                          string            `json:"name"`
	Description                   string            `json:"description"`
	Location                      string            `json:"location"`
	BillingCloudEnabled           bool              `json:"billingCloudEnabled"`
	ProviderType                  string            `json:"providerType"`
	AuthenticationInfo            map[string]string `json:"authenticationInfo"`
	TopicPattern                  string            `json:"topicPattern"`
	GroupPattern                  string            `json:"groupPattern"`
	TransactionalPattern          string            `json:"transactionalPattern"`
	SharedCluster                 bool              `json:"sharedCluster"`
	KafkaVersion                  string            `json:"kafkaVersion"`
	KafkaListeners                map[string]string `json:"kafkaListeners"`
	BootstrapUrlsByAuthentication map[string]string `json:"bootstrapUrlsByAuthentication"`
	Settings                      map[string]string `json:"settings"`
}

// ClusterRequest is the body of the create POST and the update PATCH. The nullable strings are
// always sent: the PATCH only changes a field that is present, so clearing one needs an explicit
// null. The maps replace the stored map when present.
type ClusterRequest struct {
	Name                          string            `json:"name,omitempty"`
	Description                   *string           `json:"description"`
	Location                      *string           `json:"location"`
	BillingCloudEnabled           bool              `json:"billingCloudEnabled"`
	ProviderType                  string            `json:"providerType,omitempty"`
	SharedCluster                 *bool             `json:"sharedCluster,omitempty"`
	ProviderInfo                  map[string]string `json:"providerInfo"`
	TopicPattern                  *string           `json:"topicPattern,omitempty"`
	GroupPattern                  *string           `json:"groupPattern,omitempty"`
	TransactionalPattern          *string           `json:"transactionalPattern,omitempty"`
	KafkaVersion                  *string           `json:"kafkaVersion"`
	KafkaListeners                map[string]string `json:"kafkaListeners"`
	BootstrapUrlsByAuthentication map[string]string `json:"bootstrapUrlsByAuthentication"`
	Settings                      map[string]string `json:"settings"`
}

type ClustersResponse struct {
	Embedded struct {
		Clusters []ClusterResponse `json:"clusters"`
	} `json:"_embedded"`
}
