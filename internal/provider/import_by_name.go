package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Import ID prefixes for importing by name instead of ID.
const (
	importByNamePrefix  = "name:"
	importByEmailPrefix = "email:"
)

// importByIDOrKey imports with an ID, or with "<prefix><key>", which resolve turns into the ID.
func importByIDOrKey(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
	prefix string, resolve func(key string) (string, error)) {
	key, byKey := strings.CutPrefix(req.ID, prefix)
	if !byKey {
		// An ID never contains ":", so this is another prefix or a typo.
		if strings.Contains(req.ID, ":") {
			resp.Diagnostics.AddError("Unsupported import ID", fmt.Sprintf("expected an ID or %s<value>, got %q", prefix, req.ID))
			return
		}
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

// keyedID is one search result: the value that must match, and its ID.
type keyedID struct {
	key string
	id  string
}

// onlyExactMatch runs search and returns the one ID whose key equals key exactly. Platform Manager's
// searches ignore case, so a result can differ from key in case only. An email address does not
// depend on case, so a user matches in any case.
func onlyExactMatch(kind string, key string, search func() ([]keyedID, error)) (string, error) {
	found, err := search()
	if err != nil && !errors.Is(err, webclient.NotFoundError) {
		return "", fmt.Errorf("unable to look up %s %q: %w", kind, key, err)
	}
	ignoreCase := kind == "user"
	var ids []string
	for _, result := range found {
		if result.key == key || (ignoreCase && strings.EqualFold(result.key, key)) {
			ids = append(ids, result.id)
		}
	}
	switch len(ids) {
	case 1:
		return ids[0], nil
	case 0:
		if ignoreCase {
			return "", fmt.Errorf("no %s %q found", kind, key)
		}
		return "", fmt.Errorf("no %s %q found; it must match exactly, case included", kind, key)
	default:
		return "", fmt.Errorf("%d %ss match %q; import this one by its ID", len(ids), kind, key)
	}
}

func findTopicIDByName(client *webclient.Client, name string) (string, error) {
	return onlyExactMatch("topic", name, func() ([]keyedID, error) {
		found, err := client.GetTopicByName(name)
		if err != nil {
			return nil, err
		}
		var results []keyedID
		for _, topic := range found.Embedded.Topics {
			results = append(results, keyedID{topic.Name, topic.Uid})
		}
		return results, nil
	})
}

func findEnvironmentIDByName(client *webclient.Client, name string) (string, error) {
	return onlyExactMatch("environment", name, func() ([]keyedID, error) {
		found, err := client.GetEnvironmentByName(name)
		if err != nil {
			return nil, err
		}
		var results []keyedID
		for _, environment := range found.Embedded.Environments {
			results = append(results, keyedID{environment.Name, environment.Uid})
		}
		return results, nil
	})
}

// findApplicationIDByName: Platform Manager answers this search with one application, not a list.
func findApplicationIDByName(client *webclient.Client, name string) (string, error) {
	return onlyExactMatch("application", name, func() ([]keyedID, error) {
		found, err := client.GetApplicationByNameOrShortName(url.Values{"name": {name}})
		if err != nil {
			return nil, err
		}
		return []keyedID{{found.Name, found.Uid}}, nil
	})
}

func findGroupIDByName(client *webclient.Client, name string) (string, error) {
	return onlyExactMatch("group", name, func() ([]keyedID, error) {
		found, err := client.GetGroupByName(name)
		if err != nil {
			return nil, err
		}
		var results []keyedID
		for _, group := range found.Embedded.Groups {
			results = append(results, keyedID{group.Name, group.Uid})
		}
		return results, nil
	})
}

func findUserIDByEmail(client *webclient.Client, email string) (string, error) {
	return onlyExactMatch("user", email, func() ([]keyedID, error) {
		found, err := client.FindUserByEmail(email)
		if err != nil {
			return nil, err
		}
		var results []keyedID
		for _, user := range found.Embedded.Users {
			results = append(results, keyedID{user.EmailAddress.Email, user.UID})
		}
		return results, nil
	})
}

// findTopicConfigID resolves "<topic name>/<environment short name>" to the topic config ID.
func findTopicConfigID(client *webclient.Client, key string) (string, error) {
	slash := strings.LastIndex(key, "/")
	if slash <= 0 || slash == len(key)-1 {
		return "", fmt.Errorf("expected name:<topic name>/<environment short name>, got name:%s", key)
	}
	topicName := key[:slash]
	topicID, err := findTopicIDByName(client, topicName)
	if err != nil {
		return "", err
	}
	return onlyExactMatch("topic config", key, func() ([]keyedID, error) {
		found, err := client.FindTopicConfigsByTopic(fmt.Sprintf("%s/streams/%s", client.ApiURL, topicID))
		if err != nil {
			return nil, err
		}
		var results []keyedID
		for _, config := range found.Embedded.TopicConfigs {
			results = append(results, keyedID{topicName + "/" + config.Embedded.Environment.ShortName, config.Uid})
		}
		return results, nil
	})
}
