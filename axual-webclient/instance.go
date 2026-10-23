package webclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func (c *Client) GetInstanceByName(name string) (*InstanceResponse, error) {
	o := InstanceResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/instances/search/findByName?name=%s", c.ApiURL, url.QueryEscape(name)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetInstanceByShortName(shortName string) (*InstanceResponse, error) {
	o := InstanceResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/instances/search/findByShortName?shortName=%s", c.ApiURL, url.QueryEscape(shortName)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) CreateInstance(req InstanceRequest) (*InstanceResponse, error) {
	o := InstanceResponse{}
	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/instances", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetInstance(id string) (*InstanceResponse, error) {
	o := InstanceResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/instances/%s", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateInstance(id string, req InstanceRequest) (*InstanceResponse, error) {
	o := InstanceResponse{}
	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/instances/%s", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteInstance(id string) error {
	return c.RequestAndMap("DELETE", fmt.Sprintf("%s/instances/%s", c.ApiURL, id), nil, nil, nil)
}
