package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// addNotFoundError adds the error that every data source shows when no object has the value the
// configuration looks up, for example: No Topic resources found with name 'logs'.
func addNotFoundError(diags *diag.Diagnostics, kind string, attribute string, value string) {
	diags.AddError("Resource Not Found", fmt.Sprintf("No %s resources found with %s '%s'.", kind, attribute, value))
}

// uidStringSet returns the uids as a set of strings. No uids give an empty set, not null, like `members`
// and `tags`, so a configuration can use length() on the attribute.
func uidStringSet(ctx context.Context, uids []string) types.Set {
	set, diags := types.SetValueFrom(ctx, types.StringType, append([]string{}, uids...))
	if diags.HasError() {
		tflog.Error(ctx, "Error creating a set of uids")
	}
	return set
}
