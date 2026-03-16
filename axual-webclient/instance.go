package webclient

import (
	"fmt"
	"net/url"
)

func (c *Client) GetInstancesByAttributes(attributes url.Values) (*InstancesResponseByAttributes, error) {
	o := InstancesResponseByAttributes{}

	err := c.RequestAndMap("GET", fmt.Sprintf("%s/instances/search/findByAttributes?%s", c.ApiURL, attributes.Encode()), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
