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
		Owner:            types.StringValue(owner),
		AuthorizedGroups: groups,
	}
}

// requestBody is the JSON the client sends, so a test sees what Platform Manager receives.
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
	request := createFlinkClusterRequestFromData(flinkClusterData("owner-id", types.SetNull(types.StringType)), false)

	if request.OwnerGroupId != "owner-id" {
		t.Errorf("owner = %q, expected owner-id", request.OwnerGroupId)
	}
}

func TestCreateFlinkClusterRequestOmitsAuthorizedGroupsThatAreNotConfigured(t *testing.T) {
	request := createFlinkClusterRequestFromData(flinkClusterData("owner-id", types.SetNull(types.StringType)), false)

	if _, present := requestBody(t, request)["authorizedGroupIds"]; present {
		t.Error("authorizedGroupIds was sent, expected it left out")
	}
}

func TestCreateFlinkClusterRequestSendsTheConfiguredAuthorizedGroups(t *testing.T) {
	request := createFlinkClusterRequestFromData(flinkClusterData("owner-id", groupSet("b", "a")), false)

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
	request := createFlinkClusterRequestFromData(flinkClusterData("owner-id", groupSet()), false)

	ids, present := requestBody(t, request)["authorizedGroupIds"]
	if list, ok := ids.([]any); !present || !ok || len(list) != 0 {
		t.Errorf("authorizedGroupIds = %v (present %t), expected an empty list", ids, present)
	}
}

func TestUpdateFlinkClusterRequestSendsAnEmptyListWhenAuthorizedGroupsAreRemoved(t *testing.T) {
	request := createFlinkClusterRequestFromData(flinkClusterData("owner-id", types.SetNull(types.StringType)), true)

	ids, present := requestBody(t, request)["authorizedGroupIds"]
	if list, ok := ids.([]any); !present || !ok || len(list) != 0 {
		t.Errorf("authorizedGroupIds = %v (present %t), expected an empty list: a null is rejected and a missing key keeps the groups", ids, present)
	}
}

func TestUpdateFlinkClusterRequestOmitsAuthorizedGroupsWhenThereWereNoneAndNoneArePlanned(t *testing.T) {
	request := createFlinkClusterRequestFromData(flinkClusterData("owner-id", types.SetNull(types.StringType)), false)

	if _, present := requestBody(t, request)["authorizedGroupIds"]; present {
		t.Error("authorizedGroupIds was sent, expected it left out")
	}
}

func TestMapFlinkClusterResponseMapsAMissingOwnerToNull(t *testing.T) {
	data := &FlinkClusterResourceData{}

	mapFlinkClusterResponseToData(data, &webclient.FlinkClusterResponse{Uid: "id"})

	if !data.Owner.IsNull() {
		t.Errorf("owner = %s, expected null", data.Owner)
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

	if data.Owner.ValueString() != "owner-id" {
		t.Errorf("owner = %s, expected owner-id", data.Owner)
	}
	if !data.AuthorizedGroups.Equal(groupSet("a", "b")) {
		t.Errorf("authorized_groups = %s, expected [a b]", data.AuthorizedGroups)
	}
}

func TestMapFlinkClusterResponseMapsNoAuthorizedGroupsToNullWhenTheAttributeIsNotSet(t *testing.T) {
	data := &FlinkClusterResourceData{AuthorizedGroups: types.SetNull(types.StringType)}

	mapFlinkClusterResponseToData(data, &webclient.FlinkClusterResponse{Uid: "id", OwnerGroupId: "owner-id"})

	if !data.AuthorizedGroups.IsNull() {
		t.Errorf("authorized_groups = %s, expected null", data.AuthorizedGroups)
	}
}

// The framework fails the apply with "Provider produced inconsistent result" when `authorized_groups = []`
// comes back as null, so an empty set has to stay an empty set.
func TestMapFlinkClusterResponseKeepsAnEmptySetForEmptyAuthorizedGroups(t *testing.T) {
	data := &FlinkClusterResourceData{AuthorizedGroups: groupSet()}

	mapFlinkClusterResponseToData(data, &webclient.FlinkClusterResponse{Uid: "id", OwnerGroupId: "owner-id"})

	if data.AuthorizedGroups.IsNull() || len(data.AuthorizedGroups.Elements()) != 0 {
		t.Errorf("authorized_groups = %s, expected an empty set", data.AuthorizedGroups)
	}
}
