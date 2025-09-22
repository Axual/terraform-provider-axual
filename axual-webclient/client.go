package webclient

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Client represents an HTTP client configured to communicate with the API.
type Client struct {
	HTTPClient *http.Client
	ApiURL     string
	Realm      string
	AuthMode   string
}

// AuthStruct holds the authentication configuration.
type AuthStruct struct {
	Username string
	Password string
	Url      string
	ClientId string
	Scopes   []string
	Audience string
	AuthMode string // "keycloak" or "auth0"
}

var NotFoundError = errors.New("resource not found")
var UnprocessableEntityError = errors.New("unprocessable entity")

// NewClient creates a new Client using the provided API URL, realm, and authentication settings.
func NewClient(apiUrl string, realm string, auth AuthStruct) (*Client, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	// SignIn will choose the appropriate token flow (Keycloak or Auth0) based on auth.AuthMode.
	client, err := SignIn(auth)
	if err != nil {
		return nil, err
	}
	client.Timeout = 10 * time.Second
	c := Client{
		HTTPClient: client,
		ApiURL:     apiUrl,
		Realm:      realm,
		AuthMode:   auth.AuthMode,
	}
	return &c, nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	log.Println("Executing HTTP request...")
	// Only set Realm header if AuthMode is keycloak.
	if c.AuthMode == "keycloak" {
		req.Header.Set("Realm", c.Realm)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/hal+json")
	}

	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		log.Printf("Network error during HTTP request: %v", err)
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("received nil response from HTTP client")
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, NotFoundError
	}
	if res.StatusCode == http.StatusUnprocessableEntity {
		return nil, UnprocessableEntityError
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK &&
		res.StatusCode != http.StatusNoContent &&
		res.StatusCode != http.StatusCreated {
		log.Printf("Unexpected response status: %d, body: %s", res.StatusCode, body)
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, err
}

func (c *Client) RequestAndMap(method string, url string, reqBody io.Reader, header map[string]string, m interface{}) error {
	req, err := http.NewRequest(method, url, reqBody)

	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return err
	}

	if header != nil {
		for key, value := range header {
			req.Header.Set(key, value)
		}
	}

	body, err := c.doRequest(req)
	if err != nil {
		log.Printf("Error performing HTTP request: %v", err)
		return err
	}

	if m != nil {
		if len(body) == 0 {
			return nil
		}

		err = json.Unmarshal(body, &m)
		if err != nil {
			log.Printf("Error unmarshaling response body: %v", err)
			return err
		}
	}

	return nil
}
