package webclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func (c *Client) GetGroups() (*GroupsResponse, error) {
	o := GroupsResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/groups", c.ApiURL), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) CreateGroup(group GroupRequest) (*GroupResponse, error) {
	o := GroupResponse{}
	marshal, err := json.Marshal(group)
	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("POST", fmt.Sprintf("%s/groups", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetGroup(id string) (*GroupResponse, error) {
	o := GroupResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/groups/%v", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateGroup(id string, group GroupRequest) (*GroupResponse, error) {
	o := GroupResponse{}
	marshal, err := json.Marshal(group)
	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/groups/%v", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteGroup(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/groups/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetGroupByName(name string) (*GetGroupByNameResponse, error) {
	o := GetGroupByNameResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/groups/search/findByName?name=%v", c.ApiURL, url.QueryEscape(name)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
