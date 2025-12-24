package utils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	webclient "axual-webclient"
)

// SetStringValue set a Terraform string value or null based on input
func SetStringValue(input string) types.String {
	if input != "" {
		return types.StringValue(input)
	}
	return types.StringNull()
}

// HandlePropertiesMapping map the properties's response from API to Terraform state
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

	// The properties in API response is an empty map: properties = {}
	if len(properties) == 0 {
		// The properties in config are missing or null, which means that properties = null in Terraform state
		if currentPropertiesState == nil {
			return types.MapNull(types.StringType)
		}
		// The properties in config is empty map: properties = {}, which means that properties = {} in Terraform state
		return types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	// The properties in API response is a map that contains at least one key-value pair.
	mapValue, diags := types.MapValue(types.StringType, properties)
	if diags.HasError() {
		tflog.Error(ctx, "Error creating properties map")
	}
	return mapValue
}

func ShouldStopDeployment(deploymentType string, status *webclient.ApplicationDeploymentStatusResponse) bool {
    // Reuse logic: Check if connector is not stopped or KSML is not undeployed
    if (IsKSML(deploymentType) && status.KsmlStatus.Status != "Undeployed") || (!IsKSML(deploymentType) && status.ConnectorState.State != "Stopped") {
        return true
    }
    return false
}

func IsKSML(deploymentType string) bool {
    return deploymentType == "Ksml"
}