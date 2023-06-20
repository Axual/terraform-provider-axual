package webclient

import (
	"encoding/json"
	"fmt"
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

func (c *Client) DeleteApplicationAccessGrant(applicationAccessId string, environment string) error {

	revoke := ApplicationAccessGrantRevoke{
		Reason:      "Revoked from Terraform Provider",
		Environment: fmt.Sprintf("%s/environments/%v", c.ApiURL, environment),
	}
	marshal, err1 := json.Marshal(revoke)
	if err1 != nil {
		return err1
	}

	deleteUrl := fmt.Sprintf("%s/application_access/%v/grants", c.ApiURL, applicationAccessId)

	h := make(map[string]string)
	h["content-type"] = "application/json"

	err := c.RequestAndMap("DELETE", deleteUrl, strings.NewReader(string(marshal)), h, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateApplicationAccessGrant(data ApplicationAccessGrantRequest) (*ApplicationAccessGrantResponse, error) {
	o := ApplicationAccessGrantResponse{}
	marshal, err := json.Marshal(data)

	if err != nil {
		return nil, err
	}

	err = c.RequestAndMapGrant("POST", fmt.Sprintf("%s/application_access_grants", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
