package webclient

import (
	"encoding/json"
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
	OwnersIds     string `json:"ownersIds"`
	Statuses      string `json:"statuses"`
	Sort          string `json:"sort"`
	Page          int    `json:"page"`
	Size          int    `json:"size"`
}

func (c *Client) GetApplicationAccessGrantsByAttributes(data ApplicationAccessGrantAttributes) (*GetApplicationAccessGrantsByAttributeResponse, error) {
	o := GetApplicationAccessGrantsByAttributeResponse{}
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

	endpoint := fmt.Sprintf("%s/application_access_grants/search/findByAttributes?%s", c.ApiURL, values.Encode())

	err := c.RequestAndMap("GET", endpoint, nil, headers, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
