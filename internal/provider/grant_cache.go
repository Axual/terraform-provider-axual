package provider

import (
	webclient "axual-webclient"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
)

// grantCache keeps what the access grant decisions share in one Terraform run (one provider
// process): the grant search per topic, environment and access type, and the environments. Terraform
// reads resources in parallel, so concurrent lookups of one key make a single API call.
type grantCache struct {
	client *webclient.Client
	group  singleflight.Group

	mu       sync.Mutex
	searches map[string][]webclient.GrantView
	envs     map[string]*webclient.EnvironmentResponse
	topics   map[string]string
	read     map[string]bool
	version  *string
}

func newGrantCache(client *webclient.Client) *grantCache {
	return &grantCache{
		client:   client,
		searches: map[string][]webclient.GrantView{},
		envs:     map[string]*webclient.EnvironmentResponse{},
		topics:   map[string]string{},
		read:     map[string]bool{},
	}
}

// search returns all grants of a topic in an environment for one access type. fresh skips and
// replaces the cached result, for use right before an action.
func (c *grantCache) search(topic, environment, accessType string, fresh bool) ([]webclient.GrantView, error) {
	key := topic + "/" + environment + "/" + accessType
	if !fresh {
		c.mu.Lock()
		found, ok := c.searches[key]
		c.mu.Unlock()
		if ok {
			return found, nil
		}
	}
	v, err, _ := c.group.Do(key, func() (any, error) {
		return c.client.SearchGrants(topic, environment, accessType, "", nil)
	})
	if err != nil {
		return nil, err
	}
	grants := v.([]webclient.GrantView)
	c.mu.Lock()
	c.searches[key] = grants
	c.mu.Unlock()
	return grants, nil
}

// environment returns the environment, or nil when the caller may not read it.
func (c *grantCache) environment(uid string) *webclient.EnvironmentResponse {
	c.mu.Lock()
	env, ok := c.envs[uid]
	c.mu.Unlock()
	if ok {
		return env
	}
	v, err, _ := c.group.Do("env/"+uid, func() (any, error) { return c.client.GetEnvironment(uid) })
	if err == nil {
		env = v.(*webclient.EnvironmentResponse)
	}
	c.mu.Lock()
	c.envs[uid] = env
	c.mu.Unlock()
	return env
}

// topicName returns the topic's name, or "" when the caller may not read it.
func (c *grantCache) topicName(uid string) string {
	c.mu.Lock()
	name, ok := c.topics[uid]
	c.mu.Unlock()
	if ok {
		return name
	}
	if t, err := c.client.GetTopic(uid); err == nil {
		name = t.Name
	}
	c.mu.Lock()
	c.topics[uid] = name
	c.mu.Unlock()
	return name
}

// hidesPrivateRequests reports whether Platform Manager is older than 15.0.4, where a topic owner
// can't see the requests of private applications. False when the version is unknown.
func (c *grantCache) hidesPrivateRequests() bool {
	c.mu.Lock()
	v := c.version
	c.mu.Unlock()
	if v == nil {
		got, _, _ := c.group.Do("version", func() (any, error) {
			version, _ := c.client.GetPlatformVersion()
			return version, nil
		})
		version := got.(string)
		c.mu.Lock()
		c.version = &version
		c.mu.Unlock()
		v = &version
	}
	return versionBefore(*v, 15, 0, 4)
}

// versionBefore compares a version such as "15.0.1" or "16.1.0-SNAPSHOT" with major.minor.patch.
func versionBefore(version string, major, minor, patch int) bool {
	var got [3]int
	if n, _ := fmt.Sscanf(version, "%d.%d.%d", &got[0], &got[1], &got[2]); n < 3 {
		return false
	}
	for i, want := range [3]int{major, minor, patch} {
		if got[i] != want {
			return got[i] < want
		}
	}
	return false
}

// markRead records that Read ran for the decision in this process, so ModifyPlan knows whether the
// state is fresh (it isn't with -refresh=false).
func (c *grantCache) markRead(id string) {
	c.mu.Lock()
	c.read[id] = true
	c.mu.Unlock()
}

func (c *grantCache) wasRead(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.read[id]
}

var uidPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// matchGrant finds the grant of an application, given as a UID or a name. A UID matches first,
// because application names are unique per tenant but a name could look like a UID.
func matchGrant(grants []webclient.GrantView, application string) *webclient.GrantView {
	if uidPattern.MatchString(strings.ToLower(application)) {
		for i := range grants {
			if strings.EqualFold(grants[i].ApplicationUid, application) {
				return &grants[i]
			}
		}
	}
	for i := range grants {
		if grants[i].ApplicationName == application {
			return &grants[i]
		}
	}
	return nil
}
