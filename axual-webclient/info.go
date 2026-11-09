package webclient

import "fmt"

// GetPlatformVersion returns the Platform Manager version from GET /info, for example "16.0.2".
func (c *Client) GetPlatformVersion() (string, error) {
	var info struct {
		Version string `json:"version"`
	}
	if err := c.RequestAndMap("GET", fmt.Sprintf("%s/info", c.ApiURL), nil, nil, &info); err != nil {
		return "", err
	}
	return info.Version, nil
}
