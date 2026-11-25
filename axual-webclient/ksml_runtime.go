package webclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) CreateKsmlRuntime(req KsmlRuntimeCreateRequest) (*KsmlRuntimeResponse, error) {
	o := KsmlRuntimeResponse{}
	req.Type = KsmlRuntimeType
	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/runtimes", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetKsmlRuntime(id string) (*KsmlRuntimeResponse, error) {
	o := KsmlRuntimeResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/runtimes/%s", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateKsmlRuntime(id string, req KsmlRuntimeUpdateRequest) (*KsmlRuntimeResponse, error) {
	o := KsmlRuntimeResponse{}
	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/runtimes/%s", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteKsmlRuntime(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/runtimes/%s", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
