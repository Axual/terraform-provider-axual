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

func (c *Client) GetInstancesByAttributes(attributes url.Values) (*InstancesResponseByAttributes, error) {
	o := InstancesResponseByAttributes{}

	url := fmt.Sprintf("%s/instances/search/findByAttributes?%s", c.ApiURL, attributes.Encode())
	fmt.Println("URL", url)
	err := c.RequestAndMap("GET", url, nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
