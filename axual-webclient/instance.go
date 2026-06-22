package webclient

import (
	"fmt"
	"net/url"
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
