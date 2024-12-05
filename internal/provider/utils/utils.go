package utils

import "github.com/hashicorp/terraform-plugin-framework/types"

// Helper function to set a Terraform string value or null based on input
func SetStringValue(input string) types.String {
	if input != "" {
		return types.StringValue(input)
	}
	return types.StringNull()
}
