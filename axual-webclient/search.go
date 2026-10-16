package webclient

import (
	"fmt"
	"net/url"
)

// TopicConfigsResponse is the list a topic config search answers.
type TopicConfigsResponse struct {
	Embedded struct {
		TopicConfigs []TopicConfigResponse `json:"stream_configs"`
	} `json:"_embedded"`
}

// FindTopicConfigsByTopic lists the topic configs of the topic at topicUrl.
func (c *Client) FindTopicConfigsByTopic(topicUrl string) (*TopicConfigsResponse, error) {
	o := TopicConfigsResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/stream_configs/search/findByStream?stream=%s", c.ApiURL, url.QueryEscape(topicUrl)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
