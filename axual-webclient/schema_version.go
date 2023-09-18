package webclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) ValidateSchemaVersion(schema ValidateSchemaVersionRequest) (*ValidateSchemaVersionResponse, error) {
	o := ValidateSchemaVersionResponse{}
	marshal, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/schemas/check-parse", c.ApiURL), strings.NewReader(string(marshal)), headers, &o)

	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) CreateSchemaVersion(data SchemaVersionRequest) (*CreateSchemaVersionResponse, error) {
	o := CreateSchemaVersionResponse{}
	marshal, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/schemas/upload", c.ApiURL), strings.NewReader(string(marshal)), headers, &o)

	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetSchemaVersion(id string) (*GetSchemaVersionResponse, error) {
	o := GetSchemaVersionResponse{}
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/schema_versions/%v", c.ApiURL, id), nil, headers, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteSchemaVersion(id string) error {
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/schema_versions/%v", c.ApiURL, id), nil, headers, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetKeySchemaVersion(id string) (*GetSchemaVersionResponse, error) {
	o := GetSchemaVersionResponse{}
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/stream_configs/%v/keySchemaVersion", c.ApiURL, id), nil, headers, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetValueSchemaVersion(id string) (*GetSchemaVersionResponse, error) {
	o := GetSchemaVersionResponse{}
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/schema_versions/%v/valueSchemaVersion", c.ApiURL, id), nil, headers, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
