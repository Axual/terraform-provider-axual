package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MoveState moves axual_application_access_grant_approval and _rejection to the decision, with a
// moved block (Terraform 1.8 or later). It fills the lookup attributes from the grant, so a config
// in the lookup form plans "No changes".
func (r *grantDecisionResource) MoveState(_ context.Context) []resource.StateMover {
	source := &schema.Schema{Attributes: map[string]schema.Attribute{
		"application_access_grant": schema.StringAttribute{Required: true},
		"reason":                   schema.StringAttribute{Optional: true},
	}}
	approval := &schema.Schema{Attributes: map[string]schema.Attribute{
		"application_access_grant": schema.StringAttribute{Required: true},
	}}
	return []resource.StateMover{
		{SourceSchema: approval, StateMover: r.moveFrom("axual_application_access_grant_approval", decisionApproved)},
		{SourceSchema: source, StateMover: r.moveFrom("axual_application_access_grant_rejection", decisionDenied)},
	}
}

func (r *grantDecisionResource) moveFrom(typeName, decision string) func(context.Context, resource.MoveStateRequest, *resource.MoveStateResponse) {
	return func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
		// Leave other sources to the other movers.
		if req.SourceTypeName != typeName || !strings.HasSuffix(req.SourceProviderAddress, "/axual") || req.SourceState == nil {
			return
		}
		var uid, reason types.String
		resp.Diagnostics.Append(req.SourceState.GetAttribute(ctx, path.Root("application_access_grant"), &uid)...)
		if decision == decisionDenied {
			resp.Diagnostics.Append(req.SourceState.GetAttribute(ctx, path.Root("reason"), &reason)...)
		}
		if resp.Diagnostics.HasError() {
			return
		}
		// MoveState runs before the provider is configured, so there is no API client here. The next
		// Read fills application, topic, environment, access_type and id from the grant.
		d := grantDecisionData{
			Id:              types.StringNull(),
			Grant:           uid,
			Application:     types.StringNull(),
			Topic:           types.StringNull(),
			Environment:     types.StringNull(),
			AccessType:      types.StringNull(),
			Decision:        types.StringValue(decision),
			Reason:          stringOrNull(reason.ValueString()),
			Review:          types.StringNull(),
			RevokeOnDestroy: types.BoolValue(false),
			GrantId:         uid,
			CurrentStatus:   types.StringNull(),
			NextRequest:     types.StringNull(),
		}
		if decision == decisionApproved {
			d.NextRequest = types.StringValue(nextRequestReview)
		}
		resp.Diagnostics.Append(resp.TargetState.Set(ctx, &d)...)
		if decision == decisionApproved {
			resp.Diagnostics.AddWarning(fmt.Sprintf("Moved the approval of grant %s to a decision", uid.ValueString()),
				"Two things work differently now: removing the block no longer revokes the grant unless revoke_on_destroy = true, "+
					"and a request made again after a reject or revoke waits for a change of review or decision in a PR.")
		}
	}
}
