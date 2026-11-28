package webclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TenantAuthenticationMethod is one login method that the Instances of a tenant may use. A lower rank comes first.
type TenantAuthenticationMethod struct {
	Rank      int    `json:"rank"`
	Protocol  string `json:"protocol"`
	Mechanism string `json:"mechanism"`
}

// TenantResponse is the tenant of the login, as GET /tenant returns it.
type TenantResponse struct {
	Uid                            string                       `json:"uid"`
	Name                           string                       `json:"name"`
	ShortName                      string                       `json:"shortName"`
	Settings                       map[string]*string           `json:"settings"`
	SupportedAuthenticationMethods []TenantAuthenticationMethod `json:"supportedAuthenticationMethods"`
	WizardCompleted                bool                         `json:"wizardCompleted"`
	UniqueConsumerGroup            bool                         `json:"uniqueConsumerGroup"`
	DataClassificationEnabled      bool                         `json:"dataClassificationEnabled"`
}

// TenantSettingsPatch is the body of PATCH /tenants/{id}. Platform Manager merges settings: a key that is
// not sent keeps its value. supportedAuthenticationMethods replaces the whole list. Nil fields are not sent.
type TenantSettingsPatch struct {
	Settings                       map[string]string            `json:"settings,omitempty"`
	SupportedAuthenticationMethods []TenantAuthenticationMethod `json:"supportedAuthenticationMethods,omitempty"`
}

// GroupManagementSettings is the body of POST /tenant/settings/group-management. A nil block is not sent and
// Platform Manager keeps its values.
type GroupManagementSettings struct {
	IamGroupManagement   *IamGroupManagement   `json:"iamGroupManagement,omitempty"`
	AxualGroupManagement *AxualGroupManagement `json:"axualGroupManagement,omitempty"`
}

type IamGroupManagement struct {
	Enabled                       bool  `json:"enabled"`
	AutomaticGroupCreationEnabled *bool `json:"automaticGroupCreationEnabled,omitempty"`
}

type AxualGroupManagement struct {
	Enabled bool `json:"enabled"`
}

// GetTenant returns the tenant of the login.
func (c *Client) GetTenant() (*TenantResponse, error) {
	o := TenantResponse{}
	if err := c.RequestAndMap("GET", fmt.Sprintf("%s/tenant", c.ApiURL), nil, nil, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

// PatchTenantSettings changes the settings and login methods of tenant id.
func (c *Client) PatchTenantSettings(id string, patch TenantSettingsPatch) error {
	body, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	return c.RequestAndMap("PATCH", fmt.Sprintf("%s/tenants/%s", c.ApiURL, id), strings.NewReader(string(body)), nil, nil)
}

// UpdateGroupManagementSettings changes how the groups of the tenant of the login are managed.
func (c *Client) UpdateGroupManagementSettings(settings GroupManagementSettings) error {
	body, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return c.RequestAndMap("POST", fmt.Sprintf("%s/tenant/settings/group-management", c.ApiURL), strings.NewReader(string(body)), nil, nil)
}
