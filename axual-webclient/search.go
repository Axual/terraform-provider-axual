package webclient

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// FindUidsByField runs a Platform Manager search and returns the uid of each result whose field
// (a dotted path, for example "emailAddress.email") equals value exactly. A search answers either
// a single object or a HAL list under _embedded[embeddedKey]; both are read. No result is not an error.
func (c *Client) FindUidsByField(searchPath string, query url.Values, embeddedKey string, field string, value string) ([]string, error) {
	var body map[string]any
	err := c.RequestAndMap("GET", fmt.Sprintf("%s/%s?%s", c.ApiURL, searchPath, query.Encode()), nil, nil, &body)
	if errors.Is(err, NotFoundError) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items := []map[string]any{body}
	if embedded, ok := body["_embedded"].(map[string]any); ok {
		items = nil
		list, _ := embedded[embeddedKey].([]any)
		for _, raw := range list {
			if item, ok := raw.(map[string]any); ok {
				items = append(items, item)
			}
		}
	}
	var uids []string
	for _, item := range items {
		uid, _ := item["uid"].(string)
		if uid != "" && stringAt(item, field) == value {
			uids = append(uids, uid)
		}
	}
	return uids, nil
}

func stringAt(item map[string]any, field string) string {
	var current any = item
	for _, key := range strings.Split(field, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = object[key]
	}
	value, _ := current.(string)
	return value
}
