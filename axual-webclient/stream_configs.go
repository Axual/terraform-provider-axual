package webclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) ReadStreamConfig(id string) (*StreamConfigResponse, error) {
	o := StreamConfigResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/stream_configs/%s", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) CreateStreamConfig(stream StreamConfigRequest) (*StreamConfigResponse, error) {
	o := StreamConfigResponse{}
	marshal, err := json.Marshal(stream)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/stream_configs", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateStreamConfig(id string, streamRequest StreamConfigRequest) (*StreamConfigResponse, error) {
	o := StreamConfigResponse{}
	marshal, err := json.Marshal(streamRequest)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/stream_configs/%v", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	fmt.Println("UPDATE STREAM CONFIG RESPONSE", &o)
	return &o, nil
}

func (c *Client) DeleteStreamConfig(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/stream_configs/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
