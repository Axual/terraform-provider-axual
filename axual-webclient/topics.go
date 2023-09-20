package webclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func (c *Client) ReadTopic(id string) (*TopicResponse, error) {
	o := TopicResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/streams/%s", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) CreateTopic(topic TopicRequest) (*TopicResponse, error) {
	o := TopicResponse{}
	marshal, err := json.Marshal(topic)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/streams", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateTopic(id string, TopicRequest TopicRequest) (*TopicResponse, error) {
	o := TopicResponse{}
	marshal, err := json.Marshal(TopicRequest)
	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/streams/%v", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteTopic(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/streams/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) ReadTopicByName(name string) (*TopicsByNameResponse, error) {
	o := TopicsByNameResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/streams/search/findByName?name=%s", c.ApiURL, url.QueryEscape(name)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}