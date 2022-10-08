package webclient

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
)

func (c *Client) SearchApplicationAccessByStreamAndApplication(streamId string, applicationId string) (*ApplicationAccessList, error) {
	o := ApplicationAccessList{}
	streamURI := fmt.Sprintf("%s/streams/%v", c.ApiURL, streamId)
	applicationURI := fmt.Sprintf("%s/applications/%v", c.ApiURL, applicationId)
	params := url.Values{}
	params.Add("stream", streamURI)
	params.Add("application", applicationURI)
	searchUrl := fmt.Sprintf("%s/application_access/search/findByApplicationAndStream?", c.ApiURL) + params.Encode()
	err := c.RequestAndMap("GET", searchUrl, nil, nil, &o)
	if err != nil {
		log.Println("Error searching Application Access: ", err)
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetApplicationAccess(id string) (*ApplicationResponse, error) {
	o := ApplicationResponse{}
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/applications/%v", c.ApiURL, id), nil, nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) UpdateApplicationAccess(id string, data ApplicationRequest) (*ApplicationResponse, error) {
	o := ApplicationResponse{}
	marshal, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("PATCH", fmt.Sprintf("%s/applications/%v", c.ApiURL, id), strings.NewReader(string(marshal)), nil, &o)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) DeleteApplicationAccess(id string) error {
	err := c.RequestAndMap("DELETE", fmt.Sprintf("%s/applications/%v", c.ApiURL, id), nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) CreateApplicationAccess(data ApplicationAccessRequest) (*ApplicationAccess, error) {
	o := ApplicationAccess{}
	marshal, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	err = c.RequestAndMap("POST", fmt.Sprintf("%s/application_access", c.ApiURL), strings.NewReader(string(marshal)), nil, &o)

	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (c *Client) GetOrCreateApplicationAccess(applicationId string, streamId string, accessType string) (*ApplicationAccess, error) {
	o := ApplicationAccess{}

	streamURI := fmt.Sprintf("%s/streams/%v", c.ApiURL, streamId)
	applicationURI := fmt.Sprintf("%s/applications/%v", c.ApiURL, applicationId)

	applicationAcesses, err := c.SearchApplicationAccessByStreamAndApplication(streamId, applicationId)
	if err != nil {
		log.Println("Error searching AA: ", err)
		return nil, err
	}

	for _, applicationAccess := range applicationAcesses.Embedded.ApplicationAccess {
		if applicationAccess.AccessType == accessType {
			o = applicationAccess
			log.Println("Application Access found:", o.Uid)
			break
		}
	}

	if o.Uid == "" {
		log.Println("No Application Access found, creating new ")
		appAccessRequest := ApplicationAccessRequest{
			applicationURI,
			streamURI,
			accessType,
		}
		newApplicationAccess, err := c.CreateApplicationAccess(appAccessRequest)
		if err != nil {
			log.Println("Error creating ApplicationAccess: ", err)
			return nil, err
		}
		o = *newApplicationAccess
	}
	log.Println(fmt.Sprintf("App Acces ID: %v", o.Uid))
	return &o, nil
}
