package webclient

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// Client -
type Client struct {
	HTTPClient *http.Client
	ApiURL     string
	Realm      string
}

// AuthStruct -
type AuthStruct struct {
	Username string
	Password string
	Url      string
	ClientId string
	Scopes   []string
}

var NotFoundError = errors.New("resource not found")

// NewClient -
func NewClient(apiUrl string, realm string, auth AuthStruct) (*Client, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client, err := SignIn(auth)
	if err != nil {
		return nil, err
	}
	client.Timeout = 10 * time.Second
	c := Client{
		HTTPClient: client,
		ApiURL:     apiUrl,
		Realm:      realm,
	}

	return &c, nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Realm", c.Realm)
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/hal+json")
	}

	res, err := c.HTTPClient.Do(req)
	if res.StatusCode == http.StatusNotFound {
		return nil, NotFoundError
	}
	if err != nil {
		log.Println("Error:", err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("Error:", err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK &&
		res.StatusCode != http.StatusNoContent &&
		res.StatusCode != http.StatusCreated {
		log.Printf("status: %d, body: %s", res.StatusCode, body)
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, err
}

func (c *Client) RequestAndMap(method string, url string, reqBody io.Reader, header map[string]string, m interface{}) error {
	req, err := http.NewRequest(method, url, reqBody)

	if header != nil {
		for key, value := range header {
			req.Header.Set(key, value)
		}
	}

	if err != nil {
		log.Println("Error:", err)
		return err
	}

	log.Println("Logging")
	body, err := c.doRequest(req)
	if err != nil {
		log.Println("Error:", err)
		return err
	}

	if m != nil {
		if len(body) == 0 {
			return nil
		}

		err = json.Unmarshal(body, &m)
		if err != nil {
			log.Println("Error:", err)
			return err
		}
	}

	return nil
}
