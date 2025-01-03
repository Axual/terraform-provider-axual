package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// NormalizePlanModifier ensures 2 different JSON object changes are semantically equal.
type NormalizePlanModifier struct{}

func (m NormalizePlanModifier) Description(ctx context.Context) string {
	return "Suppress differences for semantically equal JSON values."
}

func (m NormalizePlanModifier) MarkdownDescription(ctx context.Context) string {
	return "Suppress differences for semantically equal JSON values."
}

// PlanModifyString modifies the plan if the user configuration and local state are semantically equivalent.
func (m NormalizePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		tflog.Info(ctx, "Skipping normalization as either state is null(creating the resource) or plan is null(destroying the resource).")
		return
	}

	if req.PlanValue.IsUnknown() || req.StateValue.IsUnknown() {
		tflog.Info(ctx, "Skipping normalization as plan or state value is unknown.")
		return
	}

	if req.PlanValue.IsNull() {
		tflog.Info(ctx, "Skipping normalization as plan value is null.")
		return
	}

	// Create Normalized values for the state and plan.
	stateNormalized := jsontypes.NewNormalizedValue(req.StateValue.ValueString())
	planNormalized := jsontypes.NewNormalizedValue(req.PlanValue.ValueString())

	// Perform semantic equality check.
	equal, diags := stateNormalized.StringSemanticEquals(ctx, planNormalized)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		tflog.Warn(ctx, fmt.Sprintf("Diagnostics error while checking semantic equality: %v", diags.Errors()))
		return
	}

	// If the values are semantically equal, set the plan value to the state value to suppress the diff.
	if equal {
		tflog.Info(ctx, "Plan and state values are semantically equal. Suppressing differences.")
		resp.PlanValue = req.StateValue
	} else {
		tflog.Info(ctx, "Plan and state values are not semantically equal. Differences will not be suppressed.")
	}
}
