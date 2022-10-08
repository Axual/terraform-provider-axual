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

func (c *Client) CreateApplicationAccessGrant(applicationAccessId string, environmentId string) (string, error) {
	applicationAccessURI := fmt.Sprintf("%s/application_access/%s", c.ApiURL, applicationAccessId)
	environmentURI := fmt.Sprintf("%s/environments/%v", c.ApiURL, environmentId)

	grantRequestUrl := fmt.Sprintf("%s/grants", applicationAccessURI)
	accessGrantURI, err := c.doRequestAppGrant("POST", grantRequestUrl, strings.NewReader(environmentURI), nil)

	if err != nil {
		return "", err
	}
	//Get the Grant UID from the location header returned by the endpoint
	accessGrantId := strings.ReplaceAll(accessGrantURI, c.ApiURL+"/application_access_grants/", "")
	return accessGrantId, nil
}
