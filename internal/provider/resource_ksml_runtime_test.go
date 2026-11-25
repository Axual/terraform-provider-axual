package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ksmlRuntimeData(owner string, groups types.Set) *KsmlRuntimeResourceData {
	return &KsmlRuntimeResourceData{
		InstanceId:       types.StringValue("i1"),
		ClusterId:        types.StringValue("c1"),
		Name:             types.StringValue("ksml-runtime"),
		Url:              types.StringValue("https://ksml.example.com"),
		Description:      types.StringNull(),
		SchemaRegistries: types.ListNull(schemaRegistryObjectType()),
		Owners:           types.StringValue(owner),
		AuthorizedGroups: groups,
	}
}

// ksmlRuntimeRequestBodies returns the JSON bodies of the create and of the update request for data.
func ksmlRuntimeRequestBodies(t *testing.T, data *KsmlRuntimeResourceData) map[string]map[string]any {
	t.Helper()
	bodies := map[string]map[string]any{}
	for verb, request := range map[string]any{
		"create": createKsmlRuntimeRequestFromData(data),
		"update": updateKsmlRuntimeRequestFromData(data),
	} {
		body, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("marshal %s request: %v", verb, err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("unmarshal %s request: %v", verb, err)
		}
		bodies[verb] = decoded
	}
	return bodies
}

func TestCreateKsmlRuntimeRequestSendsTheNameInstanceClusterUrlAndOwner(t *testing.T) {
	body := ksmlRuntimeRequestBodies(t, ksmlRuntimeData("owner-id", types.SetNull(types.StringType)))["create"]

	for key, want := range map[string]string{"name": "ksml-runtime", "instanceId": "i1", "clusterId": "c1", "ownerGroupId": "owner-id"} {
		if body[key] != want {
			t.Errorf("%s = %v, expected %s", key, body[key], want)
		}
	}
	if config, _ := body["config"].(map[string]any); config["url"] != "https://ksml.example.com" {
		t.Errorf("config = %v, expected the url in it", body["config"])
	}
}

// Platform Manager refuses a changed name, type, instance or cluster, and the plan replaces the KSML
// runtime for those, so an update sends none of them.
func TestUpdateKsmlRuntimeRequestSendsTheUrlAndOwnerAndNoFixedFields(t *testing.T) {
	body := ksmlRuntimeRequestBodies(t, ksmlRuntimeData("owner-id", types.SetNull(types.StringType)))["update"]

	if config, _ := body["config"].(map[string]any); config["url"] != "https://ksml.example.com" || body["ownerGroupId"] != "owner-id" {
		t.Errorf("ownerGroupId = %v, config = %v; expected the owner and the url", body["ownerGroupId"], body["config"])
	}
	for _, key := range []string{"name", "type", "instanceId", "clusterId"} {
		if _, present := body[key]; present {
			t.Errorf("%s was sent, expected it left out", key)
		}
	}
}

func TestKsmlRuntimeRequestsOmitAuthorizedGroupsThatHaveNoValue(t *testing.T) {
	for name, groups := range map[string]types.Set{
		"null":    types.SetNull(types.StringType),
		"unknown": types.SetUnknown(types.StringType),
	} {
		t.Run(name, func(t *testing.T) {
			for verb, body := range ksmlRuntimeRequestBodies(t, ksmlRuntimeData("owner-id", groups)) {
				if _, present := body["authorizedGroupIds"]; present {
					t.Errorf("%s: authorizedGroupIds was sent, expected it left out", verb)
				}
			}
		})
	}
}

func TestKsmlRuntimeRequestsSendTheConfiguredAuthorizedGroups(t *testing.T) {
	for verb, body := range ksmlRuntimeRequestBodies(t, ksmlRuntimeData("owner-id", groupSet("b", "a"))) {
		ids, _ := body["authorizedGroupIds"].([]any)
		got := make([]string, 0, len(ids))
		for _, id := range ids {
			got = append(got, id.(string))
		}
		sort.Strings(got)
		if !reflect.DeepEqual(got, []string{"a", "b"}) {
			t.Errorf("%s: authorizedGroupIds = %v, expected [a b]", verb, got)
		}
	}
}

func TestKsmlRuntimeRequestsSendAnEmptyListForEmptyAuthorizedGroups(t *testing.T) {
	for verb, body := range ksmlRuntimeRequestBodies(t, ksmlRuntimeData("owner-id", groupSet())) {
		ids, present := body["authorizedGroupIds"]
		if list, ok := ids.([]any); !present || !ok || len(list) != 0 {
			t.Errorf("%s: authorizedGroupIds = %v (present %t), expected an empty list", verb, ids, present)
		}
	}
}

// A create leaves an unset description out. An update clears a removed description with an explicit
// null. Schema registries without a value are an empty list on both.
func TestKsmlRuntimeRequestsForAnUnsetDescriptionAndSchemaRegistries(t *testing.T) {
	bodies := ksmlRuntimeRequestBodies(t, ksmlRuntimeData("owner-id", groupSet()))

	if _, present := bodies["create"]["description"]; present {
		t.Error("create: description was sent, expected it left out")
	}
	if value, present := bodies["update"]["description"]; !present || value != nil {
		t.Errorf("update: description = %v (present %t), expected an explicit null", value, present)
	}
	for verb, body := range bodies {
		config, _ := body["config"].(map[string]any)
		if list, ok := config["schemaRegistries"].([]any); !ok || len(list) != 0 {
			t.Errorf("%s: config.schemaRegistries = %v, expected an empty list", verb, config["schemaRegistries"])
		}
	}
}

func TestKsmlRuntimeRequestsSendTheDescriptionAndSchemaRegistries(t *testing.T) {
	data := ksmlRuntimeData("owner-id", groupSet())
	data.Description = types.StringValue("a runtime")
	data.SchemaRegistries = schemaRegistriesToList([]webclient.SchemaRegistryConfig{{Type: "APICURIO", Url: "https://sr.example.com"}})

	want := []any{map[string]any{"type": "APICURIO", "url": "https://sr.example.com"}}
	for verb, body := range ksmlRuntimeRequestBodies(t, data) {
		config, _ := body["config"].(map[string]any)
		if body["description"] != "a runtime" || !reflect.DeepEqual(config["schemaRegistries"], want) {
			t.Errorf("%s: description = %v, config.schemaRegistries = %v; expected them sent as configured", verb, body["description"], config["schemaRegistries"])
		}
	}
}

// A runtime that was a KSML Provisioner before the upgrade has no owner.
func TestMapKsmlRuntimeResponseMapsAMissingOwnerDescriptionAndSchemaRegistriesToNull(t *testing.T) {
	data := &KsmlRuntimeResourceData{}

	mapKsmlRuntimeResponseToData(data, &webclient.KsmlRuntimeResponse{
		Uid: "id", Name: "ksml", Config: webclient.KsmlRuntimeConfig{Url: "https://ksml.example.com", SchemaRegistries: []webclient.SchemaRegistryConfig{}},
	})

	if !data.Owners.IsNull() || !data.Description.IsNull() || !data.SchemaRegistries.IsNull() {
		t.Errorf("owners = %s, description = %s, schema_registries = %s; expected all null", data.Owners, data.Description, data.SchemaRegistries)
	}
	if data.Id.ValueString() != "id" || data.Name.ValueString() != "ksml" || data.Url.ValueString() != "https://ksml.example.com" {
		t.Errorf("id = %s, name = %s, url = %s; expected them mapped", data.Id, data.Name, data.Url)
	}
}

func TestMapKsmlRuntimeResponseMapsTheOwnerGroupsDescriptionAndSchemaRegistries(t *testing.T) {
	data := &KsmlRuntimeResourceData{}
	registries := []webclient.SchemaRegistryConfig{{Type: "CONFLUENT", Url: "https://sr.example.com"}}

	mapKsmlRuntimeResponseToData(data, &webclient.KsmlRuntimeResponse{
		Uid:          "id",
		Description:  "a runtime",
		OwnerGroupId: "owner-id",
		Config:       webclient.KsmlRuntimeConfig{SchemaRegistries: registries},
		AuthorizedGroups: []webclient.AuthorizedGroup{
			{Id: "a", Name: "A"},
			{Id: "b", Name: "B"},
		},
	})

	if data.Owners.ValueString() != "owner-id" || data.Description.ValueString() != "a runtime" {
		t.Errorf("owners = %s, description = %s; expected owner-id and the description", data.Owners, data.Description)
	}
	if !data.AuthorizedGroups.Equal(groupSet("a", "b")) {
		t.Errorf("authorized_groups = %s, expected [a b]", data.AuthorizedGroups)
	}
	if !data.SchemaRegistries.Equal(schemaRegistriesToList(registries)) {
		t.Errorf("schema_registries = %s, expected the one registry", data.SchemaRegistries)
	}
}

func TestMapKsmlRuntimeResponseMapsNoAuthorizedGroupsToAnEmptySet(t *testing.T) {
	for name, before := range map[string]types.Set{
		"null":    types.SetNull(types.StringType),
		"unknown": types.SetUnknown(types.StringType),
		"groups":  groupSet("a"),
	} {
		t.Run(name, func(t *testing.T) {
			data := &KsmlRuntimeResourceData{AuthorizedGroups: before}

			mapKsmlRuntimeResponseToData(data, &webclient.KsmlRuntimeResponse{Uid: "id", OwnerGroupId: "owner-id"})

			if !data.AuthorizedGroups.Equal(groupSet()) {
				t.Errorf("authorized_groups = %s, expected an empty set", data.AuthorizedGroups)
			}
		})
	}
}

func TestKsmlRuntimeImportReadsTheInstanceClusterAndRuntimeFromTheId(t *testing.T) {
	r := &ksmlRuntimeResource{}
	resp := resource.ImportStateResponse{State: emptyState(t, r)}

	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "i1/c1/r1"}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import error = %v, expected none", resp.Diagnostics)
	}
	for attribute, want := range map[string]string{"instance_id": "i1", "cluster_id": "c1", "id": "r1"} {
		var got types.String
		resp.State.GetAttribute(context.Background(), path.Root(attribute), &got)
		if got.ValueString() != want {
			t.Errorf("%s = %s, expected %s", attribute, got, want)
		}
	}
}

func TestKsmlRuntimeImportRefusesAnIdWithoutThreeParts(t *testing.T) {
	for _, id := range []string{"r1", "i1/r1", "i1/c1/", "/c1/r1", "i1/c1/r1/x"} {
		t.Run(id, func(t *testing.T) {
			r := &ksmlRuntimeResource{}
			resp := resource.ImportStateResponse{State: emptyState(t, r)}

			r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)

			if !resp.Diagnostics.HasError() {
				t.Error("import succeeded, expected an Unexpected Import Identifier error")
			}
		})
	}
}

// readKsmlRuntime runs Read on a state for runtime r1 on instanceId and clusterId, against a Platform
// Manager that answers GET /runtimes/r1 with answer.
func readKsmlRuntime(t *testing.T, instanceId, clusterId, answer string) resource.ReadResponse {
	t.Helper()
	r := &ksmlRuntimeResource{provider: AxualProvider{client: fakePM(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(answer))
	})}}
	data := ksmlRuntimeData("owner-id", groupSet())
	data.Id, data.InstanceId, data.ClusterId = types.StringValue("r1"), types.StringValue(instanceId), types.StringValue(clusterId)
	state := stateFor(t, r, data)
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	return resp
}

const ksmlRuntimeOnI1C1 = `{"id":"r1","type":"KSML","name":"ksml-runtime","instanceId":"i1","clusterId":"c1","ownerGroupId":"owner-id",` +
	`"authorizedGroups":[],"config":{"url":"https://ksml.example.com","schemaRegistries":[]}}`

// /runtimes/{id} finds a KSML runtime by its id alone, so the instance and the cluster of an import
// identifier are checked against the answer.
func TestKsmlRuntimeReadRefusesAnotherInstanceOrCluster(t *testing.T) {
	for name, tt := range map[string]struct {
		instanceId, clusterId string
		refused               bool
	}{
		"the instance and cluster it is on": {"i1", "c1", false},
		"another instance":                  {"i2", "c1", true},
		"another cluster":                   {"i1", "c2", true},
	} {
		t.Run(name, func(t *testing.T) {
			resp := readKsmlRuntime(t, tt.instanceId, tt.clusterId, ksmlRuntimeOnI1C1)

			if resp.Diagnostics.HasError() != tt.refused {
				t.Fatalf("read error = %v, expected refused = %t", resp.Diagnostics, tt.refused)
			}
			if !tt.refused {
				return
			}
			if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "i1/c1/r1") {
				t.Errorf("detail = %q, expected it to name the import identifier i1/c1/r1", detail)
			}
		})
	}
}

// /runtimes/{id} also answers for the runtimes of other types, which an axual_ksml_runtime must not take over.
func TestKsmlRuntimeReadRefusesARuntimeOfAnotherType(t *testing.T) {
	for runtimeType, refused := range map[string]bool{"KSML": false, "ksml": false, "FLINK": true, "CONNECT": true} {
		t.Run(runtimeType, func(t *testing.T) {
			resp := readKsmlRuntime(t, "i1", "c1", strings.Replace(ksmlRuntimeOnI1C1, `"type":"KSML"`, `"type":"`+runtimeType+`"`, 1))

			if resp.Diagnostics.HasError() != refused {
				t.Fatalf("read error = %v, expected refused = %t", resp.Diagnostics, refused)
			}
			if refused && !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), runtimeType) {
				t.Errorf("detail = %q, expected it to name the type %s", resp.Diagnostics.Errors()[0].Detail(), runtimeType)
			}
		})
	}
}

// A KSML runtime needs the API of Platform Manager 17.0.0 (master builds report 16.1.0-SNAPSHOT). The plan
// reads GET /info only when it creates or changes a KSML runtime.
func TestKsmlRuntimePlanRefusesOlderPlatformManager(t *testing.T) {
	planned := func() *KsmlRuntimeResourceData {
		data := ksmlRuntimeData("owner-group", groupSet("deploy-group"))
		data.Id = types.StringValue("r1")
		return data
	}
	changed := planned()
	changed.Description = types.StringValue("changed")
	const tooOld = "axual_ksml_runtime needs Platform Manager"
	for _, tc := range []struct {
		name         string
		info         string
		current      *KsmlRuntimeResourceData
		wantInDetail string // "": no error
		wantCalls    int
	}{
		{"create on 16.0.5", `{"version":"16.0.5"}`, nil, "This Platform Manager is 16.0.5. It has no KSML runtimes.", 1},
		{"create on 15.0.11", `{"version":"15.0.11"}`, nil, "This Platform Manager is 15.0.11. It has no KSML runtimes.", 1},
		{"create on a master build", `{"version":"16.1.0-SNAPSHOT"}`, nil, "", 1},
		{"create on 17.0.0", `{"version":"17.0.0"}`, nil, "", 1},
		{"version unreadable", "", nil, "version failed", 1},
		{"change on 16.0.5", `{"version":"16.0.5"}`, changed, "It has no KSML runtimes.", 1},
		{"no change on 16.0.5", `{"version":"16.0.5"}`, planned(), "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r := &ksmlRuntimeResource{provider: infoPM(t, tc.info, &calls)}
			var current any
			if tc.current != nil {
				current = tc.current
			}
			resp := modifyPlanOf(t, r, planned(), current)
			wantError := tc.wantInDetail != ""
			if resp.Diagnostics.HasError() != wantError || calls != tc.wantCalls {
				t.Fatalf("error = %v, GET /info calls = %d; expected error %v, %d calls. Diagnostics: %v",
					resp.Diagnostics.HasError(), calls, wantError, tc.wantCalls, resp.Diagnostics)
			}
			if !wantError {
				return
			}
			got := resp.Diagnostics.Errors()[0]
			if !strings.Contains(got.Summary()+got.Detail(), tooOld) || !strings.Contains(got.Detail(), tc.wantInDetail) {
				t.Errorf("summary = %q, detail = %q; expected %q in them and a detail with %q", got.Summary(), got.Detail(), tooOld, tc.wantInDetail)
			}
		})
	}
}

// A destroy is not checked, so a KSML runtime can be removed from the state on any Platform Manager.
func TestKsmlRuntimePlanAllowsDestroyOnOlderPlatformManager(t *testing.T) {
	calls := 0
	r := &ksmlRuntimeResource{provider: infoPM(t, `{"version":"16.0.5"}`, &calls)}
	data := ksmlRuntimeData("owner-group", groupSet("deploy-group"))
	data.Id = types.StringValue("r1")
	current := stateFor(t, r, data)
	empty := emptyState(t, r)
	req := resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: empty.Schema, Raw: empty.Raw}, State: current}
	resp := resource.ModifyPlanResponse{Plan: req.Plan}

	r.ModifyPlan(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() || calls != 0 {
		t.Fatalf("error = %v, GET /info calls = %d; expected no error and no call", resp.Diagnostics, calls)
	}
}

// A changed name, instance or cluster replaces the runtime; everything else is updated in place.
func TestKsmlRuntimeSchemaReplacesOnlyOnNameInstanceAndCluster(t *testing.T) {
	replaced := map[string]bool{"name": true, "instance_id": true, "cluster_id": true}
	ctx := context.Background()

	for name, attribute := range emptyState(t, &ksmlRuntimeResource{}).Schema.GetAttributes() {
		got := false
		if stringAttribute, ok := attribute.(schema.StringAttribute); ok {
			for _, modifier := range stringAttribute.PlanModifiers {
				got = got || modifier.Description(ctx) == stringplanmodifier.RequiresReplace().Description(ctx)
			}
		}
		if got != replaced[name] {
			t.Errorf("%s: requires replace = %t, expected %t", name, got, replaced[name])
		}
	}
}

// Platform Manager refuses a blank url and removes white space around it, so the plan refuses both.
func TestKsmlRuntimeUrlRefusesBlankAndPaddedValues(t *testing.T) {
	ctx := context.Background()
	attribute := emptyState(t, &ksmlRuntimeResource{}).Schema.GetAttributes()["url"].(schema.StringAttribute)

	for value, refused := range map[string]bool{
		"":                          true,
		"   ":                       true,
		" https://ksml.example.com": true,
		"https://ksml.example.com ": true,
		"https://ksml.example.com":  false,
	} {
		resp := validator.StringResponse{}
		for _, v := range attribute.Validators {
			v.ValidateString(ctx, validator.StringRequest{Path: path.Root("url"), ConfigValue: types.StringValue(value)}, &resp)
		}
		if resp.Diagnostics.HasError() != refused {
			t.Errorf("url %q: refused = %t, expected %t", value, resp.Diagnostics.HasError(), refused)
		}
	}
}

// Platform Manager stores an empty description, which is null in the state, so the plan refuses
// `description = ""`: leaving the attribute out is what removes the description.
func TestKsmlRuntimeDescriptionRefusesAnEmptyValue(t *testing.T) {
	ctx := context.Background()
	attribute := emptyState(t, &ksmlRuntimeResource{}).Schema.GetAttributes()["description"].(schema.StringAttribute)

	for name, tt := range map[string]struct {
		value   types.String
		refused bool
	}{
		"empty":    {types.StringValue(""), true},
		"left out": {types.StringNull(), false},
		"a text":   {types.StringValue("KSML runtime"), false},
		"too long": {types.StringValue(strings.Repeat("a", ksmlRuntimeDescriptionMaxLength+1)), true},
	} {
		resp := validator.StringResponse{}
		for _, v := range attribute.Validators {
			v.ValidateString(ctx, validator.StringRequest{Path: path.Root("description"), ConfigValue: tt.value}, &resp)
		}
		if resp.Diagnostics.HasError() != tt.refused {
			t.Errorf("%s: refused = %t, expected %t", name, resp.Diagnostics.HasError(), tt.refused)
		}
	}
}

// Platform Manager answers an empty list for no schema registries, which is null in the state, so
// the plan refuses `schema_registries = []`: leaving the attribute out is what clears the list.
func TestKsmlRuntimeSchemaRegistriesRefusesAnEmptyList(t *testing.T) {
	ctx := context.Background()
	attribute := emptyState(t, &ksmlRuntimeResource{}).Schema.GetAttributes()["schema_registries"].(schema.ListNestedAttribute)
	registry := []webclient.SchemaRegistryConfig{{Type: "APICURIO", Url: "https://sr.example.com"}}

	for name, tt := range map[string]struct {
		value   types.List
		refused bool
	}{
		"empty list":   {types.ListValueMust(schemaRegistryObjectType(), []attr.Value{}), true},
		"left out":     {types.ListNull(schemaRegistryObjectType()), false},
		"one registry": {schemaRegistriesToList(registry), false},
	} {
		resp := validator.ListResponse{}
		for _, v := range attribute.Validators {
			v.ValidateList(ctx, validator.ListRequest{Path: path.Root("schema_registries"), ConfigValue: tt.value}, &resp)
		}
		if resp.Diagnostics.HasError() != tt.refused {
			t.Errorf("%s: refused = %t, expected %t", name, resp.Diagnostics.HasError(), tt.refused)
		}
	}
}
