package webclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// FlinkSQLApplicationType is the application type the API uses for Flink SQL applications. The
// upper-case spelling is deliberate: Spring Data REST would render the enum as `Flink sql` (the way
// `KSML` becomes `Ksml`) were it not overridden in the API's rest-messages_en.properties.
const FlinkSQLApplicationType = "FLINK_SQL"

// CreateApplicationDeployment creates the deployment and returns the Uid of the created
// deployment, read from the Location header of the POST response. The POST has no response body,
// so the header is the only source for the Uid; the Uid is empty when the header is absent.
func (c *Client) CreateApplicationDeployment(applicationDeploymentRequest ApplicationDeploymentCreateRequest) (ApplicationDeploymentCreateResponse, error) {
	var o ApplicationDeploymentCreateResponse
	marshal, err := json.Marshal(applicationDeploymentRequest)
	if err != nil {
		return ApplicationDeploymentCreateResponse{}, fmt.Errorf("error creating payload for application deployment: %w", err)
	}
	headers := map[string]string{"Content-Type": "application/json"}
	responseHeaders, err := c.RequestAndMapWithHeaders("POST", fmt.Sprintf("%s/application_deployments", c.ApiURL), strings.NewReader(string(marshal)), headers, nil)
	if err != nil {
		return ApplicationDeploymentCreateResponse{}, fmt.Errorf("error sending POST request for application deployment: %w", err)
	}
	o.Uid = uidFromLocationHeader(responseHeaders)
	return o, nil
}

func (c *Client) GetApplicationDeployment(id string) (*ApplicationDeploymentResponse, error) {
	o := ApplicationDeploymentResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/application_deployments/%v", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetApplicationDeploymentStatus(id string) (*ApplicationDeploymentStatusResponse, error) {
	o := ApplicationDeploymentStatusResponse{}
	headers := map[string]string{"Content-Type": "application/json"}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/application_deployments/%v/status", c.ApiURL, id), nil, headers, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) FindApplicationDeploymentByApplicationAndEnvironment(application string, environment string) (*ApplicationDeploymentFindByApplicationAndEnvironmentResponse, error) {
	o := ApplicationDeploymentFindByApplicationAndEnvironmentResponse{}

	err :=
		c.RequestAndMap("GET", fmt.Sprintf("%s/application_deployments/search/findByApplicationAndEnvironment?application=%v&environment=%v",
			c.ApiURL, url.QueryEscape(application), url.QueryEscape(environment)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// UpdateApplicationDeployment updates an existing deployment. PATCH serves every application type;
// the PUT endpoint it replaces is deprecated for removal and rejects FLINK_SQL deployments.
func (c *Client) UpdateApplicationDeployment(id string, data ApplicationDeploymentUpdateRequest) (ApplicationDeploymentUpdateResponse, error) {
	var o ApplicationDeploymentUpdateResponse
	marshal, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{"Content-Type": "application/json"}
	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/application_deployments/%v", c.ApiURL, id), strings.NewReader(string(marshal)), headers, &o)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (c *Client) OperateApplicationDeployment(id string, action string, data ApplicationDeploymentOperationRequest) error {
	marshal, err := json.Marshal(data)
	if err != nil {
		return err
	}
	headers := map[string]string{"Content-Type": "application/json"}
	err = c.RequestAndMap("PUT", fmt.Sprintf("%s/application_deployments/%v/operation?action=%s", c.ApiURL, id, action), strings.NewReader(string(marshal)), headers, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) DeleteApplicationDeployment(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/application_deployments/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

// uidFromLocationHeader extracts the resource Uid from a Location header such as
// "https://platform.local/api/application_deployments/362f33655195493c9574fc18f5d9a701".
// It returns an empty string when the header is missing or has no usable last path segment.
func uidFromLocationHeader(headers http.Header) string {
	if headers == nil {
		return ""
	}
	location := headers.Get("Location")
	if location == "" {
		return ""
	}
	if parsed, err := url.Parse(location); err == nil && parsed.Path != "" {
		location = parsed.Path
	}
	return path.Base(strings.TrimSuffix(location, "/"))
}
