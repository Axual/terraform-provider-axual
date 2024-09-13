package webclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func (c *Client) ReadApplicationPrincipal(id string) (*ApplicationPrincipalResponse, error) {
	o := ApplicationPrincipalResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/application_principals/%s", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) CreateApplicationPrincipal(applicationPrincipalRequest [1]ApplicationPrincipalRequest) (ApplicationPrincipalCreateResponse, error) {
	var o ApplicationPrincipalCreateResponse
	marshal, err := json.Marshal(applicationPrincipalRequest)
	if err != nil {
		return "Error creating payload for application principal", err
	}
	headers := map[string]string{"Content-Type": "application/json"}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/application_principals", c.ApiURL), strings.NewReader(string(marshal)), headers, &o)
	if err != nil {
		return "Error sending POST request for application principal", err
	}
	time.Sleep(5 * time.Second) // Principal application can take significant time to apply in Kafka cluster
	return o, nil
}

func (c *Client) UpdateApplicationPrincipal(id string, applicationUpdatePrincipalRequest ApplicationPrincipalUpdateRequest) (ApplicationPrincipalUpdateResponse, error) {
	var o ApplicationPrincipalUpdateResponse
	marshal, err := json.Marshal(applicationUpdatePrincipalRequest)
	if err != nil {
		return "Error creating payload for application principal", err
	}
	headers := map[string]string{"Content-Type": "application/json"}
	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/application_principals/%v", c.ApiURL, id), strings.NewReader(string(marshal)), headers, &o)
	if err != nil {
		return "Error sending PATCH request for application principal", err
	}
	return o, nil
}

func (c *Client) DeleteApplicationPrincipal(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/application_principals/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) FindApplicationPrincipalByApplicationAndEnvironment(application string, environment string) (*ApplicationPrincipalFindByApplicationAndEnvironmentResponse, error) {
	o := ApplicationPrincipalFindByApplicationAndEnvironmentResponse{}

	err :=
		c.RequestAndMap("GET", fmt.Sprintf("%s/application_principals/search/findByApplicationAndEnvironment?application=%v&environment=%v",
			c.ApiURL, url.QueryEscape(application), url.QueryEscape(environment)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
