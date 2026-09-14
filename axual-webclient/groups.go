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

// GroupMemberURI returns the URI naming uid in a group's member list.
//
// Platform Manager reads the collection segment as a claim about what kind of row the
// uid points at — /users/{uid} for a person, /service-accounts/{uid} for a service
// account — and rejects the whole write with 400 when the claim is wrong. Both kinds are
// the same row underneath, so the segment is the only thing distinguishing them.
//
// The provider holds bare uids, in the configuration and in prior state alike, so there
// is nothing to infer the kind from and the API has to be asked. /service-accounts is
// the only route that answers: it is readable by any user of the tenant, unlike the rest
// of the service account API.
//
// A 404 means "write this as a person", and covers three cases that all want that
// answer: the uid is a person, the uid does not exist at all (Platform Manager then
// rejects the write exactly as it did before service accounts existed), or Platform
// Manager is old enough to have no /service-accounts route. Any other failure is
// reported rather than guessed at — treating it as a person would send /users/{uid} for
// a service account, and turn an outage into a 400 pointing at the member.
//
// See notes/service-accounts.md, "C — decisions", for the alternatives this rules out.
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
