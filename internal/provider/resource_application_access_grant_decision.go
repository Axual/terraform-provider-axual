package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	_           resource.ResourceWithModifyPlan     = &grantDecisionResource{}
	_           resource.ResourceWithImportState    = &grantDecisionResource{}
	_           resource.ResourceWithIdentity       = &grantDecisionResource{}
	_           resource.ResourceWithValidateConfig = &grantDecisionResource{}
	_           resource.ResourceWithMoveState      = &grantDecisionResource{}
	lookupPaths                                     = []path.Path{path.Root("application"), path.Root("topic"), path.Root("environment"), path.Root("access_type")}
)

func NewApplicationAccessGrantDecisionResource(provider AxualProvider) resource.Resource {
	return &grantDecisionResource{provider: provider}
}

type grantDecisionResource struct {
	provider AxualProvider
}

type grantDecisionData struct {
	Id              types.String `tfsdk:"id"`
	Application     types.String `tfsdk:"application"`
	Topic           types.String `tfsdk:"topic"`
	Environment     types.String `tfsdk:"environment"`
	AccessType      types.String `tfsdk:"access_type"`
	Decision        types.String `tfsdk:"decision"`
	Reason          types.String `tfsdk:"reason"`
	ApproveAgain    types.String `tfsdk:"approve_again"`
	RevokeOnDestroy types.Bool   `tfsdk:"revoke_on_destroy"`
	GrantId         types.String `tfsdk:"grant_id"`
	CurrentStatus   types.String `tfsdk:"current_status"`
}

type grantDecisionIdentity struct {
	Topic       types.String `tfsdk:"topic"`
	Environment types.String `tfsdk:"environment"`
	AccessType  types.String `tfsdk:"access_type"`
	Application types.String `tfsdk:"application"`
}

// grantDecisionPrivate is kept in private state. Read stores what it saw of the grant, so a plan
// for an existing block depends only on the state (a saved plan is planned again at apply).
type grantDecisionPrivate struct {
	CanApprove bool `json:"can_approve"`
	CanDeny    bool `json:"can_deny"`
	CanRevoke  bool `json:"can_revoke"`
	Known      bool `json:"known"`
	// Reopened, OwnClose and ClosedBy describe the grant's last close (see the rules).
	Reopened bool   `json:"reopened"`
	OwnClose bool   `json:"own_close"`
	ClosedBy string `json:"closed_by,omitempty"`
	// LastClose is the closeKey Read saw; the apply compares it before it acts.
	LastClose *string `json:"last_close,omitempty"`
	// AcceptedClose is the closeKey of the grant when decision or approve_again last changed: the
	// topic team accepted that close, so a request made again after it is approved.
	AcceptedClose *string `json:"accepted_close,omitempty"`
}

const grantDecisionPrivateKey = "grant_decision"

type privateGetter interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

type privateSetter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

func readDecisionPrivate(ctx context.Context, p privateGetter) grantDecisionPrivate {
	var out grantDecisionPrivate
	if b, _ := p.GetKey(ctx, grantDecisionPrivateKey); len(b) > 0 {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

// observed is what the provider stores about a grant (nil: none found).
func (r *grantDecisionResource) observed(g *webclient.GrantView, accepted *string) grantDecisionPrivate {
	empty := ""
	p := grantDecisionPrivate{Known: true, LastClose: &empty, AcceptedClose: accepted}
	if g != nil {
		key := closeKey(*g)
		p = grantDecisionPrivate{CanApprove: g.CanApprove, CanDeny: g.CanDeny, CanRevoke: g.CanRevoke, Known: true,
			Reopened: reopened(*g), OwnClose: ownClose(g.ProcessedBy, g.RequestedBy, r.provider.grants.myEmail()),
			ClosedBy: g.ProcessedBy, LastClose: &key, AcceptedClose: accepted}
	}
	return p
}

func writeDecisionPrivate(ctx context.Context, s privateSetter, p grantDecisionPrivate, diags *diag.Diagnostics) {
	b, _ := json.Marshal(p)
	diags.Append(s.SetKey(ctx, grantDecisionPrivateKey, b)...)
}

func (r *grantDecisionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_access_grant_decision"
}

func (r *grantDecisionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	lookup := func(desc string, v ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: desc, Required: true, Validators: v}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "The topic team's decision about one application's access to a topic in an environment. " +
			"`APPROVED` approves every request from the application, `DENIED` rejects or revokes. After someone other than the application team " +
			"rejects or revokes the grant, a request made again waits for a change of `approve_again`. Nothing about the grant's status fails a plan. " +
			"Read more: https://registry.terraform.io/providers/Axual/axual/latest/docs/guides/manage-application-access-to-topics",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "`<topic>/<environment>/<access_type>/<application>`, set at create and never changed. Import accepts it, or a grant UID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"application": lookup("Application UID or name. The grant is found among the grants on the topic, also before the request exists. " +
				"In a single repository, use `axual_application_access_grant.x.application`."),
			"topic":       lookup("Topic UID."),
			"environment": lookup("Environment UID."),
			"access_type": lookup("`CONSUMER` or `PRODUCER`.", stringvalidator.OneOf("CONSUMER", "PRODUCER")),
			"decision": schema.StringAttribute{
				MarkdownDescription: "`APPROVED` approves every `Pending` request from the application, except a request made again after someone other than " +
					"the application team rejected or revoked the grant (see `approve_again`). `DENIED` rejects a `Pending` request and revokes an `Approved` grant.",
				Required:   true,
				Validators: []validator.String{stringvalidator.OneOf(decisionApproved, decisionDenied)},
			},
			"reason": schema.StringAttribute{
				MarkdownDescription: "Required when `decision = \"DENIED\"`. Sent with the reject or revoke; the application team reads it as `decision_reason`.",
				Optional:            true,
			},
			"approve_again": schema.StringAttribute{
				MarkdownDescription: "Free text, for example the PR number. After the topic team (or an admin) rejected or revoked the grant, a request made again " +
					"is approved only when `approve_again` or `decision` changed since that close. Adding or importing the block doesn't count. Not sent to the platform.",
				Optional: true,
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
// It uses the lookup attributes; a decision moved from an approval or rejection only knows grant_id
// until the first Read.
func (r *grantDecisionResource) find(d grantDecisionData, fresh bool) (*webclient.GrantView, error) {
	if known(d.Topic) && known(d.Environment) && known(d.AccessType) && known(d.Application) {
		grants, err := r.provider.grants.search(d.Topic.ValueString(), d.Environment.ValueString(), d.AccessType.ValueString(), fresh)
		if err != nil {
			return nil, err
		}
		grantID := ""
		if known(d.GrantId) {
			grantID = d.GrantId.ValueString()
		}
		return matchGrant(grants, d.Application.ValueString(), grantID), nil
	}
	if known(d.GrantId) {
		g, err := r.provider.client.GetGrantView(d.GrantId.ValueString())
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
	if g != nil {
		d.CurrentStatus, d.GrantId = types.StringValue(g.Status), types.StringValue(g.Uid)
		if app := d.Application.ValueString(); app != "" && !strings.EqualFold(app, g.ApplicationUid) && g.ApplicationName != "" && app != g.ApplicationName {
			resp.Diagnostics.AddWarning(fmt.Sprintf("Application %q is now named %q", app, g.ApplicationName),
				fmt.Sprintf("The decision still finds its grant (%s). Update application to %q, or to the application UID %s.", g.Uid, g.ApplicationName, g.ApplicationUid))
		}
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
		writeDecisionPrivate(ctx, resp.Private, r.observed(g, readDecisionPrivate(ctx, req.Private).AcceptedClose), &resp.Diagnostics)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
	r.setIdentity(ctx, d, resp.Identity, &resp.Diagnostics)
}

// acceptsClose reports whether this plan or apply accepts the grant's current close: decision or
// approve_again changed. A create or an import doesn't count, so adding blocks (for example a bulk
// import) can't give back access that someone took away.
func acceptsClose(plan, state grantDecisionData, create bool) bool {
	if create || state.Decision.IsNull() {
		return false
	}
	return !plan.Decision.Equal(state.Decision) || !plan.ApproveAgain.Equal(state.ApproveAgain)
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
	p := readDecisionPrivate(ctx, req.Private)

	// The grant this plan is about. On create or after a lookup change, look it up; otherwise use only
	// what Read stored. Terraform plans again when it applies a saved plan, so an update plan must not
	// depend on a new API call: the same state must give the same plan. The apply looks the grant up
	// itself and fails before acting when it changed.
	lookupChanged := !create && (!plan.Application.Equal(state.Application) || !plan.Topic.Equal(state.Topic) ||
		!plan.Environment.Equal(state.Environment) || !plan.AccessType.Equal(state.AccessType))
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
	}

	if lookupChanged {
		// Replace only when the edit points at another grant. Name to UID of the same application
		// stays in place.
		if g == nil || !known(state.GrantId) || g.Uid != state.GrantId.ValueString() {
			resp.RequiresReplace = append(resp.RequiresReplace, lookupPaths...)
			if state.CurrentStatus.ValueString() == "Approved" && !state.RevokeOnDestroy.ValueBool() {
				resp.Diagnostics.AddWarning(fmt.Sprintf("The old grant %s stays Approved", r.label(state, nil)),
					"This change points the decision at another grant, so the block is replaced and the old grant is no longer managed. "+
						"To remove its access, set decision = \"DENIED\" on it in an earlier PR, or set revoke_on_destroy = true.")
			}
			create = true
		}
	}
	if create {
		plan.Id = types.StringUnknown()
		if !unknownLookup(plan) {
			plan.Id = types.StringValue(compositeDecisionID(plan))
		}
	}

	if g != nil { // looked up in this plan
		accepted := p.AcceptedClose
		p = r.observed(g, accepted)
	} else if known(state.GrantId) {
		g = &webclient.GrantView{Uid: state.GrantId.ValueString(), Status: status, ProcessedBy: p.ClosedBy}
	}
	if create {
		p.AcceptedClose = nil
	}
	out := decide(decisionInput{
		Status:   status,
		Decision: plan.Decision.ValueString(),
		Reopened: status == "Pending" && p.Reopened,
		OwnClose: p.OwnClose,
		Accepted: acceptsClose(plan, state, create) || (p.AcceptedClose != nil && p.LastClose != nil && *p.AcceptedClose == *p.LastClose),
	})
	plan.CurrentStatus = types.StringValue(out.PlannedStatus)
	plan.GrantId = types.StringNull()
	if g != nil {
		plan.GrantId = types.StringValue(g.Uid)
	}
	if create {
		// The request can arrive or change between a saved plan and its apply, and Terraform checks the
		// apply's plan against the saved one. So a new block only warns what it will do.
		plan.CurrentStatus, plan.GrantId = types.StringUnknown(), types.StringUnknown()
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
	r.warn(plan, g, out, p, resp)
}

// planUnknown plans a decision whose grant can't be known yet, for example a single repository
// that creates the application and the grant in the same apply.
func (r *grantDecisionResource) planUnknown(ctx context.Context, plan *grantDecisionData, resp *resource.ModifyPlanResponse, create bool) {
	if create {
		plan.Id = types.StringUnknown()
	}
	plan.CurrentStatus, plan.GrantId = types.StringUnknown(), types.StringUnknown()
	resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
}

func (r *grantDecisionResource) warn(plan grantDecisionData, g *webclient.GrantView, out decisionOutcome, p grantDecisionPrivate, resp *resource.ModifyPlanResponse) {
	name := r.label(plan, g)
	closedBy := "someone other than the application team"
	if p.ClosedBy != "" {
		closedBy = p.ClosedBy
	}
	switch out.Warning {
	case warnWaitingForRequest:
		if plan.Decision.ValueString() == decisionApproved && r.provider.grants.hidesPrivateRequests() {
			cause := "If the application is private, its request may exist: before Platform Manager 15.0.4 a topic owner can't see the requests of private applications."
			if env := r.provider.grants.environment(plan.Environment.ValueString()); env != nil && strings.EqualFold(env.Visibility, "Private") {
				cause = fmt.Sprintf("Environment %s is private: before Platform Manager 15.0.4 a topic owner can't see any request in a private environment.", env.Name)
			}
			resp.Diagnostics.AddWarning(fmt.Sprintf("%s: no request found that this user can see", name),
				cause+" Approve such a request in the UI, or upgrade to 15.0.4 or later. "+
					"Otherwise the application team hasn't requested access yet; the topic repository's next apply after the request approves it.")
		} else if plan.Decision.ValueString() == decisionApproved {
			resp.Diagnostics.AddWarning(fmt.Sprintf("%s is waiting for a request", name),
				"No request found that you can see. The application team may not have requested access yet; the topic repository's next apply after the request approves it.")
		}
	case warnApprovedAgain:
		resp.Diagnostics.AddWarning(fmt.Sprintf("%s will be approved again", name),
			fmt.Sprintf("The grant was closed by %s, who also asked again: the application team's own close needs no new review. "+
				"After a reject or revoke by the topic team, a request made again waits for a change of approve_again.", closedBy))
	case warnReopened:
		resp.Diagnostics.AddWarning(fmt.Sprintf("%s: asked again after a close by %s", name, closedBy),
			"The application team requested access again after the grant was rejected, revoked or cancelled by someone else. "+
				"Change approve_again (for example to this PR's number) in a PR to approve it, or set decision = \"DENIED\" to reject it.")
	case warnDeniedAgain:
		resp.Diagnostics.AddWarning(fmt.Sprintf("%s asked again; DENIED rejects it again", name),
			fmt.Sprintf("The application team requested access again. decision = \"DENIED\" rejects the new request with the reason %q. "+
				"To give access, change decision to \"APPROVED\" in a PR; the application team then asks again.", plan.Reason.ValueString()))
	case warnClosed:
		if out.NextApproved {
			resp.Diagnostics.AddWarning(fmt.Sprintf("%s is %s, the next request will be approved", name, out.PlannedStatus),
				"When the application team requests again, the topic repository's next apply approves it without a new PR.")
		} else {
			resp.Diagnostics.AddWarning(fmt.Sprintf("%s is %s (by %s), a new request will wait", name, out.PlannedStatus, closedBy),
				"When the application team requests again, change approve_again in a PR to approve it. To keep the grant closed, set decision = \"DENIED\".")
		}
	}

	if out.Action != actionNone && p.Known {
		offered := map[string]bool{actionApprove: p.CanApprove, actionReject: p.CanDeny, actionRevoke: p.CanRevoke}[out.Action]
		if !offered {
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
		v.ApplicationName = d.GrantId.ValueString()
	}
	return grantName(v)
}

func (r *grantDecisionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan grantDecisionData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, plan, grantDecisionData{}, true, grantDecisionPrivate{}, &resp.State, resp.Private, resp.Identity, &resp.Diagnostics)
}

func (r *grantDecisionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state grantDecisionData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, plan, state, false, readDecisionPrivate(ctx, req.Private), &resp.State, resp.Private, resp.Identity, &resp.Diagnostics)
}

// apply looks the grant up again, checks that it is still what the plan was made from, runs the
// action the rules ask for, and stores the result.
func (r *grantDecisionResource) apply(ctx context.Context, plan, state grantDecisionData, create bool, p grantDecisionPrivate,
	out *tfsdk.State, private privateSetter, identity *tfsdk.ResourceIdentity, diags *diag.Diagnostics) {
	g, err := r.find(plan, true)
	if err != nil {
		diags.AddError("Failed to find the application access grant", err.Error())
		return
	}
	status, key := statusNotFound, ""
	if g != nil {
		status, key = g.Status, closeKey(*g)
		fillUnknown(&plan.Topic, g.TopicUid)
		fillUnknown(&plan.Environment, g.EnvironmentUid)
		fillUnknown(&plan.AccessType, g.AccessType)
		fillUnknown(&plan.Application, firstNonEmptyString(g.ApplicationName, g.ApplicationUid))
	}
	if plan.Id.IsUnknown() {
		plan.Id = types.StringValue(compositeDecisionID(plan))
	}
	// An existing block was planned from the state Read stored. If the grant changed since, the plan
	// may no longer fit (for example approved in the UI after a DENIED plan), so act on nothing.
	if !create && (status != state.CurrentStatus.ValueString() || (p.LastClose != nil && *p.LastClose != key)) {
		diags.AddError("The grant changed between plan and apply",
			fmt.Sprintf("The plan was made when the grant was %q; it is now %q, or it was closed again. Nothing was changed. Run plan again.",
				state.CurrentStatus.ValueString(), status))
		return
	}
	accepted := p.AcceptedClose
	if create {
		accepted = nil
	}
	if acceptsClose(plan, state, create) {
		accepted = &key
	}
	o := decide(decisionInput{
		Status:   status,
		Decision: plan.Decision.ValueString(),
		Reopened: g != nil && reopened(*g),
		OwnClose: g != nil && ownClose(g.ProcessedBy, g.RequestedBy, r.provider.grants.myEmail()),
		Accepted: accepted != nil && *accepted == key,
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
	plan.CurrentStatus, plan.GrantId = types.StringValue(statusNotFound), types.StringNull()
	if g != nil {
		if o.Action != actionNone {
			if fresh, err := r.provider.client.GetGrantView(g.Uid); err == nil {
				g = fresh
			}
		}
		plan.CurrentStatus, plan.GrantId = types.StringValue(g.Status), types.StringValue(g.Uid)
	}
	if known(planned) && planned.ValueString() != plan.CurrentStatus.ValueString() {
		diags.AddError("The grant changed during the apply",
			fmt.Sprintf("The plan expected status %q, but the grant is now %q. Run plan again.", planned.ValueString(), plan.CurrentStatus.ValueString()))
		return
	}
	diags.Append(out.Set(ctx, &plan)...)
	if private != nil {
		writeDecisionPrivate(ctx, private, r.observed(g, accepted), diags)
	}
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
	d.Decision, d.Reason, d.ApproveAgain = types.StringNull(), types.StringNull(), types.StringNull()
	d.GrantId, d.CurrentStatus = types.StringNull(), types.StringNull()
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
	return d.Application.IsUnknown() || d.Topic.IsUnknown() || d.Environment.IsUnknown() || d.AccessType.IsUnknown()
}

func fillUnknown(v *types.String, value string) {
	if (v.IsUnknown() || v.IsNull()) && value != "" {
		*v = types.StringValue(value)
	}
}
