package webclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) ReadStream(id string) (*StreamResponse, error) {
	o := StreamResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/streams/%s", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) CreateStream(stream StreamRequest) (*StreamResponse, error) {
	o := StreamResponse{}
	marshal, err := json.Marshal(stream)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/streams", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateStream(id string, streamRequest StreamRequest) (*StreamResponse, error) {
	o := StreamResponse{}
	marshal, err := json.Marshal(streamRequest)
	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/streams/%v", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteStream(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/streams/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
