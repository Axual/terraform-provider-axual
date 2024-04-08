package webclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func (c *Client) GetApplicationDeployment(id string) (*ApplicationDeploymentResponse, error) {
	o := ApplicationDeploymentResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/application_deployments/%v", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteApplicationDeployment(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/application_deployments/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateApplicationDeployment(applicationDeploymentRequest ApplicationDeploymentCreateRequest) (ApplicationDeploymentCreateResponse, error) {
	var o ApplicationDeploymentCreateResponse
	marshal, err := json.Marshal(applicationDeploymentRequest)
	if err != nil {
		return "Error creating payload for application deployment", err
	}
	headers := map[string]string{"Content-Type": "application/json"}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/application_deployments", c.ApiURL), strings.NewReader(string(marshal)), headers, &o)
	if err != nil {
		return "Error sending POST request for application deployment", err
	}
	return o, nil
}

func (c *Client) UpdateApplicationDeployment(id string, data ApplicationDeploymentUpdateRequest) (ApplicationDeploymentUpdateResponse, error) {
	var o ApplicationDeploymentUpdateResponse
	marshal, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{"Content-Type": "application/json"}
	err = c.RequestAndMap("PUT", fmt.Sprintf("%s/application_deployments/%v", c.ApiURL, id), strings.NewReader(string(marshal)), headers, &o)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (c *Client) FindApplicationDeploymentByApplicationAndEnvironment(application string, environment string) (*ApplicationDeploymentFindByApplicationAndEnvironmentResponse, error) {
	o := ApplicationDeploymentFindByApplicationAndEnvironmentResponse{}

	err :=
		c.RequestAndMap("GET", fmt.Sprintf("%s/application_deployments/search/findByApplicationAndEnvironment?application=%v&environment=%v",
			c.ApiURL, url.QueryEscape(application), url.QueryEscape(environment)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) OperateApplicationDeployment(id string, action string, data ApplicationDeploymentOperationRequest) error {
	marshal, err := json.Marshal(data)
	if err != nil {
		return err
	}
	headers := map[string]string{"Content-Type": "application/json"}
	err = c.RequestAndMap("PUT", fmt.Sprintf("%s/application_deployments/%v/operation?action=%s", c.ApiURL, id, action), strings.NewReader(string(marshal)), headers, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetApplicationDeploymentStatus(id string) (*ApplicationDeploymentStatusResponse, error) {
	o := ApplicationDeploymentStatusResponse{}
	headers := map[string]string{"Content-Type": "application/json"}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/application_deployments/%v/status", c.ApiURL, id), nil, headers, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
