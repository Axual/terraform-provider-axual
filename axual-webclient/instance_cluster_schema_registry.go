package webclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SchemaRegistryCredentials is how Platform Manager signs in to the schema registry. The response
// answers the password and private key as "".
type SchemaRegistryCredentials struct {
	AuthenticationType string `json:"authenticationType"`
	Username           string `json:"username,omitempty"`
	Password           string `json:"password,omitempty"`
	Certificate        string `json:"certificate,omitempty"`
	PrivateKey         string `json:"privateKey,omitempty"`
	Truststore         string `json:"truststore,omitempty"`
}

// SchemaRegistryKeycloakDetails is the Keycloak behind an Apicurio registry, used for OAuth.
type SchemaRegistryKeycloakDetails struct {
	Url           string `json:"url"`
	MasterRealm   string `json:"masterRealm"`
	AdminUsername string `json:"adminUsername"`
	AdminPassword string `json:"adminPassword,omitempty"`
	AdminClientId string `json:"adminClientId"`
	ApicurioRealm string `json:"apicurioRealm"`
}

// SchemaRegistryAuthentications lists the ways applications may sign in to an Apicurio registry.
type SchemaRegistryAuthentications struct {
	Basic bool `json:"basic"`
	OAuth bool `json:"oAuth"`
}

// InstanceClusterSchemaRegistry is the schema registry of one cluster of an Instance, in both the
// POST (create or replace) and the GET.
type InstanceClusterSchemaRegistry struct {
	Url                   string                         `json:"url"`
	SchemaRegistryType    string                         `json:"schemaRegistryType"`
	SchemaRegistryVersion *string                        `json:"schemaRegistryVersion"`
	Credentials           *SchemaRegistryCredentials     `json:"credentials"`
	KeycloakDetails       *SchemaRegistryKeycloakDetails `json:"keycloakDetails"`
	Authentications       *SchemaRegistryAuthentications `json:"authentications,omitempty"`
	Listeners             map[string]string              `json:"listeners"`
	Properties            map[string]string              `json:"properties"`
}

func (c *Client) SetInstanceClusterSchemaRegistry(instanceId string, clusterId string, req InstanceClusterSchemaRegistry) error {
	marshal, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return c.RequestAndMap("POST", fmt.Sprintf("%s/instances/%s/clusters/%s/schema-registry", c.ApiURL, instanceId, clusterId), strings.NewReader(string(marshal)), nil, nil)
}

func (c *Client) GetInstanceClusterSchemaRegistry(instanceId string, clusterId string) (*InstanceClusterSchemaRegistry, error) {
	o := InstanceClusterSchemaRegistry{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/instances/%s/clusters/%s/schema-registry", c.ApiURL, instanceId, clusterId), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteInstanceClusterSchemaRegistry(instanceId string, clusterId string) error {
	return c.RequestAndMap("DELETE", fmt.Sprintf("%s/instances/%s/clusters/%s/schema-registry", c.ApiURL, instanceId, clusterId), nil, nil, nil)
}
