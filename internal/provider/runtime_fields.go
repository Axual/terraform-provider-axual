package provider

import (
	webclient "axual-webclient"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The `schema_registries` and `authorized_groups` attributes that axual_flink_cluster and
// axual_ksml_runtime share.

// schemaRegistryObjectType mirrors the nested object of the `schema_registries` attribute.
func schemaRegistryObjectType() attr.TypeWithAttributeTypes {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"type": types.StringType,
		"url":  types.StringType,
	}}
}

// schemaRegistriesForRequest returns nil when the attribute has no value. The request sends that as
// an explicit null: an omitted key leaves the stored list in place, so this is what clears it.
func schemaRegistriesForRequest(list types.List) *[]webclient.SchemaRegistryConfig {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	registries := make([]webclient.SchemaRegistryConfig, 0, len(list.Elements()))
	for _, element := range list.Elements() {
		object, ok := element.(types.Object)
		if !ok {
			continue
		}
		attributes := object.Attributes()
		registry := webclient.SchemaRegistryConfig{}
		if value, ok := attributes["type"].(types.String); ok {
			registry.Type = value.ValueString()
		}
		if value, ok := attributes["url"].(types.String); ok {
			registry.Url = value.ValueString()
		}
		registries = append(registries, registry)
	}
	return &registries
}

// schemaRegistriesToList maps the API's list for state, keeping an unset attribute null: the API
// answers with an empty list when there are none, which is not the same as `[]`.
func schemaRegistriesToList(registries []webclient.SchemaRegistryConfig) types.List {
	objectType := schemaRegistryObjectType()
	if len(registries) == 0 {
		return types.ListNull(objectType)
	}

	elements := make([]attr.Value, 0, len(registries))
	for _, registry := range registries {
		object, diags := types.ObjectValue(objectType.AttributeTypes(), map[string]attr.Value{
			"type": types.StringValue(registry.Type),
			"url":  types.StringValue(registry.Url),
		})
		if diags.HasError() {
			continue
		}
		elements = append(elements, object)
	}

	list, diags := types.ListValue(objectType, elements)
	if diags.HasError() {
		return types.ListNull(objectType)
	}
	return list
}

// authorizedGroupIdsForRequest leaves the key out when there is no value yet, which is a create without the
// attribute: Platform Manager then starts with no authorized groups. On an update the plan holds the groups from
// the state when the attribute is left out, so they are kept.
func authorizedGroupIdsForRequest(groups types.Set) *[]string {
	if groups.IsNull() || groups.IsUnknown() {
		return nil
	}

	ids := make([]string, 0, len(groups.Elements()))
	for _, element := range groups.Elements() {
		if value, ok := element.(types.String); ok {
			ids = append(ids, value.ValueString())
		}
	}
	return &ids
}

// authorizedGroupsToSet maps the API's groups for state. The attribute is computed, so no authorized groups is an
// empty set, never null.
func authorizedGroupsToSet(groups []webclient.AuthorizedGroup) types.Set {
	ids := make([]attr.Value, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, types.StringValue(group.Id))
	}
	return types.SetValueMust(types.StringType, ids)
}
