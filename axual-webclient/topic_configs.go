package webclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) ReadTopicConfig(id string) (*TopicConfigResponse, error) {
	o := TopicConfigResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/stream_configs/%s", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) CreateTopicConfig(topic TopicConfigRequest) (*TopicConfigResponse, error) {
	o := TopicConfigResponse{}
	marshal, err := json.Marshal(topic)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/stream_configs", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateTopicConfig(id string, topicRequest TopicConfigRequest) (*TopicConfigResponse, error) {
	o := TopicConfigResponse{}
	marshal, err := json.Marshal(topicRequest)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/stream_configs/%v", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	fmt.Println("UPDATE TOPIC CONFIG RESPONSE", &o)
	return &o, nil
}

func (c *Client) DeleteTopicConfig(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/stream_configs/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
