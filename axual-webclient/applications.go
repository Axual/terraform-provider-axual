package webclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func (c *Client) GetApplication(id string) (*ApplicationResponse, error) {
	o := ApplicationResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/applications/%v", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateApplication(id string, data ApplicationRequest) (*ApplicationResponse, error) {
	o := ApplicationResponse{}
	marshal, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/applications/%v", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteApplication(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/applications/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateApplication(data ApplicationRequest) (*ApplicationResponse, error) {
	o := ApplicationResponse{}
	marshal, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("POST", fmt.Sprintf("%s/applications", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
func (c *Client) GetApplicationByName(name string) (*ApplicationByNameResponse, error) {
	o := ApplicationByNameResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/applications/search/findByName?name=%s", c.ApiURL, url.QueryEscape(name)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}