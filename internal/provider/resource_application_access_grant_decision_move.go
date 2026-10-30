package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MoveState moves axual_application_access_grant_approval and _rejection to the decision, with a
// moved block (Terraform 1.8 or later). It runs before the provider is configured, so it only copies
// the old state into grant_id; the next Read finds the grant by it and fills the lookup attributes and id. A moved block
// gets the default revoke_on_destroy = true, so removing it revokes, as removing an approval did, and the
// migration plan shows no diff.
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
			Application:     types.StringNull(),
			ApplicationUid:  types.StringNull(),
			Topic:           types.StringNull(),
			Environment:     types.StringNull(),
			AccessType:      types.StringNull(),
			Decision:        types.StringValue(decision),
			Reason:          stringOrNull(reason.ValueString()),
			RevokeOnDestroy: types.BoolValue(true),
			GrantId:         uid,
			CurrentStatus:   types.StringNull(),
		}
		resp.Diagnostics.Append(resp.TargetState.Set(ctx, &d)...)
	}
}
