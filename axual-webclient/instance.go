package webclient

import (
	"fmt"
	"net/url"
)

func (c *Client) GetInstance(id string) (*InstanceResponse, error) {
	o := InstanceResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/instances/%v", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetInstanceByName(name string) (*InstanceResponse, error) {
	o := InstanceResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/instances/search/findByName?name=%v", c.ApiURL, url.QueryEscape(name)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
