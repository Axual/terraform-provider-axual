package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Import ID prefixes for importing by name instead of ID (GitHub #114).
const (
	importByNamePrefix  = "name:"
	importByEmailPrefix = "email:"
)

// importByIDOrKey imports with an ID, or with "<prefix><key>", which resolve turns into the ID.
func importByIDOrKey(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
	prefix string, resolve func(key string) (string, error)) {
	key, byKey := strings.CutPrefix(req.ID, prefix)
	if !byKey {
		resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
		return
	}
	id, err := resolve(key)
	if err != nil {
		resp.Diagnostics.AddError("Unable to import by "+strings.TrimSuffix(prefix, ":"), err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// onlyMatch returns the one ID found for key, or an error that says why there is not exactly one.
func onlyMatch(kind string, key string, ids []string) (string, error) {
	switch len(ids) {
	case 1:
		return ids[0], nil
	case 0:
		return "", fmt.Errorf("no %s %q found", kind, key)
	default:
		return "", fmt.Errorf("%d %ss match %q; import this one by its ID", len(ids), kind, key)
	}
}

func findByName(client *webclient.Client, kind string, searchPath string, embeddedKey string, name string) (string, error) {
	ids, err := client.FindUidsByField(searchPath, url.Values{"name": {name}}, embeddedKey, "name", name)
	if err != nil {
		return "", fmt.Errorf("unable to look up %s %q: %w", kind, name, err)
	}
	return onlyMatch(kind, name, ids)
}

func findTopicIDByName(client *webclient.Client, name string) (string, error) {
	return findByName(client, "topic", "streams/search/findByName", "streams", name)
}

func findEnvironmentIDByName(client *webclient.Client, name string) (string, error) {
	return findByName(client, "environment", "environments/search/findByName", "environments", name)
}

func findApplicationIDByName(client *webclient.Client, name string) (string, error) {
	return findByName(client, "application", "applications/search/findByName", "applications", name)
}

func findGroupIDByName(client *webclient.Client, name string) (string, error) {
	return findByName(client, "group", "groups/search/findByName", "groups", name)
}

func findUserIDByEmail(client *webclient.Client, email string) (string, error) {
	ids, err := client.FindUidsByField("users/search/findByEmailAddress", url.Values{"email": {email}}, "users", "emailAddress.email", email)
	if err != nil {
		return "", fmt.Errorf("unable to look up user %q: %w", email, err)
	}
	return onlyMatch("user", email, ids)
}

// findTopicConfigID resolves "<topic name>/<environment short name>" to the topic config ID.
func findTopicConfigID(client *webclient.Client, key string) (string, error) {
	slash := strings.LastIndex(key, "/")
	if slash <= 0 || slash == len(key)-1 {
		return "", fmt.Errorf("expected name:<topic name>/<environment short name>, got name:%s", key)
	}
	topicName, environment := key[:slash], key[slash+1:]
	topicID, err := findTopicIDByName(client, topicName)
	if err != nil {
		return "", err
	}
	topic := fmt.Sprintf("%s/streams/%s", client.ApiURL, topicID)
	ids, err := client.FindUidsByField("stream_configs/search/findByStream", url.Values{"stream": {topic}},
		"stream_configs", "_embedded.environment.shortName", environment)
	if err != nil {
		return "", fmt.Errorf("unable to look up the configs of topic %q: %w", topicName, err)
	}
	return onlyMatch("topic config", key, ids)
}
