package utils

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Helper function to set a Terraform string value or null based on input
func SetStringValue(input string) types.String {
	if input != "" {
		return types.StringValue(input)
	}
	return types.StringNull()
}

// Helper function to map the properties response from API to Terraform state
func HandlePropertiesMapping(ctx context.Context, propertiesAttr types.Map, apiProperties map[string]interface{}) types.Map {
	// Map API properties to Terraform format
	properties := map[string]attr.Value{}
	for key, value := range apiProperties {
		if value != nil {
			properties[key] = types.StringValue(value.(string))
		}
	}

	// Retrieve the current Terraform state for `properties`
	var currentPropertiesState map[string]string
	diags := propertiesAttr.ElementsAs(ctx, &currentPropertiesState, false)
	if diags.HasError() {
		tflog.Error(ctx, "Error reading current properties state", map[string]interface{}{"errors": diags.Errors()})
	}

	// API response is empty
	if len(properties) == 0 {
		// No properties in config
		if currentPropertiesState == nil {
			return types.MapNull(types.StringType)
		}
		// Empty properties in config
		return types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	// Non-empty properties from API
	mapValue, diags := types.MapValue(types.StringType, properties)
	if diags.HasError() {
		tflog.Error(ctx, "Error creating properties map")
	}
	return mapValue
}
