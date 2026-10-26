package webclient

import (
	"fmt"
	"net/url"
	"strings"
)

// GrantView is one application access grant with the fields the decision resource, the grant
// resource and the grants data source need. GET /application_access_grants/{id} and the grant
// search name several fields differently (requested_at against requestedAt, processedBy against
// processedByUser.email); grantViewFromItem reads both.
type GrantView struct {
	Uid             string
	Status          string
	AccessType      string // CONSUMER or PRODUCER
	ApplicationUid  string
	ApplicationName string
	TopicUid        string
	TopicName       string
	EnvironmentUid  string
	EnvironmentName string
	Comment         string
	RequestedAt     string
	ProcessedAt     string
	ProcessedBy     string
	// The actions the signed-in user is offered on the grant now (its _links).
	CanApprove bool
	CanDeny    bool
	CanRevoke  bool
	CanCancel  bool
}

// GetGrantView returns one grant by UID.
func (c *Client) GetGrantView(uid string) (*GrantView, error) {
	var item ListItem
	if err := c.RequestAndMap("GET", fmt.Sprintf("%s/application_access_grants/%s", c.ApiURL, uid), nil, nil, &item); err != nil {
		return nil, err
	}
	g := grantViewFromItem(item)
	return &g, nil
}

// SearchGrants returns every grant that matches, over all pages. Empty arguments don't filter.
// accessType is CONSUMER or PRODUCER; statuses are upper case (PENDING, APPROVED, ...).
func (c *Client) SearchGrants(topic, environment, accessType, application string, statuses []string) ([]GrantView, error) {
	q := url.Values{}
	if topic != "" {
		q.Set("streamId", topic)
	}
	if environment != "" {
		q.Set("environmentId", environment)
	}
	if accessType != "" {
		q.Set("accessType", strings.ToUpper(accessType))
	}
	if application != "" {
		q.Set("applicationId", application)
	}
	for _, s := range statuses {
		q.Add("statuses", strings.ToUpper(s))
	}
	items, err := c.ListAll("application_access_grants/search/findByAttributes", q)
	if err != nil {
		return nil, err
	}
	out := make([]GrantView, 0, len(items))
	for _, it := range items {
		out = append(out, grantViewFromItem(it))
	}
	return out, nil
}

func grantViewFromItem(i ListItem) GrantView {
	links, _ := i["_links"].(map[string]any)
	has := func(rel string) bool { _, ok := links[rel]; return ok }
	processedBy := i.String("processedBy")
	if processedBy == "" {
		if u, ok := i["processedByUser"].(map[string]any); ok {
			processedBy, _ = u["email"].(string)
		}
	}
	return GrantView{
		Uid:             i.String("uid"),
		Status:          i.String("status"),
		AccessType:      strings.ToUpper(i.String("accessType")),
		ApplicationUid:  i.Embedded("application", "uid"),
		ApplicationName: i.Embedded("application", "name"),
		TopicUid:        i.Embedded("stream", "uid"),
		TopicName:       i.Embedded("stream", "name"),
		EnvironmentUid:  i.Embedded("environment", "uid"),
		EnvironmentName: i.Embedded("environment", "name"),
		Comment:         i.String("comment"),
		RequestedAt:     firstNonEmpty(i.String("requestedAt"), i.String("requested_at")),
		ProcessedAt:     firstNonEmpty(i.String("processedAt"), i.String("processed_at")),
		ProcessedBy:     processedBy,
		CanApprove:      has("approve"),
		CanDeny:         has("deny"),
		CanRevoke:       has("revoke"),
		CanCancel:       has("cancel"),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// IsClosed reports whether the grant is Rejected, Revoked or Cancelled. A new request reopens it.
func (g GrantView) IsClosed() bool {
	return g.Status == "Rejected" || g.Status == "Revoked" || g.Status == "Cancelled"
}

// GetPlatformVersion returns the Platform Manager version from GET /info, for example "15.0.1".
func (c *Client) GetPlatformVersion() (string, error) {
	var info struct {
		Version string `json:"version"`
	}
	if err := c.RequestAndMap("GET", fmt.Sprintf("%s/info", c.ApiURL), nil, nil, &info); err != nil {
		return "", err
	}
	return info.Version, nil
}
