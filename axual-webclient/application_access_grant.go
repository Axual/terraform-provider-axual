package webclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func (c *Client) GetApplicationAccessGrant(id string) (*ApplicationAccessGrant, error) {
	o := ApplicationAccessGrant{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/application_access_grants/%v", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) CreateApplicationAccessGrant(data ApplicationAccessGrantRequest) (*ApplicationAccessGrantResponse, error) {
	h := make(map[string]string)
	h["accept"] = "application/json, text/plain, */*"
	h["content-type"] = "application/json"

	o := ApplicationAccessGrantResponse{}
	marshal, err := json.Marshal(data)

	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("POST", fmt.Sprintf("%s/application_access_grants", c.ApiURL), strings.NewReader(string(marshal)), h, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) ApproveGrant(applicationAccessGrantId string) error {
	h := make(map[string]string)
	h["accept"] = "application/json, text/plain, */*"
	h["content-type"] = "application/json"
	err := c.RequestAndMap("PUT", fmt.Sprintf("%s/application_access_grants/%v", c.ApiURL, applicationAccessGrantId), nil, h, nil)
	if err != nil {

		return err
	}
	return nil
}

func (c *Client) CancelGrant(applicationAccessGrantId string) error {
	h := make(map[string]string)
	h["accept"] = "application/json, text/plain, */*"
	h["content-type"] = "application/json"
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/application_access_grants/%v", c.ApiURL, applicationAccessGrantId), nil, h, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) RevokeOrDenyGrant(applicationAccessGrantId string, reason string) error {
	h := make(map[string]string)
	h["accept"] = "application/json, text/plain, */*"
	h["content-type"] = "application/json"

	o := map[string]string{"reason": reason}

	marshal, err1 := json.Marshal(o)
	if err1 != nil {
		return err1
	}

	err := c.RequestAndMap("POST", fmt.Sprintf("%s/application_access_grants/%v/deny", c.ApiURL, applicationAccessGrantId), strings.NewReader(string(marshal)), h, nil)
	if err != nil {
		return err
	}
	return nil
}

type ApplicationAccessGrantAttributes struct {
	TopicId string
	ApplicationId string
	EnvironmentId string
	AccessType string
}
func (c *Client) GetApplicationAccessGrantsByAttributes(data ApplicationAccessGrantAttributes) (*GetApplicationAccessGrantsByAttributeResponse, error) {
	o := GetApplicationAccessGrantsByAttributeResponse{}
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/hal+json",
	}

	values := url.Values{}
	values.Add("streamId", data.TopicId)
	values.Add("applicationId", data.ApplicationId)
	values.Add("environmentId", data.EnvironmentId)
	values.Add("accessType", data.AccessType)

	endpoint := fmt.Sprintf("%s/application_access_grants/search/findByAttributes?%s", c.ApiURL, values.Encode())
	err := c.RequestAndMap("GET", endpoint, nil, headers, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}