package webclient

import (
	"fmt"
	"net/url"
)

func (c *Client) GetInstanceByNameOrShortName(params url.Values) (*InstanceResponse, error) {
	o := InstanceResponse{}
	endpoint := fmt.Sprintf("findByName?name=%s", url.QueryEscape(params.Get("name")))
	if params.Get("shortName") != "" {
		endpoint = fmt.Sprintf("findByShortName?shortName=%s", url.QueryEscape(params.Get("shortName")))
	}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/instances/search/%s", c.ApiURL, endpoint), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
