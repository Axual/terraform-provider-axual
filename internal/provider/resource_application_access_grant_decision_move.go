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
// moved block (Terraform 1.8 or later). It runs before the provider is configured, so it only copies
// the old state; the next Read fills the lookup attributes and id from the grant. A moved approval
// keeps today's behaviour, revoke_on_destroy = true: a config without it shows the change to false
// as a diff in the migration PR, instead of hiding it in a warning.
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
			d.RevokeOnDestroy = types.BoolValue(true) // removing an approval revoked the grant
		}
		resp.Diagnostics.Append(resp.TargetState.Set(ctx, &d)...)
		if decision == decisionApproved {
			resp.Diagnostics.AddWarning(fmt.Sprintf("Moved the approval of grant %s to a decision", uid.ValueString()),
				"revoke_on_destroy = true is kept, so removing the block still revokes the grant. If the configuration doesn't set "+
					"revoke_on_destroy = true, this plan changes it to false: then removing the block no longer revokes. "+
					"One thing works differently: a request made again after the topic team rejected or revoked waits for a change of review or decision in a PR.")
		}
	}
}
