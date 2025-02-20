package webclient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func (c *Client) ReadApplicationCredential(id string) (*ApplicationCredentialResponse, error) {
	o := ApplicationCredentialResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/application_credentials/%s", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) CreateApplicationCredential(applicationCredentialRequest ApplicationCredentialCreateRequest) (ApplicationCredentialResponse, error) {
	var responseList ApplicationCredentialResponseList
	marshal, err := json.Marshal(applicationCredentialRequest)
	if err != nil {
		return ApplicationCredentialResponse{}, fmt.Errorf("error creating payload for application credentials: %w", err)
	}
	headers := map[string]string{"Content-Type": "application/json"}
	err = c.RequestAndMap("POST", fmt.Sprintf("%s/application_authentications", c.ApiURL), strings.NewReader(string(marshal)), headers, &responseList)
	if err != nil {
		return ApplicationCredentialResponse{}, fmt.Errorf("error sending POST request for application credentials: %w", err)
	}

	time.Sleep(2 * time.Second)
	return responseList[0], nil
}

func (c *Client) DeleteApplicationCredential(applicationCredentialDeleteRequest ApplicationCredentialDeleteRequest) error {
	marshal, err := json.Marshal(applicationCredentialDeleteRequest)
	if err != nil {
		return err
	}

	err = c.RequestAndMap("DELETE", fmt.Sprintf("%s/application_authentications", c.ApiURL), strings.NewReader(string(marshal)), nil, nil)
	if err != nil {
		return err
	}
	time.Sleep(2 * time.Second) // Credential application can take significant time to apply in Kafka cluster
	return nil
}

func (c *Client) FindApplicationCredentialByApplicationAndEnvironment(application string, environment string) ([]ApplicationCredentialFindByApplicationAndEnvironmentResponse, error) {
	var o []ApplicationCredentialFindByApplicationAndEnvironmentResponse

	err :=
		c.RequestAndMap("GET", fmt.Sprintf("%s/application_credentials/search/findByApplicationIdAndEnvironmentId?applicationId=%v&environmentId=%v",
			c.ApiURL, url.QueryEscape(application), url.QueryEscape(environment)), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return o, nil
}
