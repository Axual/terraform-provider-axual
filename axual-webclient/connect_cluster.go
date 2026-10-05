package webclient

import "fmt"

// GetConnectCluster reads one registered Kafka Connect cluster by id.
func (c *Client) GetConnectCluster(instanceId string, clusterId string, id string) (*ConnectClusterResponse, error) {
	o := ConnectClusterResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/instances/%s/clusters/%s/kafka-connects/%s", c.ApiURL, instanceId, clusterId, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// FindConnectClusterByName looks up a registered Kafka Connect cluster by its name, since the API
// has no server-side name filter for this resource (unlike e.g. groups). Pages through the list
// until a match is found or every page has been read.
func (c *Client) FindConnectClusterByName(instanceId string, clusterId string, name string) (*ConnectClusterInlineResponse, error) {
	page := 0
	for {
		o := ConnectClusterListResponse{}
		url := fmt.Sprintf("%s/instances/%s/clusters/%s/kafka-connects?page=%d&size=50", c.ApiURL, instanceId, clusterId, page)
		if err := c.RequestAndMap("GET", url, nil, nil, &o); err != nil {
			return nil, err
		}
		for _, cluster := range o.Embedded.KafkaConnects {
			if cluster.Name == name {
				return &cluster, nil
			}
		}
		page++
		if page >= o.Page.TotalPages {
			return nil, NotFoundError
		}
	}
}
