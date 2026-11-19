package webclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func (c *Client) CreateCluster(req ClusterRequest) (*ClusterResponse, error) {
	o := ClusterResponse{}
	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/clusters", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetCluster(id string) (*ClusterResponse, error) {
	o := ClusterResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/clusters/%s", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// GetClustersByName returns the clusters named name, ignoring case, that the user can see: its own and
// the shared ones. The search pages; the first page holds them all, because there are only a few.
func (c *Client) GetClustersByName(name string) ([]ClusterResponse, error) {
	o := ClustersResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/clusters/search/findByName?name=%s", c.ApiURL, url.QueryEscape(name)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return o.Embedded.Clusters, nil
}

func (c *Client) UpdateCluster(id string, req ClusterRequest) (*ClusterResponse, error) {
	o := ClusterResponse{}
	marshal, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/clusters/%s", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteCluster(id string) error {
	return c.RequestAndMap("DELETE", fmt.Sprintf("%s/clusters/%s", c.ApiURL, id), nil, nil, nil)
}
