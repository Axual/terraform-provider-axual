package provider

import (
	webclient "axual-webclient"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
