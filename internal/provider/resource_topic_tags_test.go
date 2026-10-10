package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func topicRequestTags(t *testing.T, tags types.Set) string {
	t.Helper()
	r := &topicResource{provider: AxualProvider{client: &webclient.Client{ApiURL: "http://pm"}}}
	data := topicResourceData{
		Owners: types.StringValue("g1"), KeyType: types.StringValue("String"), ValueType: types.StringValue("String"),
		Viewers: types.SetNull(types.StringType), Tags: tags,
	}
	request, err := createTopicRequestFromData(context.Background(), &data, r)
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	body, _ := json.Marshal(request)
	return string(body)
}

// GitHub #177: tags left out (unknown at create, or kept from state) are not sent, so Platform
// Manager keeps the tags set in the UI; tags = [] is sent as [] and removes them.
func TestTopicRequestSendsTagsOnlyWhenSet(t *testing.T) {
	if body := topicRequestTags(t, types.SetUnknown(types.StringType)); strings.Contains(body, `"tags"`) {
		t.Errorf("unknown tags sent: %s", body)
	}
	if body := topicRequestTags(t, types.SetValueMust(types.StringType, nil)); !strings.Contains(body, `"tags":[]`) {
		t.Errorf("empty tags = %s, expected \"tags\":[]", body)
	}
	tags := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pii")})
	if body := topicRequestTags(t, tags); !strings.Contains(body, `"tags":["pii"]`) {
		t.Errorf("tags = %s, expected [\"pii\"]", body)
	}
}

func TestTopicWithoutTagsReadsAnEmptySet(t *testing.T) {
	data := topicResourceData{}
	mapTopicResponseToData(context.Background(), &data, &webclient.TopicResponse{})
	if data.Tags.IsNull() || len(data.Tags.Elements()) != 0 {
		t.Errorf("tags = %v, expected an empty set", data.Tags)
	}
}
