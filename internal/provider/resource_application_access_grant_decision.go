package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_           resource.ResourceWithModifyPlan       = &grantDecisionResource{}
	_           resource.ResourceWithImportState      = &grantDecisionResource{}
	_           resource.ResourceWithIdentity         = &grantDecisionResource{}
	_           resource.ResourceWithConfigValidators = &grantDecisionResource{}
	_           resource.ResourceWithValidateConfig   = &grantDecisionResource{}
	_           resource.ResourceWithMoveState        = &grantDecisionResource{}
	lookupPaths                                       = []path.Path{path.Root("application"), path.Root("topic"), path.Root("environment"), path.Root("access_type")}
)

func NewApplicationAccessGrantDecisionResource(provider AxualProvider) resource.Resource {
	return &grantDecisionResource{provider: provider}
}

type grantDecisionResource struct {
	provider AxualProvider
}

type grantDecisionData struct {
	Id              types.String `tfsdk:"id"`
	Grant           types.String `tfsdk:"grant"`
	Application     types.String `tfsdk:"application"`
	Topic           types.String `tfsdk:"topic"`
	Environment     types.String `tfsdk:"environment"`
	AccessType      types.String `tfsdk:"access_type"`
	Decision        types.String `tfsdk:"decision"`
	Reason          types.String `tfsdk:"reason"`
	Review          types.String `tfsdk:"review"`
	RevokeOnDestroy types.Bool   `tfsdk:"revoke_on_destroy"`
	GrantId         types.String `tfsdk:"grant_id"`
	CurrentStatus   types.String `tfsdk:"current_status"`
	NextRequest     types.String `tfsdk:"next_request"`
}

type grantDecisionIdentity struct {
	Topic       types.String `tfsdk:"topic"`
	Environment types.String `tfsdk:"environment"`
	AccessType  types.String `tfsdk:"access_type"`
	Application types.String `tfsdk:"application"`
}

// grantDecisionPrivate is kept in private state by Read and used by ModifyPlan.
type grantDecisionPrivate struct {
	CanApprove bool `json:"can_approve"`
	CanDeny    bool `json:"can_deny"`
	CanRevoke  bool `json:"can_revoke"`
	Known      bool `json:"known"`
}

const grantDecisionPrivateKey = "grant_decision"

func (r *grantDecisionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_access_grant_decision"
}

func (r *grantDecisionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	lookup := func(desc string, v ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: desc,
			Optional:            true,
			Computed:            true,
			Validators:          v,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "The topic team's decision about one application's access to a topic in an environment. " +
			"The provider approves, rejects or revokes the grant to match `decision`. Nothing about the grant's status fails a plan. " +
			"Read more: https://registry.terraform.io/providers/Axual/axual/latest/docs/guides/manage-application-access-to-topics",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "`<topic>/<environment>/<access_type>/<application>`, set at create and never changed. Import accepts it, or a grant UID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"grant": schema.StringAttribute{
				MarkdownDescription: "Grant UID, for a single repository: `axual_application_access_grant.x.id`. Set either `grant` or all of `application`, `topic`, `environment` and `access_type`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application": lookup("Application UID or name. The grant is found among the grants on the topic, so private applications work (Platform Manager 15.0.4 or later)."),
			"topic":       lookup("Topic UID."),
			"environment": lookup("Environment UID."),
			"access_type": lookup("`CONSUMER` or `PRODUCER`.", stringvalidator.OneOf("CONSUMER", "PRODUCER")),
			"decision": schema.StringAttribute{
				MarkdownDescription: "`APPROVED` or `DENIED`. `DENIED` rejects a `Pending` request and revokes an `Approved` grant. `APPROVED` approves a `Pending` request when this plan reviews it (see `review` and `next_request`).",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf(decisionApproved, decisionDenied)},
			},
			"reason": schema.StringAttribute{
				MarkdownDescription: "Required when `decision = \"DENIED\"`. Sent with the reject or revoke; the application team reads it as `decision_reason`.",
				Optional:            true,
			},
			"review": schema.StringAttribute{
				MarkdownDescription: "Free text, for example the PR number. Changing it approves the current `Pending` request, such as a request made again after a reject or revoke. Not sent to the platform.",
				Optional:            true,
			},
			"revoke_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "When `true`, removing the block revokes an `Approved` grant. Default `false`: removing the block only stops managing the decision.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"grant_id": schema.StringAttribute{
				MarkdownDescription: "The grant UID. Null until the application team requests access; changes when the application is created again.",
				Computed:            true,
			},
			"current_status": schema.StringAttribute{
				MarkdownDescription: "The grant's status: `Pending`, `Approved`, `Rejected`, `Revoked`, `Cancelled`, or `Not found` when there is no request that this user can see.",
				Computed:            true,
			},
			"next_request": schema.StringAttribute{
				MarkdownDescription: "`approve`: the next `Pending` request is approved without a new PR (set when the block is created, or `decision` changes to `APPROVED`, while nothing is `Pending`). `review`: a new request waits for a change of `review` or `decision`. Null for `DENIED`.",
				Computed:            true,
			},
		},
	}
}

func (r *grantDecisionResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	attr := func(d string) identityschema.StringAttribute {
		return identityschema.StringAttribute{RequiredForImport: true, Description: d}
	}
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"topic":       attr("Topic UID."),
			"environment": attr("Environment UID."),
			"access_type": attr("CONSUMER or PRODUCER."),
			"application": attr("Application UID or name."),
		},
	}
}

func (r *grantDecisionResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(path.MatchRoot("grant"), path.MatchRoot("application")),
		resourcevalidator.RequiredTogether(path.MatchRoot("application"), path.MatchRoot("topic"), path.MatchRoot("environment"), path.MatchRoot("access_type")),
		resourcevalidator.Conflicting(path.MatchRoot("grant"), path.MatchRoot("topic")),
		resourcevalidator.Conflicting(path.MatchRoot("grant"), path.MatchRoot("environment")),
		resourcevalidator.Conflicting(path.MatchRoot("grant"), path.MatchRoot("access_type")),
	}
}

func (r *grantDecisionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var decision, reason types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("decision"), &decision)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("reason"), &reason)...)
	if decision.ValueString() == decisionDenied && !reason.IsUnknown() && strings.TrimSpace(reason.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(path.Root("reason"), "Missing reason",
			`Set reason when decision = "DENIED". The application team reads it, and the platform refuses a reject or revoke without one.`)
	}
}

// find returns the grant the decision is about, or nil when there is none that this user can see.
func (r *grantDecisionResource) find(d grantDecisionData, fresh bool) (*webclient.GrantView, error) {
	if known(d.Topic) && known(d.Environment) && known(d.AccessType) && known(d.Application) {
		grants, err := r.provider.grants.search(d.Topic.ValueString(), d.Environment.ValueString(), d.AccessType.ValueString(), fresh)
		if err != nil {
			return nil, err
		}
		return matchGrant(grants, d.Application.ValueString()), nil
	}
	if known(d.Grant) {
		g, err := r.provider.client.GetGrantView(d.Grant.ValueString())
		if err != nil {
			if isGone(err) || isForbidden(err) {
				return nil, nil
			}
			return nil, err
		}
		return g, nil
	}
	return nil, nil
}

func known(v types.String) bool { return !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" }

func isForbidden(err error) bool {
	return err != nil && strings.Contains(err.Error(), "403")
}

func (r *grantDecisionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var d grantDecisionData
	resp.Diagnostics.Append(req.State.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	g, err := r.find(d, false)
	if err != nil {
		resp.Diagnostics.AddError("Failed to find the application access grant", err.Error())
		return
	}
	d.CurrentStatus, d.GrantId = types.StringValue(statusNotFound), types.StringNull()
	p := grantDecisionPrivate{Known: true}
	if g != nil {
		d.CurrentStatus, d.GrantId = types.StringValue(g.Status), types.StringValue(g.Uid)
		p = grantDecisionPrivate{CanApprove: g.CanApprove, CanDeny: g.CanDeny, CanRevoke: g.CanRevoke, Known: true}
		// A decision moved from an approval or rejection only knows the grant UID: MoveState runs
		// before the provider is configured, so the lookup attributes and id are filled here.
		fillUnknown(&d.Topic, g.TopicUid)
		fillUnknown(&d.Environment, g.EnvironmentUid)
		fillUnknown(&d.AccessType, g.AccessType)
		fillUnknown(&d.Application, firstNonEmptyString(g.ApplicationName, g.ApplicationUid))
		if !known(d.Id) && known(d.Topic) {
			d.Id = types.StringValue(compositeDecisionID(d))
		}
	}
	if resp.Private != nil { // nil when `terraform query` reads a found resource
		b, _ := json.Marshal(p)
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, grantDecisionPrivateKey, b)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
	r.setIdentity(ctx, d, resp.Identity, &resp.Diagnostics)
}

func (r *grantDecisionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.provider.grants == nil {
		return // provider not configured yet (validate)
	}
	var state grantDecisionData
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}
	if req.Plan.Raw.IsNull() {
		if state.CurrentStatus.ValueString() == "Approved" {
			if state.RevokeOnDestroy.ValueBool() {
				resp.Diagnostics.AddWarning(fmt.Sprintf("Removing the decision revokes %s", r.label(state, nil)),
					"revoke_on_destroy = true: the apply revokes the grant with the reason \"Decision removed from Terraform\".")
			} else {
				resp.Diagnostics.AddWarning(fmt.Sprintf("Removing the decision leaves %s Approved", r.label(state, nil)),
					"Set decision = \"DENIED\" (with a reason) to revoke, or set revoke_on_destroy = true before removing the block.")
			}
		}
		return
	}
	var plan grantDecisionData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	create := req.State.Raw.IsNull()
	var configGrant types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("grant"), &configGrant)...)
	if configGrant.IsNull() && plan.Grant.IsUnknown() {
		plan.Grant = types.StringNull()
	}

	// The grant this plan is about. On create or after a lookup change, look it up; otherwise use only
	// what Read stored. Terraform plans again when it applies a saved plan, so an update plan must not
	// depend on a new API call: the same state must give the same plan. The apply looks the grant up
	// itself and fails with the real status when it changed.
	lookupChanged := !create && ((!configGrant.IsNull() && !plan.Grant.Equal(state.Grant)) || !plan.Application.Equal(state.Application) ||
		!plan.Topic.Equal(state.Topic) || !plan.Environment.Equal(state.Environment) || !plan.AccessType.Equal(state.AccessType))
	var g *webclient.GrantView
	status := state.CurrentStatus.ValueString()
	if create || lookupChanged {
		if unknownLookup(plan) {
			r.planUnknown(ctx, &plan, resp, create)
			return
		}
		var err error
		if g, err = r.find(plan, false); err != nil {
			resp.Diagnostics.AddError("Failed to find the application access grant", err.Error())
			return
		}
		status = statusNotFound
		if g != nil {
			status = g.Status
		}
	} else if known(state.GrantId) {
		g = &webclient.GrantView{Uid: state.GrantId.ValueString(), Status: status}
	}

	if lookupChanged {
		// Replace only when the edit points at another grant. Name to UID of the same application,
		// or the grant form to the lookup form, stays in place.
		if g == nil || !known(state.GrantId) || g.Uid != state.GrantId.ValueString() {
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("grant"))
			resp.RequiresReplace = append(resp.RequiresReplace, lookupPaths...)
			if state.CurrentStatus.ValueString() == "Approved" && !state.RevokeOnDestroy.ValueBool() {
				resp.Diagnostics.AddWarning(fmt.Sprintf("The old grant %s stays Approved", r.label(state, nil)),
					"This change points the decision at another grant, so the block is replaced and the old grant is no longer managed. "+
						"To remove its access, set decision = \"DENIED\" on it in an earlier PR, or set revoke_on_destroy = true.")
			}
			create = true
		}
	}

	// Fill the lookup attributes in the grant form, and the id on create.
	if g != nil && g.TopicUid != "" {
		fillUnknown(&plan.Topic, g.TopicUid)
		fillUnknown(&plan.Environment, g.EnvironmentUid)
		fillUnknown(&plan.AccessType, g.AccessType)
		fillUnknown(&plan.Application, firstNonEmptyString(g.ApplicationName, g.ApplicationUid))
	}
	if create {
		plan.Id = types.StringUnknown()
		if !unknownLookup(plan) && known(plan.Topic) {
			plan.Id = types.StringValue(compositeDecisionID(plan))
		}
	}

	out := decide(decisionInput{
		Status:          status,
		Decision:        plan.Decision.ValueString(),
		Create:          create,
		DecisionChanged: !create && !plan.Decision.Equal(state.Decision),
		ReviewChanged:   !create && !plan.Review.Equal(state.Review),
		NextRequest:     state.NextRequest.ValueString(),
		Imported:        !create && state.Decision.IsNull(),
	})
	plan.CurrentStatus = types.StringValue(out.PlannedStatus)
	plan.NextRequest = stringOrNull(out.NextRequest)
	plan.GrantId = types.StringNull()
	if g != nil {
		plan.GrantId = types.StringValue(g.Uid)
	}
	if create {
		// The request can arrive or change between a saved plan and its apply, and Terraform checks the
		// apply's plan against the saved one. So a new block only warns what it will do.
		plan.CurrentStatus, plan.GrantId, plan.NextRequest = types.StringUnknown(), types.StringUnknown(), types.StringUnknown()
		if out.Action != actionNone {
			resp.Diagnostics.AddWarning(fmt.Sprintf("%s will be %s by this apply", r.label(plan, g), map[string]string{
				actionApprove: "approved", actionReject: "rejected", actionRevoke: "revoked"}[out.Action]),
				fmt.Sprintf("The grant is %s now.", status))
		}
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

	if g != nil && known(plan.Id) {
		if other := r.provider.grants.claim(g.Uid, plan.Id.ValueString()); other != "" {
			resp.Diagnostics.AddWarning(fmt.Sprintf("Two decisions manage %s", r.label(plan, g)),
				fmt.Sprintf("The decisions with id %q and %q point at the same grant. Keep one of them: with two, the result depends on the order of the apply.", other, plan.Id.ValueString()))
		}
	}
	r.warn(ctx, req, plan, g, out, resp)
}

// planUnknown plans a decision whose grant can't be known yet, for example a single repository
// that creates the grant in the same apply.
func (r *grantDecisionResource) planUnknown(ctx context.Context, plan *grantDecisionData, resp *resource.ModifyPlanResponse, create bool) {
	if create {
		plan.Id = types.StringUnknown()
	}
	plan.CurrentStatus, plan.GrantId, plan.NextRequest = types.StringUnknown(), types.StringUnknown(), types.StringUnknown()
	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}

func (r *grantDecisionResource) warn(ctx context.Context, req resource.ModifyPlanRequest, plan grantDecisionData, g *webclient.GrantView, out decisionOutcome, resp *resource.ModifyPlanResponse) {
	name := r.label(plan, g)
	switch out.Warning {
	case warnWaitingForRequest:
		if plan.Decision.ValueString() == decisionApproved && r.provider.grants.hidesPrivateRequests() {
			resp.Diagnostics.AddWarning(fmt.Sprintf("%s: no request found that this user can see", name),
				"This Platform Manager is older than 15.0.4: a topic owner can't see the requests of private applications. "+
					"If the application is private, its request may exist; approve it in the UI, or upgrade to 15.0.4 or later. "+
					"Otherwise the application team hasn't requested access yet; the next request is approved without a new PR (next_request = \"approve\").")
		} else if plan.Decision.ValueString() == decisionApproved {
			resp.Diagnostics.AddWarning(fmt.Sprintf("%s is waiting for a request", name),
				"No request found that you can see. The application team may not have requested access yet; the next request is approved without a new PR (next_request = \"approve\").")
		}
	case warnReopened:
		resp.Diagnostics.AddWarning(fmt.Sprintf("%s: new request since the last decision", name),
			"The application team requested access again. Change review (for example to this PR's number) or decision in a PR to approve it.")
	case warnClosed:
		if plan.NextRequest.ValueString() == nextRequestApprove {
			resp.Diagnostics.AddWarning(fmt.Sprintf("%s is %s, the next request will be approved", name, plan.CurrentStatus.ValueString()),
				"next_request = \"approve\": when the application team requests again, the next apply approves it without a new PR.")
		} else {
			resp.Diagnostics.AddWarning(fmt.Sprintf("%s is %s, waiting for a new request", name, plan.CurrentStatus.ValueString()),
				"When the application team requests again, change review or decision in a PR to approve it.")
		}
	}

	if out.Action != actionNone {
		var p grantDecisionPrivate
		if b, _ := req.Private.GetKey(ctx, grantDecisionPrivateKey); len(b) > 0 {
			_ = json.Unmarshal(b, &p)
		}
		offered := map[string]bool{actionApprove: p.CanApprove, actionReject: p.CanDeny, actionRevoke: p.CanRevoke}[out.Action]
		if g != nil && (g.CanApprove || g.CanDeny || g.CanRevoke) {
			offered = map[string]bool{actionApprove: g.CanApprove, actionReject: g.CanDeny, actionRevoke: g.CanRevoke}[out.Action]
		}
		if p.Known && !offered {
			resp.Diagnostics.AddWarning(fmt.Sprintf("%s: the platform doesn't offer %s to this user now", name, out.Action),
				"The grant has no \""+out.Action+"\" link for the signed-in user. Possible reasons: the user doesn't own the topic, or self-approval is disabled on the instance and this user made the request. The apply still tries.")
		}
	}

	if plan.Decision.ValueString() == decisionDenied && known(plan.Environment) {
		if env := r.provider.grants.environment(plan.Environment.ValueString()); env != nil && strings.EqualFold(env.AuthorizationIssuer, "Auto") {
			resp.Diagnostics.AddWarning(fmt.Sprintf("%s: DENIED doesn't last in an Auto environment", name),
				"In an Auto environment the platform approves every new request straight away. Use a Stream owner environment for the topic team to decide.")
		}
	}
}

// label names the decision's grant in messages: "Log Scraper (CONSUMER on logs, dev)".
func (r *grantDecisionResource) label(d grantDecisionData, g *webclient.GrantView) string {
	v := webclient.GrantView{ApplicationName: d.Application.ValueString(), AccessType: d.AccessType.ValueString(),
		TopicUid: d.Topic.ValueString(), EnvironmentUid: d.Environment.ValueString()}
	if g != nil && g.ApplicationName != "" {
		v = *g
	} else if r.provider.grants != nil && known(d.Topic) && known(d.Environment) && known(d.AccessType) {
		if grants, err := r.provider.grants.search(d.Topic.ValueString(), d.Environment.ValueString(), d.AccessType.ValueString(), false); err == nil {
			for _, found := range grants {
				if found.Uid == d.GrantId.ValueString() || found.ApplicationUid == d.Application.ValueString() || found.ApplicationName == d.Application.ValueString() {
					v = found
					break
				}
			}
		}
	}
	if v.TopicName == "" && r.provider.grants != nil && known(d.Topic) {
		v.TopicName = r.provider.grants.topicName(d.Topic.ValueString())
	}
	if v.EnvironmentName == "" && r.provider.grants != nil && known(d.Environment) {
		if env := r.provider.grants.environment(d.Environment.ValueString()); env != nil {
			v.EnvironmentName = env.Name
		}
	}
	if v.ApplicationName == "" {
		v.ApplicationName = d.Grant.ValueString()
	}
	return grantName(v)
}

func (r *grantDecisionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan grantDecisionData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, plan, grantDecisionData{}, true, &resp.State, resp.Identity, &resp.Diagnostics)
}

func (r *grantDecisionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state grantDecisionData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, plan, state, false, &resp.State, resp.Identity, &resp.Diagnostics)
}

// apply looks the grant up again, runs the action the rules ask for, and stores the result.
func (r *grantDecisionResource) apply(ctx context.Context, plan, state grantDecisionData, create bool, out *tfsdk.State, identity *tfsdk.ResourceIdentity, diags *diag.Diagnostics) {
	g, err := r.find(plan, true)
	if err != nil {
		diags.AddError("Failed to find the application access grant", err.Error())
		return
	}
	status := statusNotFound
	if g != nil {
		status = g.Status
		fillUnknown(&plan.Topic, g.TopicUid)
		fillUnknown(&plan.Environment, g.EnvironmentUid)
		fillUnknown(&plan.AccessType, g.AccessType)
		fillUnknown(&plan.Application, firstNonEmptyString(g.ApplicationName, g.ApplicationUid))
	}
	if plan.Id.IsUnknown() {
		plan.Id = types.StringValue(compositeDecisionID(plan))
	}
	o := decide(decisionInput{
		Status:          status,
		Decision:        plan.Decision.ValueString(),
		Create:          create,
		DecisionChanged: !create && !plan.Decision.Equal(state.Decision),
		ReviewChanged:   !create && !plan.Review.Equal(state.Review),
		NextRequest:     state.NextRequest.ValueString(),
		Imported:        !create && state.Decision.IsNull(),
	})
	planned := plan.CurrentStatus
	switch o.Action {
	case actionApprove:
		tflog.Info(ctx, fmt.Sprintf("Approving grant %s", g.Uid))
		err = r.provider.client.ApproveGrant(g.Uid)
	case actionReject, actionRevoke:
		tflog.Info(ctx, fmt.Sprintf("Denying grant %s (%s)", g.Uid, o.Action))
		err = r.provider.client.RevokeOrDenyGrant(g.Uid, plan.Reason.ValueString())
	}
	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to %s %s", o.Action, grantName(*g)), err.Error())
		return
	}
	plan.CurrentStatus, plan.GrantId, plan.NextRequest = types.StringValue(statusNotFound), types.StringNull(), stringOrNull(o.NextRequest)
	if g != nil {
		if o.Action != actionNone {
			if fresh, err := r.provider.client.GetGrantView(g.Uid); err == nil {
				g = fresh
			}
		}
		plan.CurrentStatus, plan.GrantId = types.StringValue(g.Status), types.StringValue(g.Uid)
	}
	if known(planned) && planned.ValueString() != plan.CurrentStatus.ValueString() {
		diags.AddError("The grant changed between plan and apply",
			fmt.Sprintf("The plan expected status %q, but the grant is now %q. Run plan again.", planned.ValueString(), plan.CurrentStatus.ValueString()))
		return
	}
	diags.Append(out.Set(ctx, &plan)...)
	r.setIdentity(ctx, plan, identity, diags)
}

func (r *grantDecisionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state grantDecisionData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !state.RevokeOnDestroy.ValueBool() {
		return
	}
	g, err := r.find(state, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to find the application access grant", err.Error())
		return
	}
	if g != nil && g.Status == "Approved" {
		if err := r.provider.client.RevokeOrDenyGrant(g.Uid, "Decision removed from Terraform"); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to revoke %s", grantName(*g)), err.Error())
		}
	}
}

func (r *grantDecisionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var d grantDecisionData
	switch {
	case req.ID == "" && req.Identity != nil:
		var id grantDecisionIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &id)...)
		d.Topic, d.Environment, d.AccessType, d.Application = id.Topic, id.Environment, id.AccessType, id.Application
	case strings.Contains(req.ID, "/"):
		parts := strings.SplitN(req.ID, "/", 4)
		if len(parts) != 4 {
			resp.Diagnostics.AddError("Invalid import ID", "Use <topic>/<environment>/<access_type>/<application>, or a grant UID.")
			return
		}
		d.Topic, d.Environment, d.AccessType, d.Application = types.StringValue(parts[0]), types.StringValue(parts[1]),
			types.StringValue(strings.ToUpper(parts[2])), types.StringValue(parts[3])
	default:
		g, err := r.provider.client.GetGrantView(req.ID)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read the application access grant", err.Error())
			return
		}
		d.Topic, d.Environment, d.AccessType = types.StringValue(g.TopicUid), types.StringValue(g.EnvironmentUid), types.StringValue(g.AccessType)
		d.Application = types.StringValue(firstNonEmptyString(g.ApplicationName, g.ApplicationUid))
	}
	if resp.Diagnostics.HasError() {
		return
	}
	d.Id = types.StringValue(compositeDecisionID(d))
	d.RevokeOnDestroy = types.BoolValue(false)
	d.Grant, d.Decision, d.Reason, d.Review = types.StringNull(), types.StringNull(), types.StringNull(), types.StringNull()
	d.GrantId, d.CurrentStatus, d.NextRequest = types.StringNull(), types.StringNull(), types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
	r.setIdentity(ctx, d, resp.Identity, &resp.Diagnostics)
}

func (r *grantDecisionResource) setIdentity(ctx context.Context, d grantDecisionData, identity *tfsdk.ResourceIdentity, diags *diag.Diagnostics) {
	if identity == nil || diags.HasError() || !known(d.Id) {
		return
	}
	// The identity comes from id, which never changes, so it can't change either.
	parts := strings.SplitN(d.Id.ValueString(), "/", 4)
	if len(parts) != 4 {
		return
	}
	diags.Append(identity.Set(ctx, grantDecisionIdentity{
		Topic: types.StringValue(parts[0]), Environment: types.StringValue(parts[1]),
		AccessType: types.StringValue(parts[2]), Application: types.StringValue(parts[3]),
	})...)
}

// compositeDecisionID is <topic>/<environment>/<access_type>/<application>. The application is
// last, so a name with a "/" still splits into four parts.
func compositeDecisionID(d grantDecisionData) string {
	return strings.Join([]string{d.Topic.ValueString(), d.Environment.ValueString(), d.AccessType.ValueString(), d.Application.ValueString()}, "/")
}

func unknownLookup(d grantDecisionData) bool {
	if d.Grant.IsUnknown() {
		return true
	}
	if known(d.Grant) {
		return false
	}
	return d.Application.IsUnknown() || d.Topic.IsUnknown() || d.Environment.IsUnknown() || d.AccessType.IsUnknown()
}

func fillUnknown(v *types.String, value string) {
	if (v.IsUnknown() || v.IsNull()) && value != "" {
		*v = types.StringValue(value)
	}
}
