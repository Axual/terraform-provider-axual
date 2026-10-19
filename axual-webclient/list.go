package webclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"strconv"
)

// listPageSize is the page size used when walking a paged list or search endpoint.
const listPageSize = 200

// listMaxPages stops a paged walk that never ends, for example when the server ignores `page`.
const listMaxPages = 1000

// ListItem is one entity from a Platform Manager list or search response, as plain JSON.
type ListItem map[string]any

// ListAll returns the items of every page. path is relative to the API URL. It accepts paged HAL,
// unpaged HAL and a plain JSON array.
func (c *Client) ListAll(path string, params url.Values) ([]ListItem, error) {
	q := url.Values{}
	maps.Copy(q, params)
	q.Set("size", strconv.Itoa(listPageSize))
	if q.Get("sort") == "" {
		// A fixed order, so pages do not overlap or skip an item. Every list endpoint takes "id";
		// "uid" is refused.
		q.Set("sort", "id,asc")
	}

	var items []ListItem
	seen := map[string]bool{}
	add := func(found []ListItem) {
		for _, it := range found {
			if uid := it.String("uid"); uid != "" {
				if seen[uid] {
					continue
				}
				seen[uid] = true
			}
			items = append(items, it)
		}
	}
	for page := 0; page < listMaxPages; page++ {
		q.Set("page", strconv.Itoa(page))
		var raw json.RawMessage
		if err := c.RequestAndMap("GET", fmt.Sprintf("%s/%s?%s", c.ApiURL, path, q.Encode()), nil, nil, &raw); err != nil {
			return nil, err
		}
		raw = bytes.TrimSpace(raw)
		if len(raw) == 0 {
			return items, nil
		}
		if raw[0] == '[' {
			var arr []ListItem
			if err := json.Unmarshal(raw, &arr); err != nil {
				return nil, err
			}
			add(arr)
			return items, nil
		}

		var body struct {
			Embedded map[string][]ListItem `json:"_embedded"`
			Page     *struct {
				TotalPages int `json:"totalPages"`
			} `json:"page"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, err
		}
		for _, v := range body.Embedded {
			add(v)
		}
		if body.Page == nil || page+1 >= body.Page.TotalPages {
			return items, nil
		}
	}
	return nil, fmt.Errorf("stopped listing %s after %d pages", path, listMaxPages)
}

// String returns the string value of key, or "" when it is missing or not a string.
func (i ListItem) String(key string) string {
	s, _ := i[key].(string)
	return s
}

// Embedded returns the string value of key in the related object name, for example
// Embedded("environment", "uid"). HAL responses put the object under `_embedded`; plain JSON
// responses put it at the top level. It returns "" when either is missing.
func (i ListItem) Embedded(name, key string) string {
	e, _ := i["_embedded"].(map[string]any)
	o, ok := e[name].(map[string]any)
	if !ok {
		o, _ = i[name].(map[string]any)
	}
	s, _ := o[key].(string)
	return s
}
