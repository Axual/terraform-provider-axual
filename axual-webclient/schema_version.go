package webclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) CreateSchemaVersion(data SchemaVersionRequest) (*SchemaVersionCreateResponse, error) {
	o := SchemaVersionCreateResponse{}
	marshal, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/schemas/upload", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)

	if err != nil {
		return nil, err
	}
	return &o, nil
}

// func (c *Client) GetSchemas() (*SchemasResponse, error) {
// 	o := SchemasResponse{}
// 	err := c.RequestAndMap("GET", fmt.Sprintf("%s/schemas", c.ApiURL), nil, nil, &o)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &o, nil
// }

// func (c *Client) GetSchema(id string) (*SchemaResponse, error) {
// 	o := SchemaResponse{}
// 	err := c.RequestAndMap("GET", fmt.Sprintf("%s/schemas/%v", c.ApiURL, id), nil, nil, &o)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &o, nil
// }

// func (c *Client) UpdateSchema(id string, data SchemaRequest) (*SchemaResponse, error) {
	
// 	o := SchemaResponse{}
// 	marshal, err := json.Marshal(data)
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/schemas/%v", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &o, nil
// }

// func (c *Client) DeleteSchema(id string) error {
// 	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/schemas/%v", c.ApiURL, id), nil, nil, nil)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }



