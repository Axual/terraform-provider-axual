package webclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (c *Client) ReadTopicConfig(id string) (*TopicConfigResponse, error) {
	o := TopicConfigResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/stream_configs/%s", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}

	keySchemaVersion, err := c.GetKeySchemaVersion(id)
	if err == nil {
		o.KeySchemaVersion = keySchemaVersion.Id
	} else if !errors.Is(err, NotFoundError) {
		return nil, err
	}

	valueSchemaVersion, err := c.GetValueSchemaVersion(id)
	if err == nil {
		o.ValueSchemaVersion = valueSchemaVersion.Id
	} else if !errors.Is(err, NotFoundError) {
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

func (c *Client) GetTopicConfigPermissions(topicConfigID string, permType string) ([]PermissionResponse, error) {
	var perms []PermissionResponse
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/stream_configs/%s/permissions?type=%s", c.ApiURL, topicConfigID, permType), nil, nil, &perms)
	if err != nil {
		return nil, err
	}
	return perms, nil
}

func (c *Client) DeleteTopicConfigPermissions(topicConfigID string, request PermissionRequest) error {
	marshal, err := json.Marshal(request)
	if err != nil {
		return err
	}
	headers := map[string]string{"Content-Type": "application/json"}
	err = c.RequestAndMap("DELETE", fmt.Sprintf("%s/stream_configs/%s/permissions?type=browse", c.ApiURL, topicConfigID), strings.NewReader(string(marshal)), headers, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) AddTopicConfigPermissions(topicConfigID string, request PermissionRequest) error {
	marshal, err := json.Marshal(request)
	if err != nil {
		return err
	}
	headers := map[string]string{"Content-Type": "application/json"}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/stream_configs/%s/permissions", c.ApiURL, topicConfigID), strings.NewReader(string(marshal)), headers, nil)
	if err != nil {
		return err
	}
	return nil
}
