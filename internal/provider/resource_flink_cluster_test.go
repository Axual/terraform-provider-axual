package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func groupSet(ids ...string) types.Set {
	values := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		values = append(values, types.StringValue(id))
	}
	return types.SetValueMust(types.StringType, values)
}

func flinkClusterData(owner string, groups types.Set) *FlinkClusterResourceData {
	return &FlinkClusterResourceData{
		Name:             types.StringValue("flink-cluster"),
		Url:              types.StringValue("https://flink.example.com"),
		Workspace:        types.StringValue("workspace"),
		Namespace:        types.StringValue("namespace"),
		DeploymentTarget: types.StringValue("target"),
		ApiToken:         types.StringValue("token"),
		SchemaRegistries: types.ListNull(schemaRegistryObjectType()),
		Owners:           types.StringValue(owner),
		AuthorizedGroups: groups,
	}
}

func requestBody(t *testing.T, request webclient.FlinkClusterRequest) map[string]any {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	return decoded
}

func TestCreateFlinkClusterRequestSendsTheOwner(t *testing.T) {
	request := createFlinkClusterRequestFromData(flinkClusterData("owner-id", types.SetNull(types.StringType)))

	if request.OwnerGroupId != "owner-id" {
		t.Errorf("owner = %q, expected owner-id", request.OwnerGroupId)
	}
}

func TestCreateFlinkClusterRequestOmitsAuthorizedGroupsThatHaveNoValue(t *testing.T) {
	for name, groups := range map[string]types.Set{
		"null":    types.SetNull(types.StringType),
		"unknown": types.SetUnknown(types.StringType),
	} {
		t.Run(name, func(t *testing.T) {
			request := createFlinkClusterRequestFromData(flinkClusterData("owner-id", groups))

			if _, present := requestBody(t, request)["authorizedGroupIds"]; present {
				t.Error("authorizedGroupIds was sent, expected it left out")
			}
		})
	}
}

func TestCreateFlinkClusterRequestSendsTheConfiguredAuthorizedGroups(t *testing.T) {
	request := createFlinkClusterRequestFromData(flinkClusterData("owner-id", groupSet("b", "a")))

	ids, _ := requestBody(t, request)["authorizedGroupIds"].([]any)
	got := make([]string, 0, len(ids))
	for _, id := range ids {
		got = append(got, id.(string))
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("authorizedGroupIds = %v, expected [a b]", got)
	}
}

func TestCreateFlinkClusterRequestSendsAnEmptyListForEmptyAuthorizedGroups(t *testing.T) {
	request := createFlinkClusterRequestFromData(flinkClusterData("owner-id", groupSet()))

	ids, present := requestBody(t, request)["authorizedGroupIds"]
	if list, ok := ids.([]any); !present || !ok || len(list) != 0 {
		t.Errorf("authorizedGroupIds = %v (present %t), expected an empty list", ids, present)
	}
}

func TestMapFlinkClusterResponseMapsAMissingOwnerToNull(t *testing.T) {
	data := &FlinkClusterResourceData{}

	mapFlinkClusterResponseToData(data, &webclient.FlinkClusterResponse{Uid: "id"})

	if !data.Owners.IsNull() {
		t.Errorf("owners = %s, expected null", data.Owners)
	}
}

func TestMapFlinkClusterResponseMapsTheOwnerAndAuthorizedGroups(t *testing.T) {
	data := &FlinkClusterResourceData{}

	mapFlinkClusterResponseToData(data, &webclient.FlinkClusterResponse{
		Uid:            "id",
		OwnerGroupId:   "owner-id",
		OwnerGroupName: "Owner",
		AuthorizedGroups: []webclient.FlinkClusterAuthorizedGroup{
			{Id: "a", Name: "A"},
			{Id: "b", Name: "B"},
		},
	})

	if data.Owners.ValueString() != "owner-id" {
		t.Errorf("owners = %s, expected owner-id", data.Owners)
	}
	if !data.AuthorizedGroups.Equal(groupSet("a", "b")) {
		t.Errorf("authorized_groups = %s, expected [a b]", data.AuthorizedGroups)
	}
}

func TestMapFlinkClusterResponseMapsNoAuthorizedGroupsToAnEmptySet(t *testing.T) {
	for name, before := range map[string]types.Set{
		"null":    types.SetNull(types.StringType),
		"unknown": types.SetUnknown(types.StringType),
		"groups":  groupSet("a"),
	} {
		t.Run(name, func(t *testing.T) {
			data := &FlinkClusterResourceData{AuthorizedGroups: before}

			mapFlinkClusterResponseToData(data, &webclient.FlinkClusterResponse{Uid: "id", OwnerGroupId: "owner-id"})

			if !data.AuthorizedGroups.Equal(groupSet()) {
				t.Errorf("authorized_groups = %s, expected an empty set", data.AuthorizedGroups)
			}
		})
	}
}

// A Flink Cluster needs the API of Platform Manager 17.0.0 (master builds report 16.1.0-SNAPSHOT). The plan
// reads GET /info only when it creates or changes a Flink Cluster.
func TestFlinkClusterPlanRefusesOlderPlatformManager(t *testing.T) {
	changed := flinkClusterData("owner-group", groupSet("deploy-group"))
	changed.Description = types.StringValue("changed")
	for _, tc := range []struct {
		name      string
		info      string
		current   *FlinkClusterResourceData
		wantError bool
		wantCalls int
	}{
		{"create on 16.0.5", `{"version":"16.0.5"}`, nil, true, 1},
		{"create on 15.0.11", `{"version":"15.0.11"}`, nil, true, 1},
		{"create on a master build", `{"version":"16.1.0-SNAPSHOT"}`, nil, false, 1},
		{"create on 17.0.0", `{"version":"17.0.0"}`, nil, false, 1},
		{"version unreadable", "", nil, true, 1},
		{"change on 16.0.5", `{"version":"16.0.5"}`, changed, true, 1},
		{"no change on 16.0.5", `{"version":"16.0.5"}`, flinkClusterData("owner-group", groupSet("deploy-group")), false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r := &flinkClusterResource{provider: infoPM(t, tc.info, &calls)}
			var current any
			if tc.current != nil {
				current = tc.current
			}
			resp := modifyPlanOf(t, r, flinkClusterData("owner-group", groupSet("deploy-group")), current)
			if resp.Diagnostics.HasError() != tc.wantError || calls != tc.wantCalls {
				t.Fatalf("error = %v, GET /info calls = %d; expected error %v, %d calls. Diagnostics: %v",
					resp.Diagnostics.HasError(), calls, tc.wantError, tc.wantCalls, resp.Diagnostics)
			}
			if tc.wantError && resp.Diagnostics.Errors()[0].Summary() != "axual_flink_cluster needs Platform Manager 17.0.0 or later" {
				t.Errorf("summary = %q", resp.Diagnostics.Errors()[0].Summary())
			}
		})
	}
}

// A destroy is not checked, so a Flink Cluster can be removed from any Platform Manager.
func TestFlinkClusterPlanAllowsDestroyOnOlderPlatformManager(t *testing.T) {
	calls := 0
	r := &flinkClusterResource{provider: infoPM(t, `{"version":"16.0.5"}`, &calls)}
	current := stateFor(t, r, flinkClusterData("owner-group", groupSet("deploy-group")))
	empty := emptyState(t, r)
	req := resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: empty.Schema, Raw: empty.Raw}, State: current}
	resp := resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() || calls != 0 {
		t.Fatalf("error = %v, GET /info calls = %d; expected no error and no call", resp.Diagnostics, calls)
	}
}
