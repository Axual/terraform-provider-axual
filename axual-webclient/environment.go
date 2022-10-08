package webclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) CreateEnvironment(env EnvironmentRequest) (*EnvironmentResponse, error) {
	o := EnvironmentResponse{}
	marshal, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/environments", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) ReadEnvironment(id string) (*EnvironmentResponse, error) {
	o := EnvironmentResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/environments/%s", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateEnvironment(id string, env map[string]interface{}) (*EnvironmentResponse, error) {
	o := EnvironmentResponse{}
	marshal, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/environments/%v", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteEnvironment(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/environments/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) ReadEnvironments() (*EnvironmentsResponse, error) {
	o := EnvironmentsResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/environments/", c.ApiURL), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
