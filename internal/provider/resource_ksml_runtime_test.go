package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ksmlRuntimeData(owner string, groups types.Set) *KsmlRuntimeResourceData {
	return &KsmlRuntimeResourceData{
		Name:             types.StringValue("ksml-runtime"),
		Url:              types.StringValue("https://ksml.example.com"),
		Description:      types.StringNull(),
		SchemaRegistries: types.ListNull(schemaRegistryObjectType()),
		Owners:           types.StringValue(owner),
		AuthorizedGroups: groups,
	}
}

func ksmlRuntimeRequestBody(t *testing.T, request webclient.KsmlRuntimeRequest) map[string]any {
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

func TestCreateKsmlRuntimeRequestSendsTheNameUrlAndOwner(t *testing.T) {
	body := ksmlRuntimeRequestBody(t, createKsmlRuntimeRequestFromData(ksmlRuntimeData("owner-id", types.SetNull(types.StringType))))

	for key, want := range map[string]string{"name": "ksml-runtime", "url": "https://ksml.example.com", "ownerGroupId": "owner-id"} {
		if body[key] != want {
			t.Errorf("%s = %v, expected %s", key, body[key], want)
		}
	}
}

func TestCreateKsmlRuntimeRequestOmitsAuthorizedGroupsThatHaveNoValue(t *testing.T) {
	for name, groups := range map[string]types.Set{
		"null":    types.SetNull(types.StringType),
		"unknown": types.SetUnknown(types.StringType),
	} {
		t.Run(name, func(t *testing.T) {
			request := createKsmlRuntimeRequestFromData(ksmlRuntimeData("owner-id", groups))

			if _, present := ksmlRuntimeRequestBody(t, request)["authorizedGroupIds"]; present {
				t.Error("authorizedGroupIds was sent, expected it left out")
			}
		})
	}
}

func TestCreateKsmlRuntimeRequestSendsTheConfiguredAuthorizedGroups(t *testing.T) {
	request := createKsmlRuntimeRequestFromData(ksmlRuntimeData("owner-id", groupSet("b", "a")))

	ids, _ := ksmlRuntimeRequestBody(t, request)["authorizedGroupIds"].([]any)
	got := make([]string, 0, len(ids))
	for _, id := range ids {
		got = append(got, id.(string))
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("authorizedGroupIds = %v, expected [a b]", got)
	}
}

func TestCreateKsmlRuntimeRequestSendsAnEmptyListForEmptyAuthorizedGroups(t *testing.T) {
	request := createKsmlRuntimeRequestFromData(ksmlRuntimeData("owner-id", groupSet()))

	ids, present := ksmlRuntimeRequestBody(t, request)["authorizedGroupIds"]
	if list, ok := ids.([]any); !present || !ok || len(list) != 0 {
		t.Errorf("authorizedGroupIds = %v (present %t), expected an empty list", ids, present)
	}
}

// A removed description or schema registry list is cleared by an explicit null.
func TestCreateKsmlRuntimeRequestSendsNullForAnUnsetDescriptionAndSchemaRegistries(t *testing.T) {
	body := ksmlRuntimeRequestBody(t, createKsmlRuntimeRequestFromData(ksmlRuntimeData("owner-id", groupSet())))

	for _, key := range []string{"description", "schemaRegistries"} {
		if value, present := body[key]; !present || value != nil {
			t.Errorf("%s = %v (present %t), expected an explicit null", key, value, present)
		}
	}
}

func TestCreateKsmlRuntimeRequestSendsTheDescriptionAndSchemaRegistries(t *testing.T) {
	data := ksmlRuntimeData("owner-id", groupSet())
	data.Description = types.StringValue("a runtime")
	data.SchemaRegistries = schemaRegistriesToList([]webclient.SchemaRegistryConfig{{Type: "APICURIO", Url: "https://sr.example.com"}})

	body := ksmlRuntimeRequestBody(t, createKsmlRuntimeRequestFromData(data))

	want := []any{map[string]any{"type": "APICURIO", "url": "https://sr.example.com"}}
	if body["description"] != "a runtime" || !reflect.DeepEqual(body["schemaRegistries"], want) {
		t.Errorf("description = %v, schemaRegistries = %v; expected them sent as configured", body["description"], body["schemaRegistries"])
	}
}

// A runtime that was a KSML Provisioner before the upgrade has no owner.
func TestMapKsmlRuntimeResponseMapsAMissingOwnerDescriptionAndSchemaRegistriesToNull(t *testing.T) {
	data := &KsmlRuntimeResourceData{}

	mapKsmlRuntimeResponseToData(data, &webclient.KsmlRuntimeResponse{Uid: "id", Name: "ksml", Url: "https://ksml.example.com"})

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
		Uid:              "id",
		Description:      "a runtime",
		OwnerGroupId:     "owner-id",
		SchemaRegistries: registries,
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
