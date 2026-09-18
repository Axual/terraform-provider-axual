package webclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

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

// GroupMemberURI returns the URI naming uid in a group's member list. A bare uid does not
// say which kind it is and Platform Manager rejects the wrong one, so it is resolved by asking.
func (c *Client) GroupMemberURI(uid string) (string, error) {
	collection := "users"
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/service-accounts/%v", c.ApiURL, uid), nil, nil, nil)
	switch {
	case err == nil:
		collection = "service-accounts"
	case errors.Is(err, NotFoundError):
		// a person, an unknown uid, or a Platform Manager without service accounts
	default:
		return "", fmt.Errorf("could not determine whether member %q is a service account: %w", uid, err)
	}
	return fmt.Sprintf("%s/%s/%v", c.ApiURL, collection, uid), nil
}
