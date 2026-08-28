package webclient

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) CreateFlinkCluster(instanceId string, clusterId string, req FlinkClusterRequest) (*FlinkClusterResponse, error) {
	o := FlinkClusterResponse{}
	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/instances/%s/clusters/%s/flink-clusters", c.ApiURL, instanceId, clusterId), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetFlinkCluster(instanceId string, clusterId string, id string) (*FlinkClusterResponse, error) {
	o := FlinkClusterResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/instances/%s/clusters/%s/flink-clusters/%s", c.ApiURL, instanceId, clusterId, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateFlinkCluster(instanceId string, clusterId string, id string, req FlinkClusterRequest) (*FlinkClusterResponse, error) {
	o := FlinkClusterResponse{}
	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/instances/%s/clusters/%s/flink-clusters/%s", c.ApiURL, instanceId, clusterId, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteFlinkCluster(instanceId string, clusterId string, id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/instances/%s/clusters/%s/flink-clusters/%s", c.ApiURL, instanceId, clusterId, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
