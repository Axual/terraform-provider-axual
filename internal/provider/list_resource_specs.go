package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/list"
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
)

// listSpecs returns the list resources of the provider. Every type must also be wrapped with the
// same identity attribute in Resources.
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
				{"role", "Only users with this role, for example `TENANT_ADMIN`."},
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
				{"application_type", "Only applications of this type: `Custom`, `Connector`, `KSML` or `Flink`."},
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
			filters:     []listFilter{filterTopic, filterEnv, {"owners", "Only configurations of topics owned by this group. Takes the group's ID or its exact name."}},
			find:        findTopicConfigs,
		},
		{
			typeName:    "axual_topic_browse_permissions",
			attr:        "topic_config",
			description: "Lists the browse permissions of topic configurations. Only topic configurations that have at least one user or group with browse permission are returned.",
			newResource: NewTopicBrowsePermissionsResource,
			filters:     []listFilter{filterTopic, filterEnv, {"owners", "Only configurations of topics owned by this group. Takes the group's ID or its exact name."}},
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
			filters:     grantFilters(true),
			find: func(ctx context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
				return findGrants(ctx, c, f, "")
			},
		},
		{
			typeName:    "axual_application_access_grant_approval",
			attr:        "application_access_grant",
			description: "Lists the approvals of application access grants: every grant with status `Approved`.",
			newResource: NewApplicationAccessGrantApprovalResource,
			filters:     grantFilters(false),
			find: func(ctx context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
				return findGrants(ctx, c, f, "APPROVED")
			},
		},
		{
			typeName:    "axual_application_access_grant_rejection",
			attr:        "application_access_grant",
			description: "Lists the rejections of application access grants: every grant with status `Rejected`.",
			newResource: NewApplicationAccessGrantRejectionResource,
			filters:     grantFilters(false),
			find: func(ctx context.Context, c *webclient.Client, f map[string]string) ([]listFound, error) {
				return findGrants(ctx, c, f, "REJECTED")
			},
		},
		{
			typeName:    "axual_application_deployment",
			attr:        "id",
			description: "Lists application deployments.",
			newResource: NewApplicationDeploymentResource,
			filters:     []listFilter{filterApp, filterEnv, {"owners", "Only deployments of applications owned by this group. Takes the group's ID or its exact name."}},
			find:        findDeployments,
		},
		{
			typeName:    "axual_application_deployment_state",
			attr:        "id",
			description: "Lists the running state of application deployments, one result per deployment.",
			newResource: NewApplicationDeploymentStateResource,
			filters:     []listFilter{filterApp, filterEnv, {"owners", "Only deployments of applications owned by this group. Takes the group's ID or its exact name."}},
			find:        findDeployments,
			skipReason:  deploymentStateSkipReason,
		},
	}
}

// ListResources returns the list resources used by `terraform query`.
func (p *AxualProvider) ListResources(_ context.Context) []func() list.ListResource {
	var out []func() list.ListResource
	for _, s := range listSpecs() {
		spec := s
		out = append(out, func() list.ListResource { return newListResource(*p, spec) })
	}
	return out
}

func grantFilters(withStatus bool) []listFilter {
	f := []listFilter{
		{"application", "Only grants of this application. Takes the application's ID, short name or exact name."},
		{"topic", "Only grants on this topic. Takes the topic's ID or exact name."},
		{"environment", "Only grants on this environment. Takes the environment's ID, short name or exact name."},
		{"owners", "Only grants of applications owned by this group. Takes the group's ID or its exact name."},
		{"access_type", "Only grants of this access type: `PRODUCER` or `CONSUMER`."},
	}
	if withStatus {
		f = append(f, listFilter{"status", "Only grants with this status: `PENDING`, `APPROVED`, `REJECTED`, `REVOKED` or `CANCELLED`."})
	}
	return f
}

// ---- resolvers: turn a filter value (an ID or a name) into an ID ----

// resolveOne returns the uid of the only item in found, or an error when there is none or more
// than one.
func resolveOne(kind, value string, found []webclient.ListItem, match func(webclient.ListItem) bool) (string, error) {
	var ids []string
	for _, it := range found {
		if match == nil || match(it) {
			ids = append(ids, it.String("uid"))
		}
	}
	switch len(ids) {
	case 0:
		return "", fmt.Errorf("no %s found with ID or name %q", kind, value)
	case 1:
		return ids[0], nil
	default:
		return "", fmt.Errorf("more than one %s found with name %q; use its ID instead", kind, value)
	}
}

func resolveGroup(c *webclient.Client, v string) (string, error) {
	if _, err := c.GetGroup(v); err == nil {
		return v, nil
	}
	found, err := c.ListAll("groups/search/findByName", url.Values{"name": {v}})
	if err != nil {
		return "", err
	}
	return resolveOne("group", v, found, nil)
}

func resolveEnvironment(c *webclient.Client, v string) (string, error) {
	if _, err := c.GetEnvironment(v); err == nil {
		return v, nil
	}
	found, err := c.ListAll("environments/search/findByShortName", url.Values{"shortName": {v}})
	if err != nil {
		return "", err
	}
	if len(found) == 0 {
		if found, err = c.ListAll("environments/search/findByName", url.Values{"name": {v}}); err != nil {
			return "", err
		}
	}
	return resolveOne("environment", v, found, nil)
}

func resolveApplication(c *webclient.Client, v string) (string, error) {
	if _, err := c.GetApplication(v); err == nil {
		return v, nil
	}
	found, err := c.ListAll("applications/search/findByAttributes", url.Values{"name": {v}})
	if err != nil {
		return "", err
	}
	more, err := c.ListAll("applications/search/findByAttributes", url.Values{"shortName": {v}})
	if err != nil {
		return "", err
	}
	byID := map[string]webclient.ListItem{}
	for _, it := range append(found, more...) {
		byID[it.String("uid")] = it
	}
	var all []webclient.ListItem
	for _, it := range byID {
		all = append(all, it)
	}
	return resolveOne("application", v, all, func(it webclient.ListItem) bool {
		return strings.EqualFold(it.String("name"), v) || strings.EqualFold(it.String("shortName"), v)
	})
}

func resolveTopic(c *webclient.Client, v string) (string, error) {
	if _, err := c.GetTopic(v); err == nil {
		return v, nil
	}
	found, err := c.ListAll("streams/search/findByName", url.Values{"name": {v}})
	if err != nil {
		return "", err
	}
	return resolveOne("topic", v, found, nil)
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

// ---- finders ----

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
	for k, v := range extra {
		params[k] = v
	}
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

	var topicIDs []string
	if f["topic"] != "" {
		topicIDs = []string{f["topic"]}
	} else {
		params := url.Values{}
		if f["owners"] != "" {
			params.Set("groupId", f["owners"])
		}
		topics, err := c.ListAll("streams/search/findByAttributes", params)
		if err != nil {
			return nil, err
		}
		for _, t := range topics {
			topicIDs = append(topicIDs, t.String("uid"))
		}
	}

	var out []webclient.ListItem
	for _, id := range topicIDs {
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
	return toFound(items, func(it webclient.ListItem) string {
		return fmt.Sprintf("%s %s", it.Embedded("schema", "name"), it.String("version"))
	}), nil
}

// applicationIDs returns the applications to look at for a resource that belongs to an
// application: the one in the application filter, or all applications of the owners filter, or
// all applications the user can see.
func applicationIDs(c *webclient.Client, f map[string]string) ([]string, error) {
	if f["application"] != "" {
		return []string{f["application"]}, nil
	}
	params := url.Values{}
	if f["owners"] != "" {
		params.Set("groupId", f["owners"])
	}
	apps, err := c.ListAll("applications/search/findByAttributes", params)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(apps))
	for _, a := range apps {
		ids = append(ids, a.String("uid"))
	}
	return ids, nil
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
	ids, err := applicationIDs(c, f)
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
// empty, overrides the status filter.
func findGrants(_ context.Context, c *webclient.Client, f map[string]string, status string) ([]listFound, error) {
	f, err := resolveFilters(c, f)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if f["topic"] != "" {
		params.Set("streamId", f["topic"])
	}
	if f["environment"] != "" {
		params.Set("environmentId", f["environment"])
	}
	if f["access_type"] != "" {
		params.Set("accessType", strings.ToUpper(f["access_type"]))
	}
	if status == "" {
		status = strings.ToUpper(f["status"])
	}
	if status != "" {
		params.Set("statuses", status)
	}

	var appIDs []string
	if f["application"] != "" || f["owners"] != "" {
		if appIDs, err = applicationIDs(c, f); err != nil {
			return nil, err
		}
	} else {
		appIDs = []string{""}
	}

	var out []listFound
	for _, appID := range appIDs {
		p := url.Values{}
		for k, v := range params {
			p[k] = v
		}
		if appID != "" {
			p.Set("applicationId", appID)
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
	return out, nil
}

// deploymentStateSkipReason leaves out deployments that are neither running nor stopped, for example
// FAILED: `state` only accepts RUNNING and STOPPED.
func deploymentStateSkipReason(ctx context.Context, state tfsdk.State) string {
	var v types.String
	if diags := state.GetAttribute(ctx, path.Root("state"), &v); diags.HasError() {
		return ""
	}
	if st := v.ValueString(); st != "RUNNING" && st != "STOPPED" {
		return fmt.Sprintf("the deployment is %s; only RUNNING and STOPPED can be configured", st)
	}
	return ""
}
