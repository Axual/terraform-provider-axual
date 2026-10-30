package provider

import (
	webclient "axual-webclient"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

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
	version  *string
	claims   map[string]decisionClaim
	removals map[string]string
	apps     map[string]appLookup
	waits    map[string]*waitGroup
}

// appLookup is an application found by UID or name: found is false when there is none that this
// user can see, uid and owners (its owner group's name) are "" when unknown.
type appLookup struct {
	found  bool
	uid    string
	owners string
}

// waitGroup collects the decisions of one topic, environment and access type that wait for a request.
type waitGroup struct {
	names []string
	last  time.Time
}

// How long the first waiting decision of a group waits for the others to join (see waiting).
var (
	waitQuiet = 300 * time.Millisecond
	waitMax   = 3 * time.Second
)

func newGrantCache(client *webclient.Client) *grantCache {
	return &grantCache{
		client:   client,
		searches: map[string][]webclient.GrantView{},
		envs:     map[string]*webclient.EnvironmentResponse{},
		topics:   map[string]string{},
		claims:   map[string]decisionClaim{},
		removals: map[string]string{},
		apps:     map[string]appLookup{},
		waits:    map[string]*waitGroup{},
	}
}

// waiting collects the decisions that wait for a request, so a topic repository gets one warning per
// topic, environment and access type instead of one per decision. Terraform plans the decisions of a
// run in parallel: the first one of a group waits until no other joined for waitQuiet (at most
// waitMax) and gets all names, sorted; the others get nil. A decision planned later starts a new group.
func (c *grantCache) waiting(key, name string) []string {
	c.mu.Lock()
	if w, ok := c.waits[key]; ok {
		w.names = append(w.names, name)
		w.last = time.Now()
		c.mu.Unlock()
		return nil
	}
	start := time.Now()
	w := &waitGroup{names: []string{name}, last: start}
	c.waits[key] = w
	c.mu.Unlock()
	for {
		time.Sleep(waitQuiet / 4)
		c.mu.Lock()
		if time.Since(w.last) >= waitQuiet || time.Since(start) >= waitMax {
			delete(c.waits, key)
			names := append([]string(nil), w.names...)
			c.mu.Unlock()
			sort.Strings(names)
			return names
		}
		c.mu.Unlock()
	}
}

// application looks an application up by UID or name. When the lookup fails for another reason than
// "not found", it counts as found, so a failing lookup never claims that an application is missing.
func (c *grantCache) application(nameOrUID string) appLookup {
	c.mu.Lock()
	a, ok := c.apps[nameOrUID]
	c.mu.Unlock()
	if ok {
		return a
	}
	v, _, _ := c.group.Do("app/"+nameOrUID, func() (any, error) {
		var app *webclient.ApplicationResponse
		var err error
		if uidPattern.MatchString(strings.ToLower(nameOrUID)) {
			app, err = c.client.GetApplication(nameOrUID)
		}
		if app == nil {
			app, err = c.client.GetApplicationByNameOrShortName(url.Values{"name": {nameOrUID}})
		}
		switch {
		case err == nil:
			return appLookup{found: true, uid: app.Uid, owners: app.Owners.Name}, nil
		case isGone(err):
			return appLookup{}, nil
		}
		return appLookup{found: true}, nil
	})
	a = v.(appLookup)
	c.mu.Lock()
	c.apps[nameOrUID] = a
	c.mu.Unlock()
	return a
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

// hidesPrivateRequests reports whether Platform Manager is older than 15.0.3, where a topic owner
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
	return versionBefore(*v, 15, 0, 3)
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

// decisionClaim is the first decision planned for a grant in this run.
type decisionClaim struct {
	id, decision string
}

// claim records that the decision with this id and value manages the grant, and returns the first
// decision that claimed the same grant in this run when it differs (zero when it doesn't). The
// provider never sees a block's address, so two blocks with the same id and value can't be told apart
// from one block planned twice; they also do the same, so that isn't reported.
func (c *grantCache) claim(grantUid, id, decision string) decisionClaim {
	c.mu.Lock()
	defer c.mu.Unlock()
	if other, ok := c.claims[grantUid]; ok {
		if other.id != id || other.decision != decision {
			return other
		}
		return decisionClaim{}
	}
	c.claims[grantUid] = decisionClaim{id: id, decision: decision}
	return decisionClaim{}
}

// remove records that the decision with this id is removed in this run and revokes the grant at apply
// (revoke_on_destroy), and returns the decision that approves the same grant in this run, if any.
func (c *grantCache) remove(grantUid, id string) decisionClaim {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removals[grantUid] = id
	if other := c.claims[grantUid]; other.decision == decisionApproved {
		return other
	}
	return decisionClaim{}
}

// removed returns the id of a decision that is removed in this run and revokes the grant, or "".
func (c *grantCache) removed(grantUid string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.removals[grantUid]
}

var uidPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// matchGrant finds the grant of an application. The grant UID from the last Read matches first, so a
// renamed application keeps its grant; then the application UID, then its name (names are unique
// per tenant, but a name could look like a UID).
func matchGrant(grants []webclient.GrantView, application, grantID string) *webclient.GrantView {
	if grantID != "" {
		for i := range grants {
			if grants[i].Uid == grantID {
				return &grants[i]
			}
		}
	}
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
