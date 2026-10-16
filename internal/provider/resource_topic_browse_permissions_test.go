package provider

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func browseState(t *testing.T, r resource.Resource) *topicBrowsePermissionsResourceData {
	t.Helper()
	return &topicBrowsePermissionsResourceData{
		TopicConfig: types.StringValue("tc1"),
		Users:       types.SetValueMust(types.StringType, []attr.Value{types.StringValue("u1")}),
		Groups:      types.SetValueMust(types.StringType, []attr.Value{types.StringValue("g1")}),
	}
}

// The topic configuration was deleted (Platform Manager answers 400 "converted to null"): the
// permissions are removed from the state, so plan and destroy go on.
func TestBrowsePermissionsReadDropsAGoneTopicConfig(t *testing.T) {
	r := &topicBrowsePermissionsResource{provider: AxualProvider{client: fakePM(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"Required URI template variable 'uid' for method parameter type is present but converted to null"}`))
	})}}
	state := stateFor(t, r, browseState(t, r))
	resp := resource.ReadResponse{State: state}

	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

	if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
		t.Errorf("read = %v, state null = %t; expected the permissions removed from the state", resp.Diagnostics, resp.State.Raw.IsNull())
	}
}

// A group deleted in the same destroy is left out of the delete request, which the API would refuse.
func TestBrowsePermissionsDeleteSkipsAGoneGroup(t *testing.T) {
	var deleted string
	r := &topicBrowsePermissionsResource{provider: AxualProvider{client: fakePM(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			body, _ := io.ReadAll(req.Body)
			deleted = string(body)
			return
		}
		_, _ = w.Write([]byte(`[{"uid":"u1","type":"USER"}]`))
	})}}

	resp := deleteResource(t, r, browseState(t, r))

	if resp.Diagnostics.HasError() || deleted != `{"type":"browse","users":["u1"]}` {
		t.Errorf("delete = %v, sent %s; expected only the user that still has the permission", resp.Diagnostics, deleted)
	}
}
