package webclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"
)

func (c *Client) GetApplicationAccessGrant(id string) (*ApplicationAccessGrant, error) {
	o := ApplicationAccessGrant{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/application_access_grants/%v", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) CreateApplicationAccessGrant(data ApplicationAccessGrantRequest) (*ApplicationAccessGrantResponse, error) {
	h := make(map[string]string)
	h["accept"] = "application/json, text/plain, */*"
	h["content-type"] = "application/json"

	o := ApplicationAccessGrantResponse{}
	marshal, err := json.Marshal(data)

	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("POST", fmt.Sprintf("%s/application_access_grants", c.ApiURL), strings.NewReader(string(marshal)), h, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) ApproveGrant(applicationAccessGrantId string) error {
	h := make(map[string]string)
	h["accept"] = "application/json, text/plain, */*"
	h["content-type"] = "application/json"
	err := c.RequestAndMap("PUT", fmt.Sprintf("%s/application_access_grants/%v", c.ApiURL, applicationAccessGrantId), nil, h, nil)
	if err != nil {

		return err
	}
	time.Sleep(10 * time.Second) // Grant approval can take significant time
	return nil
}

func (c *Client) CancelGrant(applicationAccessGrantId string) error {
	h := make(map[string]string)
	h["accept"] = "application/json, text/plain, */*"
	h["content-type"] = "application/json"
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/application_access_grants/%v", c.ApiURL, applicationAccessGrantId), nil, h, nil)
	if err != nil {
		return err
	}
	time.Sleep(10 * time.Second) //to give time for Connect/Kafka to propagate changes
	return nil
}

func (c *Client) RevokeOrDenyGrant(applicationAccessGrantId string, reason string) error {
	h := make(map[string]string)
	h["accept"] = "application/json, text/plain, */*"
	h["content-type"] = "application/json"

	o := map[string]string{"reason": reason}

	marshal, err1 := json.Marshal(o)
	if err1 != nil {
		return err1
	}

	err := c.RequestAndMap("POST", fmt.Sprintf("%s/application_access_grants/%v/deny", c.ApiURL, applicationAccessGrantId), strings.NewReader(string(marshal)), h, nil)
	if err != nil {
		return err
	}
	return nil
}

type ApplicationAccessGrantAttributes struct {
	TopicId       string `json:"streamId"`
	ApplicationId string `json:"applicationId"`
	EnvironmentId string `json:"environmentId"`
	AccessType    string `json:"accessType"`
	Statuses      string `json:"statuses"`
	Size          int    `json:"size"`
}

// The grant searches, relative to the API URL. findSummariesByAttributes exists from Platform
// Manager 17.0.0. findByAttributes is deprecated in 17.0.0 and removed in 2027.4: then remove
// oldGrantSearchPath and grantSearch, and require Platform Manager 17.0.0 in the compatibility table.
const (
	grantSearchPath    = "application_access_grants/search/findSummariesByAttributes"
	oldGrantSearchPath = "application_access_grants/search/findByAttributes"
)

// grantSearch calls search with grantSearchPath. On a 404 (Platform Manager older than 17.0.0) it
// calls search with oldGrantSearchPath, now and for every later search of this client.
// findSummariesByAttributes answers an empty page, not 404, when no grant matches.
func (c *Client) grantSearch(search func(path string) error) error {
	if !c.oldGrantSearch.Load() {
		err := search(grantSearchPath)
		if !errors.Is(err, NotFoundError) {
			return err
		}
		c.oldGrantSearch.Store(true)
	}
	return search(oldGrantSearchPath)
}

func (c *Client) GetApplicationAccessGrantsByAttributes(data ApplicationAccessGrantAttributes) (*GetApplicationAccessGrantsByAttributeResponse, error) {
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/hal+json",
	}
	values := url.Values{}
	v := reflect.ValueOf(data)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		name := t.Field(i).Tag.Get("json")
		if field.Kind() == reflect.Int && field.Int() != 0 {
			values.Add(name, fmt.Sprint(field.Int()))
		}
		if field.Kind() == reflect.String && field.String() != "" {
			values.Add(name, field.String())
		}
	}

	var o GetApplicationAccessGrantsByAttributeResponse
	err := c.grantSearch(func(path string) error {
		o = GetApplicationAccessGrantsByAttributeResponse{}
		return c.RequestAndMap("GET", fmt.Sprintf("%s/%s?%s", c.ApiURL, path, values.Encode()), nil, headers, &o)
	})
	if err != nil {
		return nil, err
	}
	for i := range o.Embedded.ApplicationAccessGrantResponses {
		o.Embedded.ApplicationAccessGrantResponses[i].useEmbedded()
	}
	return &o, nil
}

// ListApplicationAccessGrants returns every grant that matches params, from all pages. See ListAll.
// ListItem.Embedded reads both the findSummariesByAttributes and the findByAttributes answer.
func (c *Client) ListApplicationAccessGrants(params url.Values) ([]ListItem, error) {
	var items []ListItem
	err := c.grantSearch(func(path string) error {
		var err error
		items, err = c.ListAll(path, params)
		return err
	})
	return items, err
}
