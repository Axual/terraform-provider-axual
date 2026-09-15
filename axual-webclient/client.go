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
	client.Timeout = 30 * time.Second
	c := Client{
		HTTPClient: client,
		ApiURL:     apiUrl,
		Realm:      realm,
		AuthMode:   auth.AuthMode,
	}
	return &c, nil
}

// doRequest performs the request and returns only the response body. Use
// doRequestWithHeaders when a response header (such as Location) is needed.
func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	body, _, err := c.doRequestWithHeaders(req)
	return body, err
}

func (c *Client) doRequestWithHeaders(req *http.Request) ([]byte, http.Header, error) {
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
		return nil, nil, err
	}
	if res == nil {
		return nil, nil, fmt.Errorf("received nil response from HTTP client")
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, nil, NotFoundError
	}
	if res.StatusCode == http.StatusUnprocessableEntity {
		return nil, nil, UnprocessableEntityError
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil, nil, err
	}

	if res.StatusCode != http.StatusOK &&
		res.StatusCode != http.StatusNoContent &&
		res.StatusCode != http.StatusCreated {
		log.Printf("Unexpected response status: %d, body: %s", res.StatusCode, body)
		return nil, nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, res.Header, err
}

func (c *Client) RequestAndMap(method string, url string, reqBody io.Reader, header map[string]string, m interface{}) error {
	_, err := c.RequestAndMapWithHeaders(method, url, reqBody, header, m)
	return err
}

// RequestAndMapWithHeaders behaves like RequestAndMap but also returns the response headers,
// so callers can read values the API only exposes there (for example the Location header of a
// POST, which carries the Uid of the newly created resource).
func (c *Client) RequestAndMapWithHeaders(method string, url string, reqBody io.Reader, header map[string]string, m interface{}) (http.Header, error) {
	req, err := http.NewRequest(method, url, reqBody)

	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return nil, err
	}

	if header != nil {
		for key, value := range header {
			req.Header.Set(key, value)
		}
	}

	body, responseHeaders, err := c.doRequestWithHeaders(req)
	if err != nil {
		log.Printf("Error performing HTTP request: %v", err)
		return nil, err
	}

	if m != nil && len(body) > 0 {
		err = json.Unmarshal(body, &m)
		if err != nil {
			log.Printf("Error unmarshaling response body: %v", err)
			return responseHeaders, err
		}
	}

	return responseHeaders, nil
}
