package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Filter descriptions shared by several list resources.
var (
	filterOwners = listFilter{"owners", "Only resources owned by this group. Takes the group's ID or its exact name."}
	filterEnv    = listFilter{"environment", "Only resources on this environment. Takes the environment's ID, short name or exact name."}
	filterApp    = listFilter{"application", "Only resources of this application. Takes the application's ID, short name or exact name."}
	filterTopic  = listFilter{"topic", "Only resources of this topic. Takes the topic's ID or exact name."}
	filterName   = listFilter{"name", "Only resources whose name contains this text (case-insensitive)."}

	filterTopicConfigOwners = listFilter{"owners", "Only configurations of topics owned by this group. Takes the group's ID or its exact name."}
	filterDeploymentOwners  = listFilter{"owners", "Only deployments of applications owned by this group. Takes the group's ID or its exact name."}
)

// listSpecs returns the list resources of the provider. Resources wraps each type with the same
// identity attribute.
func listSpecs() []listSpec {
	return []listSpec{
		{
			typeName:    "axual_group",
			attr:        "id",
			description: "Lists groups in the tenant.",
			newResource: NewGroupResource,
			filters:     []listFilter{filterName},
			find:        findGroups,
		},
		{
			typeName:    "axual_user",
			attr:        "id",
			description: "Lists users in the tenant. Users cannot be created with Terraform, so a generated `axual_user` can only manage a user that already exists.",
			newResource: NewUserResource,
			filters: []listFilter{
				{"email", "Only users whose email address contains this text (case-insensitive)."},
				{"role", "Only users with this role, for example `TENANT_ADMIN`. Platform Manager checks the value."},
			},
			find: findUsers,
		},
		{
			typeName:    "axual_environment",
			attr:        "id",
			description: "Lists environments the user can see.",
			newResource: NewEnvironmentResource,
			filters:     []listFilter{filterName, filterOwners},
			find:        findEnvironments,
		},
		{
			typeName:    "axual_application",
			attr:        "id",
			description: "Lists applications the user can see.",
			newResource: NewApplicationResource,
			filters: []listFilter{filterName, filterOwners,
				{"short_name", "Only applications whose short name contains this text (case-insensitive)."},
				{"application_type", "Only applications of this type, as in `axual_application`: `Custom`, `Connector`, `Ksml` or `FLINK_SQL`."},
			},
			find: findApplications,
		},
		{
			typeName:    "axual_topic",
			attr:        "id",
			description: "Lists topics in the tenant.",
			newResource: NewTopicResource,
			filters:     []listFilter{filterName, filterOwners},
			find:        findTopics,
		},
		{
			typeName:    "axual_topic_config",
			attr:        "id",
			description: "Lists topic configurations (a topic on an environment).",
			newResource: NewTopicConfigResource,
			filters:     []listFilter{filterTopic, filterEnv, filterTopicConfigOwners},
			find:        findTopicConfigs,
		},
		{
			typeName:    "axual_topic_browse_permissions",
			attr:        "topic_config",
			description: "Lists the browse permissions of topic configurations. Only topic configurations that have at least one user or group with browse permission are returned. Topic configurations whose browse permissions you may not manage are left out with a warning.",
			newResource: NewTopicBrowsePermissionsResource,
			filters:     []listFilter{filterTopic, filterEnv, filterTopicConfigOwners},
			find:        findTopicBrowsePermissions,
		},
		{
			typeName:    "axual_schema_version",
			attr:        "id",
			description: "Lists schema versions in the tenant.",
			newResource: NewSchemaVersionResource,
			filters:     []listFilter{{"schema", "Only versions of the schema with this exact full name, for example `io.axual.example.Person`."}},
			find:        findSchemaVersions,
		},
		{
			typeName:    "axual_application_principal",
			attr:        "id",
			description: "Lists application principals (certificates and custom principals). `principal` and `private_key` are sensitive, so Terraform leaves them out of generated configuration: set them before you apply.",
			newResource: NewApplicationPrincipalResource,
			filters:     []listFilter{filterApp, filterEnv, {"owners", "Only principals of applications owned by this group. Takes the group's ID or its exact name."}},
			find:        findApplicationPrincipals,
		},
		{
			typeName:    "axual_application_credential",
			attr:        "id",
			description: "Lists application credentials (SASL). The password is never returned by the API, so a generated configuration creates a new credential when applied elsewhere.",
			newResource: NewApplicationCredentialResource,
			filters:     []listFilter{filterApp, filterEnv, {"owners", "Only credentials of applications owned by this group. Takes the group's ID or its exact name."}},
			find:        findApplicationCredentials,
		},
		{
			typeName:    "axual_application_access_grant",
			attr:        "id",
			description: "Lists application access grants (requests of an application to produce to or consume from a topic).",
			newResource: NewApplicationAccessGrantResource,
			filters:     grantFilters(false),
			find: func(ctx context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
				return findGrants(ctx, c, f, "", false)
			},
		},
		{
			typeName:    "axual_application_access_grant_approval",
			attr:        "application_access_grant",
			description: "Lists the approvals of application access grants: every grant with status `Approved`.",
			newResource: NewApplicationAccessGrantApprovalResource,
			filters:     grantFilters(true),
			find: func(ctx context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
				return findGrants(ctx, c, f, "APPROVED", true)
			},
		},
		{
			typeName:    "axual_application_access_grant_rejection",
			attr:        "application_access_grant",
			description: "Lists the rejections of application access grants: every grant with status `Rejected`.",
			newResource: NewApplicationAccessGrantRejectionResource,
			filters:     grantFilters(true),
			find: func(ctx context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
				return findGrants(ctx, c, f, "REJECTED", true)
			},
		},
		{
			typeName:    "axual_application_access_grant_decision",
			ownIdentity: true,
			description: "Lists the decisions on application access grants, for the topic team: `decision = \"APPROVED\"` for an `Approved` grant, " +
				"`decision = \"DENIED\"` with the platform's comment as `reason` for a `Rejected` or `Revoked` one. `Pending` and `Cancelled` grants " +
				"are left out: a pending request is for the topic team to decide, and generated config must not approve it. " +
				"A grant the application team revoked itself is left out too: that isn't a decision of the topic team.",
			newResource: NewApplicationAccessGrantDecisionResource,
			filters:     grantFilters(true),
			find: func(ctx context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
				var out []listFound
				for _, status := range []string{"APPROVED", "REJECTED", "REVOKED"} {
					found, err := findGrants(ctx, c, f, status, true)
					if err != nil {
						return nil, err
					}
					out = append(out, found...)
				}
				return out, nil
			},
			adjustWithClient: decisionFromStatus,
		},
		{
			typeName:    "axual_application_deployment",
			attr:        "id",
			description: "Lists application deployments.",
			newResource: NewApplicationDeploymentResource,
			filters:     []listFilter{filterApp, filterEnv, filterDeploymentOwners},
			find:        findDeployments,
			// An exported deployment can get an axual_application_deployment_state, which needs
			// autostart = false; with it an update leaves a stopped deployment stopped.
			adjust: func(ctx context.Context, state *tfsdk.State) diag.Diagnostics {
				return state.SetAttribute(ctx, path.Root("autostart"), false)
			},
		},
		{
			typeName:    "axual_application_deployment_state",
			attr:        "id",
			description: "Lists the running state of application deployments, one result per deployment.",
			newResource: NewApplicationDeploymentStateResource,
			filters:     []listFilter{filterApp, filterEnv, filterDeploymentOwners},
			find:        findDeployments,
			skipReason:  deploymentStateSkipReason,
		},
	}
}

// grantFilters returns the filters of the grant list resources. For an approval or rejection,
// owners is the topic's owner (the team that approves); for a grant, the application's owner.
func grantFilters(approvalOrRejection bool) []listFilter {
	owners := listFilter{"owners", "Only grants of applications owned by this group. Takes the group's ID or its exact name."}
	if approvalOrRejection {
		owners = listFilter{"owners", "Only grants on topics owned by this group, the team that approves them. Takes the group's ID or its exact name."}
	}
	f := []listFilter{
		{"application", "Only grants of this application. Takes the application's ID, short name or exact name."},
		{"topic", "Only grants on this topic. Takes the topic's ID or exact name."},
		{"environment", "Only grants on this environment. Takes the environment's ID, short name or exact name."},
		owners,
		{"access_type", "Only grants of this access type: `PRODUCER` or `CONSUMER`."},
	}
	if !approvalOrRejection {
		f = append(f, listFilter{"status", "Only grants with this status: `PENDING`, `APPROVED`, `REJECTED`, `REVOKED` or `CANCELLED`. Without it, revoked and cancelled grants are left out."})
	}
	return f
}

// notAnID is true when a GET by ID failed because v is not an ID: the API answers 404, or 400 for
// an environment. Any other error (401, 403, 5xx, a timeout) is a real error.
func notAnID(err error) bool {
	var httpErr *webclient.HTTPError
	return errors.Is(err, webclient.NotFoundError) || (errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest)
}

func resolveGroup(c *webclient.Client, v string) (string, error) {
	if _, err := c.GetGroup(url.PathEscape(v)); err == nil {
		return v, nil
	} else if !notAnID(err) {
		return "", err
	}
	return findGroupIDByName(c, v)
}

func resolveTopic(c *webclient.Client, v string) (string, error) {
	if _, err := c.GetTopic(url.PathEscape(v)); err == nil {
		return v, nil
	} else if !notAnID(err) {
		return "", err
	}
	return findTopicIDByName(c, v)
}

// resolveEnvironment takes an ID, a short name or an exact name. A short name is unique, so it wins
// over another environment's name.
func resolveEnvironment(c *webclient.Client, v string) (string, error) {
	if _, err := c.GetEnvironment(url.PathEscape(v)); err == nil {
		return v, nil
	} else if !notAnID(err) {
		return "", err
	}
	if found, err := c.GetEnvironmentByShortName(v); err == nil {
		for _, e := range found.Embedded.Environments {
			if e.ShortName == v {
				return e.Uid, nil
			}
		}
	} else if !errors.Is(err, webclient.NotFoundError) {
		return "", err
	}
	return onlyExactMatch("environment", v, func() ([]keyedID, error) {
		var results []keyedID
		for _, search := range []func(string) (*webclient.EnvironmentsResponse, error){c.GetEnvironmentByShortName, c.GetEnvironmentByName} {
			found, err := search(v)
			if err != nil && !errors.Is(err, webclient.NotFoundError) {
				return nil, err
			}
			if found == nil {
				continue
			}
			for _, e := range found.Embedded.Environments {
				results = append(results, keyedID{matchingKey(v, e.ShortName, e.Name), e.Uid})
			}
		}
		return uniqueIDs(results), nil
	})
}

// resolveApplication takes an ID, a short name or an exact name. A short name is unique, so it wins
// over another application's name.
func resolveApplication(c *webclient.Client, v string) (string, error) {
	if _, err := c.GetApplication(url.PathEscape(v)); err == nil {
		return v, nil
	} else if !notAnID(err) {
		return "", err
	}
	if found, err := c.GetApplicationByNameOrShortName(url.Values{"shortName": {v}}); err == nil && found.ShortName == v {
		return found.Uid, nil
	} else if err != nil && !errors.Is(err, webclient.NotFoundError) {
		return "", err
	}
	return onlyExactMatch("application", v, func() ([]keyedID, error) {
		var results []keyedID
		for _, key := range []string{"shortName", "name"} {
			found, err := c.GetApplicationByNameOrShortName(url.Values{key: {v}})
			if errors.Is(err, webclient.NotFoundError) {
				continue
			}
			if err != nil {
				return nil, err
			}
			results = append(results, keyedID{matchingKey(v, found.ShortName, found.Name), found.Uid})
		}
		return uniqueIDs(results), nil
	})
}

// matchingKey returns v when one of the keys equals it, so onlyExactMatch keeps the result.
func matchingKey(v string, keys ...string) string {
	for _, k := range keys {
		if k == v {
			return v
		}
	}
	return keys[len(keys)-1]
}

// uniqueIDs drops results with an ID seen before: a search by name and by short name can find the
// same resource.
func uniqueIDs(results []keyedID) []keyedID {
	seen := map[string]bool{}
	var out []keyedID
	for _, r := range results {
		if !seen[r.id] {
			seen[r.id] = true
			out = append(out, r)
		}
	}
	return out
}

// resolveFilters replaces the values of the owners, environment, application and topic filters
// with IDs.
func resolveFilters(c *webclient.Client, f map[string]string) (map[string]string, error) {
	resolvers := map[string]func(*webclient.Client, string) (string, error){
		"owners":      resolveGroup,
		"environment": resolveEnvironment,
		"application": resolveApplication,
		"topic":       resolveTopic,
	}
	out := map[string]string{}
	for k, v := range f {
		if r, ok := resolvers[k]; ok {
			id, err := r(c, v)
			if err != nil {
				return nil, err
			}
			out[k] = id
		} else {
			out[k] = v
		}
	}
	return out, nil
}

// resourceURL returns the full URL of a resource. Spring Data REST search parameters that point to
// another resource (for example `stream` or `application`) take a full URL, not an ID.
func resourceURL(c *webclient.Client, kind, id string) string {
	return fmt.Sprintf("%s/%s/%s", c.ApiURL, kind, id)
}

func toFound(items []webclient.ListItem, name func(webclient.ListItem) string) []listFound {
	out := make([]listFound, 0, len(items))
	for _, it := range items {
		out = append(out, listFound{id: it.String("uid"), displayName: name(it)})
	}
	return out
}

func byKey(key string) func(webclient.ListItem) string {
	return func(it webclient.ListItem) string { return it.String(key) }
}

func findGroups(_ context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
	path, params := "groups", url.Values{}
	if f["name"] != "" {
		path, params = "groups/search/findByNameContaining", url.Values{"name": {f["name"]}}
	}
	items, err := c.ListAll(path, params)
	if err != nil {
		return nil, err
	}
	return toFound(items, byKey("name")), nil
}

func findUsers(_ context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
	params := url.Values{}
	if f["email"] != "" {
		params.Set("email", f["email"])
	}
	if f["role"] != "" {
		params.Set("roles", f["role"])
	}
	items, err := c.ListAll("users/search/findByAttributes", params)
	if err != nil {
		return nil, err
	}
	return toFound(items, func(it webclient.ListItem) string {
		e, _ := it["emailAddress"].(map[string]any)
		email, _ := e["email"].(string)
		return email
	}), nil
}

// ownedItems calls a findByAttributes endpoint with the name and owners filters.
func ownedItems(c *webclient.Client, path string, f map[string]string, extra url.Values) ([]webclient.ListItem, error) {
	f, err := resolveFilters(c, f)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	maps.Copy(params, extra)
	if f["name"] != "" {
		params.Set("name", f["name"])
	}
	if f["owners"] != "" {
		params.Set("groupId", f["owners"])
	}
	return c.ListAll(path, params)
}

func findEnvironments(_ context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
	items, err := ownedItems(c, "environments/search/findByAttributes", f, nil)
	if err != nil {
		return nil, err
	}
	return toFound(items, byKey("name")), nil
}

func findApplications(_ context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
	extra := url.Values{}
	if f["short_name"] != "" {
		extra.Set("shortName", f["short_name"])
	}
	if f["application_type"] != "" {
		extra.Set("applicationType", strings.ToUpper(f["application_type"]))
	}
	items, err := ownedItems(c, "applications/search/findByAttributes", f, extra)
	if err != nil {
		return nil, err
	}
	return toFound(items, byKey("name")), nil
}

func findTopics(_ context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
	items, err := ownedItems(c, "streams/search/findByAttributes", f, nil)
	if err != nil {
		return nil, err
	}
	return toFound(items, byKey("name")), nil
}

// topicConfigItems returns the topic configurations that match the topic, environment and owners
// filters. There is no endpoint that lists all of them, so they are found per topic, or per
// environment when only an environment is given.
func topicConfigItems(c *webclient.Client, f map[string]string) ([]webclient.ListItem, error) {
	f, err := resolveFilters(c, f)
	if err != nil {
		return nil, err
	}

	if f["topic"] == "" && f["owners"] == "" && f["environment"] != "" {
		return c.ListAll("stream_configs/search/findByEnvironment", url.Values{"environment": {resourceURL(c, "environments", f["environment"])}})
	}

	ids, err := idsOwnedBy(c, "streams/search/findByAttributes", f["topic"], f["owners"])
	if err != nil {
		return nil, err
	}
	var out []webclient.ListItem
	for _, id := range ids {
		items, err := c.ListAll("stream_configs/search/findByStream", url.Values{"stream": {resourceURL(c, "streams", id)}})
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			if f["environment"] == "" || it.Embedded("environment", "uid") == f["environment"] {
				out = append(out, it)
			}
		}
	}
	return out, nil
}

func topicConfigName(it webclient.ListItem) string {
	return fmt.Sprintf("%s on %s", it.Embedded("stream", "name"), it.Embedded("environment", "shortName"))
}

func findTopicConfigs(_ context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
	items, err := topicConfigItems(c, f)
	if err != nil {
		return nil, err
	}
	return toFound(items, topicConfigName), nil
}

func findTopicBrowsePermissions(_ context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
	items, err := topicConfigItems(c, f)
	if err != nil {
		return nil, err
	}
	var out []listFound
	for _, it := range items {
		perms, err := c.GetTopicConfigPermissions(it.String("uid"), "browse")
		var httpErr *webclient.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusForbidden {
			out = append(out, listFound{id: it.String("uid"), displayName: topicConfigName(it), forbidden: true})
			continue
		}
		if err != nil {
			return nil, err
		}
		if len(perms) > 0 {
			out = append(out, listFound{id: it.String("uid"), displayName: topicConfigName(it)})
		}
	}
	return out, nil
}

func findSchemaVersions(_ context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
	path, params := "schema_versions", url.Values{}
	if f["schema"] != "" {
		path, params = "schema_versions/search/findBySchemaNameAndTenant", url.Values{"name": {f["schema"]}}
	}
	items, err := c.ListAll(path, params)
	if err != nil {
		return nil, err
	}
	if want := f["schema"]; want != "" {
		// The API search ignores case; the filter is exact, like the other name filters.
		items = slices.DeleteFunc(items, func(it webclient.ListItem) bool { return it.Embedded("schema", "name") != want })
	}
	return toFound(items, func(it webclient.ListItem) string {
		return fmt.Sprintf("%s %s", it.Embedded("schema", "name"), it.String("version"))
	}), nil
}

// idsOwnedBy returns the applications or topics (path) to look at: id, or all of the owners group,
// or all the user can see. With both id and owners, only id when that group owns it.
func idsOwnedBy(c *webclient.Client, path, id, owners string) ([]string, error) {
	if id != "" && owners == "" {
		return []string{id}, nil
	}
	params := url.Values{}
	if owners != "" {
		params.Set("groupId", owners)
	}
	items, err := c.ListAll(path, params)
	if err != nil {
		return nil, err
	}
	return uidsMatching(items, id), nil
}

// uidsMatching returns the uid of every item, or only want when it is set.
func uidsMatching(items []webclient.ListItem, want string) []string {
	var ids []string
	for _, it := range items {
		if want == "" || it.String("uid") == want {
			ids = append(ids, it.String("uid"))
		}
	}
	return ids
}

func appOnEnvName(it webclient.ListItem) string {
	return fmt.Sprintf("%s on %s", it.Embedded("application", "shortName"), it.Embedded("environment", "shortName"))
}

// perApplication collects the items of an application search for every application that matches
// the filters, keeping only those on the environment filter.
func perApplication(c *webclient.Client, f map[string]string, search func(appID string) ([]webclient.ListItem, error)) ([]webclient.ListItem, error) {
	f, err := resolveFilters(c, f)
	if err != nil {
		return nil, err
	}
	ids, err := idsOwnedBy(c, "applications/search/findByAttributes", f["application"], f["owners"])
	if err != nil {
		return nil, err
	}
	var out []webclient.ListItem
	for _, id := range ids {
		items, err := search(id)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			if f["environment"] == "" || it.Embedded("environment", "uid") == f["environment"] {
				out = append(out, it)
			}
		}
	}
	return out, nil
}

func findApplicationPrincipals(_ context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
	items, err := perApplication(c, f, func(appID string) ([]webclient.ListItem, error) {
		return c.ListAll("application_principals/search/findByApplication", url.Values{"application": {resourceURL(c, "applications", appID)}})
	})
	if err != nil {
		return nil, err
	}
	return toFound(items, appOnEnvName), nil
}

func findApplicationCredentials(_ context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
	items, err := perApplication(c, f, func(appID string) ([]webclient.ListItem, error) {
		return c.ListAll("application_credentials/search/findByApplicationId", url.Values{"applicationId": {appID}})
	})
	if err != nil {
		return nil, err
	}
	return toFound(items, func(it webclient.ListItem) string {
		if u := it.String("username"); u != "" {
			return u
		}
		return appOnEnvName(it)
	}), nil
}

func findDeployments(_ context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
	items, err := perApplication(c, f, func(appID string) ([]webclient.ListItem, error) {
		return c.ListAll("application_deployments/search/findByApplication", url.Values{"application": {resourceURL(c, "applications", appID)}})
	})
	if err != nil {
		return nil, err
	}
	return toFound(items, appOnEnvName), nil
}

// findGrants lists grants with the search endpoint, which takes plain IDs. status, when not
// empty, overrides the status filter. With ownersOfTopic, owners is the group that owns the topic
// (it approves or rejects the grant), otherwise the group that owns the application.
func findGrants(_ context.Context, c *webclient.Client, f map[string]string, status string, ownersOfTopic bool) ([]listFound, error) {
	f, err := resolveFilters(c, f)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if f["environment"] != "" {
		params.Set("environmentId", f["environment"])
	}
	if f["access_type"] != "" {
		params.Set("accessType", strings.ToUpper(f["access_type"]))
	}
	switch {
	case status != "":
		params.Set("statuses", status)
	case f["status"] != "":
		params.Set("statuses", strings.ToUpper(f["status"]))
	default:
		// A revoked or cancelled grant has nothing left to manage.
		params["statuses"] = []string{"PENDING", "APPROVED", "REJECTED"}
	}

	appFilters := map[string]string{"application": f["application"]}
	topicFilters := map[string]string{"topic": f["topic"]}
	if ownersOfTopic {
		topicFilters["owners"] = f["owners"]
	} else {
		appFilters["owners"] = f["owners"]
	}
	appIDs, streamIDs := []string{""}, []string{""}
	if appFilters["application"] != "" || appFilters["owners"] != "" {
		if appIDs, err = idsOwnedBy(c, "applications/search/findByAttributes", appFilters["application"], appFilters["owners"]); err != nil {
			return nil, err
		}
	}
	if topicFilters["topic"] != "" || topicFilters["owners"] != "" {
		if streamIDs, err = idsOwnedBy(c, "streams/search/findByAttributes", topicFilters["topic"], topicFilters["owners"]); err != nil {
			return nil, err
		}
	}

	var out []listFound
	for _, appID := range appIDs {
		for _, streamID := range streamIDs {
			p := url.Values{}
			maps.Copy(p, params)
			if appID != "" {
				p.Set("applicationId", appID)
			}
			if streamID != "" {
				p.Set("streamId", streamID)
			}
			items, err := c.ListAll("application_access_grants/search/findByAttributes", p)
			if err != nil {
				return nil, err
			}
			out = append(out, toFound(items, func(it webclient.ListItem) string {
				return fmt.Sprintf("%s %s %s on %s", it.Embedded("application", "shortName"), strings.ToLower(it.String("accessType")),
					it.Embedded("stream", "name"), it.Embedded("environment", "shortName"))
			})...)
		}
	}
	return out, nil
}

// deploymentStateSkipReason leaves out deployments that are neither running nor stopped, for example
// FAILED: `state` only accepts RUNNING and STOPPED.
func deploymentStateSkipReason(ctx context.Context, state tfsdk.State) string {
	var v types.String
	if diags := state.GetAttribute(ctx, path.Root("state"), &v); diags.HasError() {
		return ""
	}
	if st := v.ValueString(); st != deploymentStateRunning && st != deploymentStateStopped {
		if st == "" {
			st = "unknown"
		}
		return fmt.Sprintf("its state is %s (a failed connector task counts as failed); only %s and %s can be configured",
			st, deploymentStateRunning, deploymentStateStopped)
	}
	return ""
}

// decisionFromStatus writes the grant's current status as the decision: APPROVED for an Approved
// grant, DENIED with the platform's comment as reason for a Rejected or Revoked one. A grant the
// application team revoked itself is left out: that isn't a decision of the topic team, and DENIED
// would reject the team's next request.
func decisionFromStatus(ctx context.Context, c *webclient.Client, state *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics
	var status, grantID types.String
	diags.Append(state.GetAttribute(ctx, path.Root("current_status"), &status)...)
	diags.Append(state.GetAttribute(ctx, path.Root("grant_id"), &grantID)...)
	if diags.HasError() {
		return diags
	}
	switch status.ValueString() {
	case "Approved":
		diags.Append(state.SetAttribute(ctx, path.Root("decision"), types.StringValue(decisionApproved))...)
	case "Rejected", "Revoked":
		g, err := c.GetGrantView(grantID.ValueString())
		if err != nil {
			diags.AddError("Cannot be configured", fmt.Sprintf("couldn't read who closed the grant: %s", err))
			return diags
		}
		if status.ValueString() == "Revoked" && closedByRequester(*g) {
			diags.AddError("Cannot be configured", "the application team revoked this grant itself; that isn't a decision of the topic team. "+
				"Add a decision by hand if you need one.")
			return diags
		}
		reason := "Decided in Axual Self-Service before the export"
		if strings.TrimSpace(g.Comment) != "" {
			reason = g.Comment
		}
		diags.Append(state.SetAttribute(ctx, path.Root("decision"), types.StringValue(decisionDenied))...)
		diags.Append(state.SetAttribute(ctx, path.Root("reason"), types.StringValue(reason))...)
	default:
		diags.AddError("Cannot be configured", fmt.Sprintf("the grant is %s; only Approved, Rejected and Revoked grants are exported", status.ValueString()))
	}
	return diags
}
