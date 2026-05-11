package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &applicationPrincipalResource{}
var _ resource.ResourceWithImportState = &applicationPrincipalResource{}

func NewApplicationPrincipalResource(provider AxualProvider) resource.Resource {
	return &applicationPrincipalResource{
		provider: provider,
	}
}

type applicationPrincipalResource struct {
	provider AxualProvider
}

type applicationPrincipalResourceData struct {
	Principal   types.String `tfsdk:"principal"`
	PrivateKey  types.String `tfsdk:"private_key"`
	Application types.String `tfsdk:"application"`
	Environment types.String `tfsdk:"environment"`
	Custom      types.Bool   `tfsdk:"custom"`
	Active      types.Bool   `tfsdk:"active"`
	Id          types.String `tfsdk:"id"`
}

func (r *applicationPrincipalResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_principal"
}

// trimSpaceSemanticallyEqual suppresses diffs caused only by surrounding whitespace.
// This is needed because the Create function trims whitespace before sending to the API,
// so the API returns a trimmed value, while the Terraform config (from file()) may include
// a trailing newline. Without this, every import would show a spurious replacement diff.
type trimSpaceSemanticallyEqual struct{}

func (m trimSpaceSemanticallyEqual) Description(_ context.Context) string {
	return "Suppresses diffs caused only by surrounding whitespace differences."
}

func (m trimSpaceSemanticallyEqual) MarkdownDescription(_ context.Context) string {
	return "Suppresses diffs caused only by surrounding whitespace differences."
}

func (m trimSpaceSemanticallyEqual) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if strings.TrimSpace(req.StateValue.ValueString()) == strings.TrimSpace(req.ConfigValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}

// isCertificateChanging returns true when principal or private_key differs between plan and state,
// indicating a certificate rotation is in progress. Used by plan modifiers for id and active
// to mark those attributes as (known after apply) so the plan/apply cycle is consistent.
func isCertificateChanging(ctx context.Context, plan tfsdk.Plan, state tfsdk.State) bool {
	if state.Raw.IsNull() {
		return false // Create operation — not an update
	}
	var planPrincipal, statePrincipal types.String
	plan.GetAttribute(ctx, path.Root("principal"), &planPrincipal)
	state.GetAttribute(ctx, path.Root("principal"), &statePrincipal)
	if !planPrincipal.IsNull() && !planPrincipal.IsUnknown() &&
		!statePrincipal.IsNull() && !statePrincipal.IsUnknown() &&
		strings.TrimSpace(planPrincipal.ValueString()) != strings.TrimSpace(statePrincipal.ValueString()) {
		return true
	}
	var planPrivateKey, statePrivateKey types.String
	plan.GetAttribute(ctx, path.Root("private_key"), &planPrivateKey)
	state.GetAttribute(ctx, path.Root("private_key"), &statePrivateKey)
	return !planPrivateKey.Equal(statePrivateKey)
}

// unknownWhenCertChangesString marks a string attribute as (known after apply)
// during certificate rotation so Terraform's plan matches what Update actually produces.
type unknownWhenCertChangesString struct{}

func (m unknownWhenCertChangesString) Description(_ context.Context) string {
	return "Marks attribute as unknown when the certificate is being rotated."
}
func (m unknownWhenCertChangesString) MarkdownDescription(_ context.Context) string {
	return "Marks attribute as unknown when the certificate is being rotated."
}
func (m unknownWhenCertChangesString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if isCertificateChanging(ctx, req.Plan, req.State) {
		resp.PlanValue = types.StringUnknown()
	}
}

// unknownWhenCertChangesBool marks a bool attribute as (known after apply)
// during certificate rotation so Terraform's plan matches what Update actually produces.
type unknownWhenCertChangesBool struct{}

func (m unknownWhenCertChangesBool) Description(_ context.Context) string {
	return "Marks attribute as unknown when the certificate is being rotated."
}
func (m unknownWhenCertChangesBool) MarkdownDescription(_ context.Context) string {
	return "Marks attribute as unknown when the certificate is being rotated."
}
func (m unknownWhenCertChangesBool) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() {
		return // user explicitly set active — respect it, don't override to unknown
	}
	if isCertificateChanging(ctx, req.Plan, req.State) {
		resp.PlanValue = types.BoolUnknown()
	}
}

func (r *applicationPrincipalResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An Application Principal is a security principal (certificate or comparable) that uniquely authenticates an Application in an Environment. Read more: https://docs.axual.io/axual/2026.1/self-service/application-management.html#configuring-application-securityauthentication",

		Attributes: map[string]schema.Attribute{
			"principal": schema.StringAttribute{
				MarkdownDescription: "The principal of an Application for an Environment. Must be PEM-format.",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					trimSpaceSemanticallyEqual{},
				},
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "The private key of a Connector Application for an Environment. Must be PEM-format. If committing terraform configuration(.tf) file in version control repository, please make sure there is a secure way of providing private key for a Connector application's Application Principal. Here are best practices for handling secrets in Terraform: https://blog.gitguardian.com/how-to-handle-secrets-in-terraform/.",
				Optional:            true,
				Sensitive:           true,
			},
			"application": schema.StringAttribute{
				MarkdownDescription: "A valid UID of an existing application",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "A valid Uid of an existing environment",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Activation intent for Connector application principals. `true` on create activates the principal immediately. During certificate rotation, activation is automatic for Connector applications unless explicitly set to `false`. This attribute is **not** refreshed from the API on Read — it reflects the last value set by Terraform, not live API state. Deleting an active principal is not allowed; activate another principal first.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					unknownWhenCertChangesBool{},
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application Principal ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					unknownWhenCertChangesString{},
				},
			},
			"custom": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "A boolean identifying whether we are creating a custom principal. If true, the custom principal will be stored in `principal` property. Custom principal allows an application with SASL+OAUTHBEARER to produce/consume a topic. Custom Application Principal certificate is used to authenticate your application with an IAM provider using the custom ApplicationPrincipal as Client ID",
			},
		},
	}
}

func (r *applicationPrincipalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data applicationPrincipalResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationPrincipalRequest, err := createApplicationPrincipalRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CREATE request struct for application principal resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	tflog.Info(ctx, "Create application principal request", map[string]interface{}{
		"application": applicationPrincipalRequest[0].Application,
		"environment": applicationPrincipalRequest[0].Environment,
		"custom":      applicationPrincipalRequest[0].Custom,
	})
	applicationPrincipal, err := r.provider.client.CreateApplicationPrincipal(applicationPrincipalRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for application principal resource", fmt.Sprintf("Error message: %s %s", applicationPrincipal, err))
		return
	}

	var trimmedResponse = strings.Trim(string(applicationPrincipal), "\"")
	returnedUid := strings.ReplaceAll(trimmedResponse, fmt.Sprintf("%s/%s", r.provider.client.ApiURL, "application_principals/"), "")

	data.Id = types.StringValue(returnedUid)

	if !data.Active.IsNull() && data.Active.ValueBool() {
		err = r.provider.client.ActivateApplicationPrincipal(returnedUid)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to activate application principal, got error: %s", err))
			return
		}
		data.Active = types.BoolValue(true)
	} else if !data.Active.IsNull() {
		data.Active = types.BoolValue(false)
	}

	tflog.Trace(ctx, "Created an application principal resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationPrincipalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data applicationPrincipalResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationPrincipal, err := r.provider.client.ReadApplicationPrincipal(data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Error(ctx, fmt.Sprintf("Application Principal not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application principal, got error: %s", err))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapApplicationPrincipalResponseToData(ctx, &data, applicationPrincipal)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationPrincipalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan applicationPrincipalResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldId := state.Id.ValueString()
	application, err := r.provider.client.GetApplication(plan.Application.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application to determine type, got error: %s", err))
		return
	}

	// Cert-unchanged fast path: only `active` (or other non-cert attrs) changed.
	// Skip rotation — POST with the same fingerprint returns errmsg.duplicate.principal.
	// No deactivate API exists; active=false is a write-only intent (atomic swap by activating another principal).
	certUnchanged := strings.TrimSpace(plan.Principal.ValueString()) == strings.TrimSpace(state.Principal.ValueString()) &&
		strings.TrimSpace(plan.PrivateKey.ValueString()) == strings.TrimSpace(state.PrivateKey.ValueString())
	if certUnchanged {
		tflog.Info(ctx, fmt.Sprintf("Update application principal: cert unchanged for %s, no rotation", oldId))
		active, err := r.resolveActivation(ctx, oldId, plan, state, application, false)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		plan.Active = active
		plan.Id = state.Id
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	// Cert rotation: create new principal, activate (if Connector), delete old.
	principalReq, err := createApplicationPrincipalRequestFromData(ctx, &plan, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating UPDATE request struct for application principal resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Update application principal: creating new principal to replace %s", oldId))
	newPrincipal, err := r.provider.client.CreateApplicationPrincipal(principalReq)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error during application principal update", fmt.Sprintf("Error message: %s %s", newPrincipal, err))
		return
	}
	trimmedResponse := strings.Trim(string(newPrincipal), "\"")
	newId := strings.ReplaceAll(trimmedResponse, fmt.Sprintf("%s/%s", r.provider.client.ApiURL, "application_principals/"), "")

	active, err := r.resolveActivation(ctx, newId, plan, state, application, true)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	plan.Active = active

	tflog.Info(ctx, fmt.Sprintf("Deleting old application principal %s", oldId))
	if err := r.provider.client.DeleteApplicationPrincipal(oldId); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete old application principal, got error: %s", err))
		return
	}

	plan.Id = types.StringValue(newId)
	tflog.Trace(ctx, "Updated application principal resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *applicationPrincipalResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data applicationPrincipalResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteApplicationPrincipal(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete application principal, got error: %s", err))
		return
	}
}

func (r *applicationPrincipalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// boolTrue reports whether v is known and true.
func boolTrue(v types.Bool) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueBool()
}

// inactiveForState returns the resting `active` value when no activation occurs:
// false if the prior state carried a value (preserve shape), null otherwise.
func inactiveForState(prior types.Bool) types.Bool {
	if !prior.IsNull() {
		return types.BoolValue(false)
	}
	return types.BoolNull()
}

// resolveActivation decides whether to activate the principal at `id` and applies it.
// Returns the final `active` value to persist in the state.
//
// autoActivate=true (rotation path): Connector apps activate by default unless the user explicitly set false.
// autoActivate=false (cert-unchanged fast path): only activate when the user explicitly sets true and the state isn't yet active.
func (r *applicationPrincipalResource) resolveActivation(ctx context.Context, id string, plan, state applicationPrincipalResourceData, app *webclient.ApplicationResponse, autoActivate bool) (types.Bool, error) {
	if app.ApplicationType != "Connector" {
		tflog.Info(ctx, fmt.Sprintf("Skipping activation of application principal %s (application type: %s)", id, app.ApplicationType))
		return inactiveForState(state.Active), nil
	}
	if autoActivate {
		return r.activateRotation(ctx, id, plan, state)
	}
	return r.activateInPlace(ctx, id, plan, state)
}

// activateRotation: cert rotation auto-activates Connector principals unless the user explicitly set active=false.
func (r *applicationPrincipalResource) activateRotation(ctx context.Context, id string, plan, state applicationPrincipalResourceData) (types.Bool, error) {
	if plan.Active.Equal(types.BoolValue(false)) {
		tflog.Info(ctx, fmt.Sprintf("Skipping activation of application principal %s (active=false)", id))
		return inactiveForState(state.Active), nil
	}
	return r.activate(ctx, id, plan, state)
}

// activateInPlace: cert-unchanged path activates only on explicit true when state isn't already active.
func (r *applicationPrincipalResource) activateInPlace(ctx context.Context, id string, plan, state applicationPrincipalResourceData) (types.Bool, error) {
	if !boolTrue(plan.Active) || boolTrue(state.Active) {
		tflog.Info(ctx, fmt.Sprintf("Skipping activation of application principal %s (active: %s)", id, plan.Active))
		if plan.Active.IsUnknown() {
			return state.Active, nil
		}
		return plan.Active, nil
	}
	return r.activate(ctx, id, plan, state)
}

// activate calls the activation API and returns the resulting `active` value.
// IsUnknown plan values fall back to prior state value — avoids upgrading unintentional rotations to active=true.
func (r *applicationPrincipalResource) activate(ctx context.Context, id string, plan, state applicationPrincipalResourceData) (types.Bool, error) {
	tflog.Info(ctx, fmt.Sprintf("Activating application principal %s", id))
	if err := r.provider.client.ActivateApplicationPrincipal(id); err != nil {
		return types.BoolNull(), fmt.Errorf("unable to activate application principal: %w", err)
	}
	if plan.Active.IsUnknown() {
		return state.Active, nil
	}
	return types.BoolValue(true), nil
}

func createApplicationPrincipalRequestFromData(ctx context.Context, data *applicationPrincipalResourceData, r *applicationPrincipalResource) ([1]webclient.ApplicationPrincipalRequest, error) {
	rawEnvironment, err := data.Environment.ToTerraformValue(ctx)
	if err != nil {
		return [1]webclient.ApplicationPrincipalRequest{}, err
	}
	var environment string
	err = rawEnvironment.As(&environment)
	if err != nil {
		return [1]webclient.ApplicationPrincipalRequest{}, err
	}
	environment = fmt.Sprintf("%s/%v", r.provider.client.ApiURL, environment)

	rawApplication, err := data.Application.ToTerraformValue(ctx)
	if err != nil {
		return [1]webclient.ApplicationPrincipalRequest{}, err
	}
	var application string
	err = rawApplication.As(&application)
	if err != nil {
		return [1]webclient.ApplicationPrincipalRequest{}, err
	}
	application = fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, application)

	var applicationPrincipalRequestArray [1]webclient.ApplicationPrincipalRequest
	applicationPrincipalRequestArray[0] =
		webclient.ApplicationPrincipalRequest{
			Principal:   strings.TrimSpace(data.Principal.ValueString()),
			Application: application,
			Environment: environment,
		}
	// optional fields
	if !data.Custom.IsNull() && data.Custom.ValueBool() {
		applicationPrincipalRequestArray[0].Custom = data.Custom.ValueBool()
	}
	if !data.PrivateKey.IsNull() {
		applicationPrincipalRequestArray[0].PrivateKey = strings.TrimSpace(data.PrivateKey.ValueString())
	}
	return applicationPrincipalRequestArray, err
}

func mapApplicationPrincipalResponseToData(_ context.Context, data *applicationPrincipalResourceData, applicationPrincipal *webclient.ApplicationPrincipalResponse) {
	data.Id = types.StringValue(applicationPrincipal.Uid)
	data.Environment = types.StringValue(applicationPrincipal.Embedded.Environment.Uid)
	data.Application = types.StringValue(applicationPrincipal.Embedded.Application.Uid)
	// active is intentionally not refreshed from the API. It is write-only intent:
	// Terraform sets it on Create/Update; Read() preserving the prior state value prevents
	// false drift when another principal is externally activated (atomic swap by the API
	// silently deactivates this one). Perpetual re-activation loops are avoided this way.
	// Branch on API type: only SSL deals with PEM certificate files.
	if applicationPrincipal.Type == "OAUTH" {
		data.Custom = types.BoolValue(true)
		data.Principal = types.StringValue(applicationPrincipal.Principal)
	} else {
		// SSL: applicationPem contains the full PEM certificate chain.
		// Preserve existing state value when only whitespace differs. The API returns
		// trimmed values, but the config (from file()) may include a trailing newline.
		apiPrincipal := applicationPrincipal.ApplicationPem
		if !data.Principal.IsNull() && !data.Principal.IsUnknown() &&
			strings.TrimSpace(data.Principal.ValueString()) == strings.TrimSpace(apiPrincipal) {
			// Keep existing state value — semantically equal
		} else {
			data.Principal = types.StringValue(apiPrincipal)
		}
	}
}
