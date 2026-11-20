package webclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) CreateKsmlRuntime(instanceId string, clusterId string, req KsmlRuntimeRequest) (*KsmlRuntimeResponse, error) {
	o := KsmlRuntimeResponse{}
	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/instances/%s/clusters/%s/ksml-runtimes", c.ApiURL, instanceId, clusterId), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetKsmlRuntime(instanceId string, clusterId string, id string) (*KsmlRuntimeResponse, error) {
	o := KsmlRuntimeResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/instances/%s/clusters/%s/ksml-runtimes/%s", c.ApiURL, instanceId, clusterId, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateKsmlRuntime(instanceId string, clusterId string, id string, req KsmlRuntimeRequest) (*KsmlRuntimeResponse, error) {
	o := KsmlRuntimeResponse{}
	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/instances/%s/clusters/%s/ksml-runtimes/%s", c.ApiURL, instanceId, clusterId, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteKsmlRuntime(instanceId string, clusterId string, id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/instances/%s/clusters/%s/ksml-runtimes/%s", c.ApiURL, instanceId, clusterId, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
