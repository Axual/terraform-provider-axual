package webclient

import "fmt"

// GetDeploymentTargets lists the deployment targets available to applicationId on environmentId,
// including their eligibility and metadata (e.g. kafkaAuthMethod, supportedConnectorPlugins).
func (c *Client) GetDeploymentTargets(applicationId string, environmentId string) (*DeploymentTargetsResponse, error) {
	o := DeploymentTargetsResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/applications/%s/deployment-targets?environmentId=%s", c.ApiURL, applicationId, environmentId), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
