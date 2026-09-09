package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &applicationDeploymentResource{}
	_ resource.ResourceWithImportState = &applicationDeploymentResource{}
)

// NewApplicationDeploymentResource creates a new application deployment resource
func NewApplicationDeploymentResource(provider AxualProvider) resource.Resource {
	return &applicationDeploymentResource{
		provider: provider,
	}
}

type applicationDeploymentResource struct {
	provider AxualProvider
}

// legacyAxualConnectTargetPrefix prefixes the deployment target id the API synthesizes for a
// Connector deployment that has none stored, standing for the legacy Axual Connect runtime
// (ConnectorDeploymentTargetResolver.AXUAL_CONNECT_ID_PREFIX). It is not a registered Kafka
// Connect cluster and is rejected when sent back as a `targetId`.
const legacyAxualConnectTargetPrefix = "axualconnect-"

// startAttempts is how often a START is retried before the apply is failed.
const startAttempts = 3

// stopWaitAttempts and stopWaitDelay bound how long an update or a delete waits for a stopped
// deployment to stop running before it PATCHes or DELETEs it.
const (
	stopWaitAttempts = 10
	stopWaitDelay    = 3 * time.Second
)

// startWaitAttempts and startWaitDelay bound how long a START waits for a deployment that is
// still stopping to become startable.
const (
	startWaitAttempts = 30
	startWaitDelay    = 2 * time.Second
)

type ApplicationDeploymentResourceData struct {
	Id                types.String `tfsdk:"id"`
	Application       types.String `tfsdk:"application"`
	Environment       types.String `tfsdk:"environment"`
	Configs           types.Map    `tfsdk:"configs"`
	Type              types.String `tfsdk:"type"`
	Definition        types.String `tfsdk:"definition"`
	DeploymentSize    types.String `tfsdk:"deployment_size"`
	RestartPolicy     types.String `tfsdk:"restart_policy"`
	TargetId          types.String `tfsdk:"target_id"`
	SqlScript         types.String `tfsdk:"sql_script"`
	GenerateTablesSql types.Bool   `tfsdk:"generate_tables_sql"`
}

func (r *applicationDeploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_deployment"
}
func (r *applicationDeploymentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An Application Deployment stores the configs for 'Connector' or 'Ksml' application type that is saved for an Application on an Environment.",

		Attributes: map[string]schema.Attribute{
			"application": schema.StringAttribute{
				MarkdownDescription: "A valid Uid of an existing application",
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
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of application deployment. This is automatically set based on the application's type.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					deploymentTypePlanModifier{},
				},
			},
			"configs": schema.MapAttribute{
				MarkdownDescription: "Connector config for Application Deployment. Required for Connector deployments. This field is Sensitive and will not be displayed in server log outputs when using Terraform commands. All available application plugin class names, plugin types and plugin configs are listed here in API- `GET: /api/connect_plugins?page=0&size=9999&sort=pluginClass` and in Axual Connect Docs: https://docs.axual.io/connect/Axual-Connect/connect-plugins-catalog/connect-plugins-catalog.html",
				Optional:            true,
				ElementType:         types.StringType,
				Sensitive:           true,
			},
			"definition": schema.StringAttribute{
				MarkdownDescription: "KSML definition for Application Deployment. Required for KSML deployments. This field is Sensitive and will not be displayed in server log outputs when using Terraform commands.",
				Optional:            true,
				Sensitive:           true,
			},
			"deployment_size": schema.StringAttribute{
				MarkdownDescription: "The t-shirt size (e.g. XS, S, M, L, XL) of the deployment. Optional for KSML and FLINK_SQL deployments; for FLINK_SQL it sizes the Flink TaskManager. If not specified, the Platform Manager will assign a default value.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"restart_policy": schema.StringAttribute{
				MarkdownDescription: "The restart policy for KSML applications. Valid values are 'on_exit' and 'never'. Required for KSML deployments.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("on_exit", "never"),
				},
			},
			"target_id": schema.StringAttribute{
				// Optional and Computed: the Platform Manager assigns a default target when none is
				// set, and UseStateForUnknown keeps an omitted `target_id` on that value.
				// A changed target replaces the deployment: FLINK_SQL rejects a target change
				// outright (AXPD-11759), and Connector deployments are replaced anyway.
				MarkdownDescription: "The id of the deployment target to deploy to. Required for FLINK_SQL deployments, where it must be the id of an `axual_flink_cluster` registered for the environment. For other deployment types the Platform Manager assigns a default target if not specified. Changing this value replaces the Application Deployment, as the deployment target can only be set when the deployment is created. Available targets can be listed via `GET /applications/{applicationId}/deployment-targets`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sql_script": schema.StringAttribute{
				MarkdownDescription: "The transformation SQL for a FLINK_SQL deployment (an `INSERT INTO ... SELECT ...` statement, without credentials or fully-qualified topic names). Required for FLINK_SQL deployments. This field is Sensitive and will not be displayed in server log outputs when using Terraform commands.",
				Optional:            true,
				Sensitive:           true,
			},
			"generate_tables_sql": schema.BoolAttribute{
				// Computed as well as Optional: the API defaults an absent value to `false` and
				// always stores the config key, so a null plan value would not match what the
				// deployment reports back.
				MarkdownDescription: "For FLINK_SQL deployments, whether to auto-generate the `CREATE TABLE` statements for the topics referenced by `sql_script`. Optional for FLINK_SQL deployments; defaults to `false`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// deploymentTypePlanModifier keeps KSML and FLINK_SQL deployments on an in-place update and replaces
// Connector deployments, whose update the Platform Manager currently rejects with "No Kafka Connect
// cluster matching the deployment `targetId` axualconnect-<env>". Leaving the planned type unknown is
// what makes Terraform replace it, so the state value is only copied for the types updated in place
// (UseStateForUnknown plus RequiresReplaceIf cannot express that: making the values equal is exactly
// what stops RequiresReplaceIf from firing).
type deploymentTypePlanModifier struct{}

func (m deploymentTypePlanModifier) Description(_ context.Context) string {
	return "Connector Application Deployments are replaced rather than updated in place; KSML and FLINK_SQL deployments are updated in place."
}

func (m deploymentTypePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m deploymentTypePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Nothing to decide when the deployment is being created or destroyed.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	if isConnector(req.StateValue.ValueString()) {
		resp.RequiresReplace = true
		return
	}

	if req.PlanValue.IsUnknown() {
		resp.PlanValue = req.StateValue
	}
}

func (r *applicationDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ApplicationDeploymentResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	// Fetch application to determine its type
	application, err := r.provider.client.GetApplication(data.Application.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error fetching application", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	// Set the type based on the application's type
	data.Type = types.StringValue(application.ApplicationType)

	applicationURL := fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, data.Application.ValueString())
	environmentURL := fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, data.Environment.ValueString())

	// A FLINK_SQL job runs on the application's SASL/SCRAM Kafka credential, which the platform
	// injects into the generated SQL at deploy time. An mTLS Application Principal cannot be used,
	// so the principal/credential count check below does not apply to FLINK_SQL.
	var applicationPrincipalsResponse *webclient.ApplicationPrincipalFindByApplicationAndEnvironmentResponse
	if isFlinkSQL(data.Type.ValueString()) {
		// The credential search endpoint takes plain ids (`applicationId=`/`environmentId=`), unlike the
		// principal search below, which takes resource URLs (`application=`/`environment=`).
		credentials, err := r.provider.client.FindApplicationCredentialByApplicationAndEnvironment(data.Application.ValueString(), data.Environment.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error querying for Application Credential for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
			return
		}
		if countScramCredentials(credentials) == 0 {
			resp.Diagnostics.AddError(
				"Missing SASL/SCRAM Application Credential",
				"A FLINK_SQL Application Deployment requires an axual_application_credential supporting SCRAM_SHA_512 for this application and environment. "+
					"An axual_application_principal cannot be used: the platform rejects the deployment with "+
					"\"a Flink SQL application requires SASL/SCRAM Kafka credentials\".",
			)
			return
		}
	} else {
		// we count if there is at least one authentication defined for these application and environment
		authenticationCount := 0
		// We check if Application Principal exists for this environment and application
		applicationPrincipalsResponse, err = r.provider.client.FindApplicationPrincipalByApplicationAndEnvironment(applicationURL, environmentURL)
		if err != nil {
			resp.Diagnostics.AddError("Error querying for Application Principal for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
			return
		}
		authenticationCount += len(applicationPrincipalsResponse.Embedded.ApplicationPrincipalResponses)
		if isKSML(data.Type.ValueString()) {
			// For KSML applications, we check if Application Credential exists for this environment and application
			applicationCredentialsResponse, err := r.provider.client.FindApplicationCredentialByApplicationAndEnvironment(data.Application.ValueString(), data.Environment.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Error querying for Application Credential for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
				return
			}
			authenticationCount += len(applicationCredentialsResponse)
		}

		if authenticationCount == 0 {
			resp.Diagnostics.AddError("Error from Terraform Provider validation", "Please first create an Application Principal or Application Credential for this application and environment")
			return
		}
	}

	// Connector deployments require at least one active Application Principal: the deployment START
	// will fail at the API otherwise. The API does NOT auto-activate, so we surface a clear error here
	// instead of letting deployment creation fail with a generic platform error.
	if isConnector(data.Type.ValueString()) {
		if countActivePrincipals(applicationPrincipalsResponse) == 0 {
			resp.Diagnostics.AddError(
				"No active Application Principal",
				fmt.Sprintf(
					"No active Application Principal found for application=%s environment=%s. "+
						"Activate one by setting `active = true` on the axual_application_principal resource for this application and environment, "+
						"or activate it manually via the Axual Self Service UI. "+
						"For cross-repo setups (where the principal is managed in a different Terraform configuration), "+
						"ensure activation has been applied before creating this deployment.",
					data.Application.ValueString(), data.Environment.ValueString(),
				),
			)
			return
		}
	}

	// Registering a Connector deployment target and adding the configs later is a supported flow
	// (ConnectorDeploymentManager.validateConfig returns early on empty configs), so this only
	// warns: the deployment is created but offers no START action until `configs` are set.
	if data.Type.ValueString() == "Connector" && countConfigs(data.Configs) == 0 {
		resp.Diagnostics.AddWarning(
			"Connector Application Deployment has no configs",
			"The deployment is created but cannot be started until `configs` are set.",
		)
	}
	// A FLINK_SQL deployment without SQL is rejected here rather than created: the API accepts it
	// and then never offers a START action, leaving a created-but-unstartable deployment behind.
	if isFlinkSQL(data.Type.ValueString()) && data.SqlScript.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing Flink SQL script",
			"A FLINK_SQL Application Deployment requires a `sql_script`.",
		)
		return
	}

	// The target is not optional for FLINK_SQL: it names the axual_flink_cluster to deploy to.
	if isFlinkSQL(data.Type.ValueString()) && data.TargetId.IsNull() {
		resp.Diagnostics.AddError(
			"Missing deployment target",
			"A FLINK_SQL Application Deployment requires a `target_id` referring to an `axual_flink_cluster` registered for this environment.",
		)
		return
	}

	// We check if Approved Application Access Grant exists for this environment and application
	accessGrantRequest := webclient.ApplicationAccessGrantAttributes{
		ApplicationId: data.Application.ValueString(),
		EnvironmentId: data.Environment.ValueString(),
		Statuses:      "APPROVED",
	}
	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrantsByAttributes(accessGrantRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error querying for Application Access Grant for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	// We do not allow creating Application Deployment if there is no Approved Application Access Grant, because we can't start the connector without it
	if len(applicationAccessGrant.Embedded.ApplicationAccessGrantResponses) == 0 {
		resp.Diagnostics.AddError("Error from Terraform Provider validation", "Please first create and approve Application Access Grant for this application and environment")
		return
	}

	// We create Application Deployment
	ApplicationDeploymentRequest, err := createApplicationDeploymentRequestFromData(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request struct for application deployment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	createResponse, err := r.provider.client.CreateApplicationDeployment(ApplicationDeploymentRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for application deployment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	// Build the Flink SQL config PATCH from the plan-supplied values BEFORE mapping the (target-only,
	// configs-less) create response into data - that mapping nulls out SqlScript/DeploymentSize/
	// GenerateTablesSql to reflect what the API actually has, which would leave nothing to PATCH.
	var flinkConfigs map[string]string
	if isFlinkSQL(data.Type.ValueString()) {
		flinkConfigs, err = createConfigsForDeploymentType(&data)
		if err != nil {
			resp.Diagnostics.AddError("Error creating Flink configs for application deployment resource", fmt.Sprintf("Error message: %s", err.Error()))
			return
		}
	}

	// The POST response has no body: the Uid of the created deployment comes from the Location
	// header of the response, which the API returns for every application type.
	if createResponse.Uid == "" {
		resp.Diagnostics.AddError(
			"Error reading the created application deployment",
			"The API did not return a Location header with the Uid of the created Application Deployment.",
		)
		return
	}
	createdDeployment, err := r.provider.client.GetApplicationDeployment(createResponse.Uid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading the created application deployment", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	mapApplicationDeploymentByIdResponseToData(ctx, &data, createdDeployment)

	// Save state as soon as the deployment's Id is known, BEFORE the FLINK_SQL config PATCH or
	// START - a failure past this point still leaves the deployment trackable, so a retried apply
	// goes through Update() (which knows to PATCH FLINK_SQL configs) instead of Create() orphaning
	// an untracked deployment the API won't let a fresh Create() replace.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A FLINK_SQL deployment is created with a deployment target only (see
	// createApplicationDeploymentRequestFromData); its SQL config is set here via a follow-up PATCH.
	if isFlinkSQL(data.Type.ValueString()) {
		flinkUpdate := webclient.ApplicationDeploymentUpdateRequest{Configs: flinkConfigs}
		_, err = r.provider.client.UpdateApplicationDeployment(data.Id.ValueString(), flinkUpdate)
		if err != nil {
			// A warning, not an error: the deployment is already in state, and failing Create with
			// state set taints the resource, turning every retry into a destroy-and-create that
			// fails the same way instead of the Update() that applies the config.
			resp.Diagnostics.AddWarning(
				"Deployment created but the Flink SQL config was not applied",
				fmt.Sprintf("The deployment was created and saved to state, but its SQL config is not set: %s. "+
					"Run 'terraform apply' again to apply it.", err),
			)
			return
		}
		updatedDeployment, err := r.provider.client.GetApplicationDeployment(data.Id.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading application deployment after setting Flink SQL config", fmt.Sprintf("Error message: %s", err.Error()))
			return
		}
		mapApplicationDeploymentByIdResponseToData(ctx, &data, updatedDeployment)
		diags = resp.State.Set(ctx, &data)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	tflog.Info(ctx, "State saved immediately after deployment creation")

	// A START failure is reported as a warning, not an error: the deployment is already saved to
	// state above, and failing Create with state set would taint the resource, turning the next
	// apply into a destroy-and-create instead of the Update() that retries the START.
	if err := r.startApplicationDeployment(ctx, data.Id.ValueString(), data.Type.ValueString(), 10*time.Second); err != nil {
		resp.Diagnostics.AddWarning(
			"Deployment created but START failed",
			fmt.Sprintf("The deployment was created and saved to state, but it is not running: %s. "+
				"Run 'terraform apply' again to retry starting the deployment.", err),
		)
		return
	}

	tflog.Info(ctx, "Successfully created and started Application Deployment")
}

func (r *applicationDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ApplicationDeploymentResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationWithUrl := fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, data.Application.ValueString())
	environmentWithUrl := fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, data.Environment.ValueString())
	ApplicationDeploymentFindByApplicationAndEnvironmentResponse, err := r.provider.client.FindApplicationDeploymentByApplicationAndEnvironment(applicationWithUrl, environmentWithUrl)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to find Application Deployment with ID: %s, got error: %s", data.Id.ValueString(), err))
		} else {
			resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to read Application Deployment, got error: %s", err))
		}
		return
	}
	err = mapApplicationDeploymentByApplicationAndEnvironmentResponseToData(ctx, &data, ApplicationDeploymentFindByApplicationAndEnvironmentResponse)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to map Application Deployment, got error: %s", err))
		return
	}
	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData ApplicationDeploymentResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)

	var stateData ApplicationDeploymentResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Resizing a Flink job is the API's cheap path: a `flink_task_size`-only patch reaches
	// `saveTaskSizeOnly`, which upserts that one config key without calling Ververica and without
	// checking the job's live state, so the deployment neither has to be stopped nor restarted.
	if isFlinkTaskSizeOnlyChange(&stateData, &planData) {
		sizeOnlyRequest := webclient.ApplicationDeploymentUpdateRequest{
			Configs: map[string]string{"flink_task_size": planData.DeploymentSize.ValueString()},
		}
		if _, err := r.provider.client.UpdateApplicationDeployment(planData.Id.ValueString(), sizeOnlyRequest); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to resize Application Deployment, got error: %s", err))
			return
		}
		tflog.Info(ctx, fmt.Sprintf("Resized Application Deployment %s to %s without redeploying", planData.Id.ValueString(), planData.DeploymentSize.ValueString()))

		resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
		return
	}

	// A running deployment cannot be updated: `beforeUpdateImpl` rejects a save while the desired
	// state is RUNNING, and `saveFlinkSql` rejects one while a Flink job actually is. Stop it and
	// wait, exactly as Delete does.
	if err := r.stopDeploymentAndWait(ctx, planData.Id.ValueString(), planData.Type.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	ApplicationDeploymentUpdateRequest, err := createApplicationUpdateDeploymentRequestFromData(ctx, &planData)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request struct for application deployment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	_, err = r.provider.client.UpdateApplicationDeployment(planData.Id.ValueString(), ApplicationDeploymentUpdateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Application Deployment, got error: %s", err))
		return
	}
	tflog.Info(ctx, "Successfully updated Application Deployment")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)

	// Unlike Create, Update fails the apply when the deployment cannot be started: the resource
	// is already tracked, so an error here does not taint it and the next apply retries Update.
	if err := r.startApplicationDeployment(ctx, planData.Id.ValueString(), planData.Type.ValueString(), 5*time.Second); err != nil {
		resp.Diagnostics.AddError(
			"Application Deployment START failed",
			fmt.Sprintf("The deployment was updated but it is not running: %s. "+
				"Fix the cause of the failure and run 'terraform apply' again.", err),
		)
		return
	}

	tflog.Info(ctx, "Successfully started Application Deployment after update")
}

func (r *applicationDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ApplicationDeploymentResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.stopDeploymentAndWait(ctx, data.Id.ValueString(), data.Type.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// No `delete` rel pre-check: the rel follows the deployment's desired state, which STOP changes
	// at once, so it is offered while a Flink job is still draining and never offered for a failed
	// FLINK_SQL job that was never stopped - a DELETE the API does accept (AXPD-11714). The API's
	// own error is the better one to report.
	if err := r.provider.client.DeleteApplicationDeployment(data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Application Deployment, got error: %s", err))
		return
	}
}

func mapApplicationDeploymentByApplicationAndEnvironmentResponseToData(
	ctx context.Context,
	data *ApplicationDeploymentResourceData,
	applicationDeploymentResponse *webclient.ApplicationDeploymentFindByApplicationAndEnvironmentResponse) error {
	if len(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses) == 0 {
		return fmt.Errorf("error processing mapping application deployment response, no application deployment found for the application and environment")
	} else if len(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses) > 1 {
		return fmt.Errorf("error processing mapping application deployment response, multiple application deployments found for the application and environment")
	} else {
		data.Id = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Uid)
		data.Environment = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Environment.Uid)
		data.Application = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Application.Uid)
		mapTargetIdToData(data, applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].TargetId)
		mapResponseConfigsToData(ctx, data, applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Application.ApplicationType, applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Configs)
		return nil
	}
}

func mapApplicationDeploymentByIdResponseToData(ctx context.Context, data *ApplicationDeploymentResourceData, applicationDeploymentResponse *webclient.ApplicationDeploymentResponse) {
	data.Id = types.StringValue(applicationDeploymentResponse.Uid)
	data.Environment = types.StringValue(applicationDeploymentResponse.Embedded.Environment.Uid)
	data.Application = types.StringValue(applicationDeploymentResponse.Embedded.Application.Uid)

	mapTargetIdToData(data, applicationDeploymentResponse.TargetId)
	mapResponseConfigsToData(ctx, data, applicationDeploymentResponse.Embedded.Application.ApplicationType, applicationDeploymentResponse.Configs)
}

func mapTargetIdToData(data *ApplicationDeploymentResourceData, targetId string) {
	if targetId == "" {
		data.TargetId = types.StringNull()
	} else {
		data.TargetId = types.StringValue(targetId)
	}
}

func createApplicationDeploymentRequestFromData(ctx context.Context, data *ApplicationDeploymentResourceData) (webclient.ApplicationDeploymentCreateRequest, error) {
	// A FLINK_SQL deployment is created with a deployment target only: `validateEmptyConfigs`
	// refuses a non-empty `configs` on POST, so the SQL is set by the follow-up PATCH in Create().
	var configs map[string]string
	if !isFlinkSQL(data.Type.ValueString()) {
		var err error
		configs, err = createConfigsForDeploymentType(data)
		if err != nil {
			return webclient.ApplicationDeploymentCreateRequest{}, err
		}
	}

	ApplicationDeploymentRequest := webclient.ApplicationDeploymentCreateRequest{
		Application: data.Application.ValueString(),
		Environment: data.Environment.ValueString(),
		Configs:     configs,
	}
	// A Connector deployment with no stored target reads back with a synthesized
	// `axualconnect-<instance>` id standing for legacy Axual Connect. That id is not a registered
	// Kafka Connect cluster, so sending it back on a create fails with "No Kafka Connect cluster
	// matching the deployment `targetId`" - leave it out and let the platform resolve the target,
	// which is what an unset target means.
	if !data.TargetId.IsNull() && !data.TargetId.IsUnknown() &&
		!strings.HasPrefix(data.TargetId.ValueString(), legacyAxualConnectTargetPrefix) {
		ApplicationDeploymentRequest.TargetId = data.TargetId.ValueString()
	}

	tflog.Info(ctx, fmt.Sprintf("Application request completed: %q", ApplicationDeploymentRequest))
	return ApplicationDeploymentRequest, nil
}

// createApplicationUpdateDeploymentRequestFromData builds the update request. The target is absent
// because `target_id` requires replacement in the schema, so a change never reaches this path.
func createApplicationUpdateDeploymentRequestFromData(ctx context.Context, data *ApplicationDeploymentResourceData) (webclient.ApplicationDeploymentUpdateRequest, error) {
	configs, err := createConfigsForDeploymentType(data)

	if err != nil {
		return webclient.ApplicationDeploymentUpdateRequest{}, err
	}

	ApplicationDeploymentUpdateRequest := webclient.ApplicationDeploymentUpdateRequest{
		Configs: configs,
	}

	tflog.Info(ctx, fmt.Sprintf("Application update request completed: %q", ApplicationDeploymentUpdateRequest))
	return ApplicationDeploymentUpdateRequest, nil
}

func (r *applicationDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	applicationDeployment, err := r.provider.client.GetApplicationDeployment(req.ID)

	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			resp.Diagnostics.AddError(
				"Application Deployment Not Found",
				fmt.Sprintf("Application Deployment with ID: %s not found.", req.ID),
			)
		} else {
			resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to find Application Deployment, got error: %s", err))
		}
		return
	}

	if applicationDeployment.State != "Running" && applicationDeployment.State != "Started" {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to import an Application Deployment with status: %s. In order to import an Application deployment, it should be in RUNNING state", applicationDeployment.State))
		return

	}

	// Map the response to Terraform state
	var data ApplicationDeploymentResourceData
	mapApplicationDeploymentByIdResponseToData(ctx, &data, applicationDeployment)

	// Validate that the mapped data is complete
	if data.Id.IsNull() || data.Id.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Invalid Application Deployment Data",
			"The imported Application Deployment does not have a valid ID",
		)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully imported Application Deployment with ID: %s", data.Id.ValueString()))

	// Set the state with the imported data
	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state during import")
		return
	}

	tflog.Info(ctx, "Application Deployment import completed successfully")

}

func createConfigsForDeploymentType(data *ApplicationDeploymentResourceData) (map[string]string, error) {
	configs := make(map[string]string)

	// Determine the deployment type, default to "Connector" if not specified
	deploymentType := "Connector"
	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		deploymentType = data.Type.ValueString()
	}

	if isKSML(deploymentType) {
		// For KSML deployments, add KSML-specific configs
		if !data.Definition.IsNull() && !data.Definition.IsUnknown() {
			configs["ksml_definition"] = data.Definition.ValueString()
		}
		if !data.DeploymentSize.IsNull() && !data.DeploymentSize.IsUnknown() {
			configs["ksml_deployment_size"] = data.DeploymentSize.ValueString()
		}
		if !data.RestartPolicy.IsNull() && !data.RestartPolicy.IsUnknown() {
			configs["ksml_restart_policy"] = data.RestartPolicy.ValueString()
		}
	} else if isFlinkSQL(deploymentType) {
		// For FLINK_SQL deployments, add Flink-specific configs
		if !data.SqlScript.IsNull() && !data.SqlScript.IsUnknown() {
			configs["flink_sql"] = data.SqlScript.ValueString()
		}
		if !data.GenerateTablesSql.IsNull() && !data.GenerateTablesSql.IsUnknown() {
			configs["flink_generate_tables_sql"] = fmt.Sprintf("%t", data.GenerateTablesSql.ValueBool())
		}
		// The size attribute is shared with KSML, only the config key differs per application type.
		if !data.DeploymentSize.IsNull() && !data.DeploymentSize.IsUnknown() {
			configs["flink_task_size"] = data.DeploymentSize.ValueString()
		}
	} else {
		// For Connector deployments, use the configs map
		for key, value := range data.Configs.Elements() {
			strValue, ok := value.(types.String)
			if !ok {
				return configs, fmt.Errorf("type assertion to types.String failed for key: %s", key)
			}
			configs[key] = strValue.ValueString()
		}
	}
	return configs, nil
}

func mapResponseConfigsToData(ctx context.Context, data *ApplicationDeploymentResourceData, deploymentType string, responseConfigs []webclient.Config) {
	// Initialize the map for configs
	configs := make(map[string]attr.Value)

	var ksmlDefinition, ksmlDeploymentSize, ksmlRestartPolicy string
	var flinkSql, flinkTaskSize string
	var flinkGenerateTablesSql *bool

	// We iterate through the Configs and extract KSML- and Flink-specific ones
	for _, config := range responseConfigs {
		switch config.ConfigKey {
		case "ksml_definition":
			ksmlDefinition = config.ConfigValue
		case "ksml_deployment_size":
			ksmlDeploymentSize = config.ConfigValue
		case "ksml_restart_policy":
			ksmlRestartPolicy = config.ConfigValue
		case "flink_sql":
			flinkSql = config.ConfigValue
		case "flink_task_size":
			flinkTaskSize = config.ConfigValue
		case "flink_generate_tables_sql":
			b := config.ConfigValue == "true"
			flinkGenerateTablesSql = &b
		case "flink_deployment_id":
			// internal Ververica-side link, not exposed to Terraform
		default:
			configs[config.ConfigKey] = types.StringValue(config.ConfigValue)
		}
	}

	if isKSML(deploymentType) {
		data.Type = types.StringValue("Ksml")
		data.Definition = types.StringValue(ksmlDefinition)
		if ksmlDeploymentSize != "" {
			data.DeploymentSize = types.StringValue(ksmlDeploymentSize)
		} else {
			data.DeploymentSize = types.StringNull()
		}
		if ksmlRestartPolicy != "" {
			data.RestartPolicy = types.StringValue(ksmlRestartPolicy)
		} else {
			data.RestartPolicy = types.StringNull()
		}
		// For KSML, the configs map should be empty or null
		data.Configs = types.MapNull(types.StringType)
		data.SqlScript = types.StringNull()
		data.GenerateTablesSql = types.BoolNull()
	} else if isFlinkSQL(deploymentType) {
		data.Type = types.StringValue("FLINK_SQL")
		if flinkSql != "" {
			data.SqlScript = types.StringValue(flinkSql)
		} else {
			data.SqlScript = types.StringNull()
		}
		if flinkTaskSize != "" {
			data.DeploymentSize = types.StringValue(flinkTaskSize)
		} else {
			data.DeploymentSize = types.StringNull()
		}
		if flinkGenerateTablesSql != nil {
			data.GenerateTablesSql = types.BoolValue(*flinkGenerateTablesSql)
		} else {
			data.GenerateTablesSql = types.BoolNull()
		}
		// For FLINK_SQL, the configs map and KSML-specific fields should be null
		data.Configs = types.MapNull(types.StringType)
		data.Definition = types.StringNull()
		data.RestartPolicy = types.StringNull()
	} else {
		if data.Type.IsNull() || data.Type.IsUnknown() {
			data.Type = types.StringValue("Connector")
		}
		if len(configs) == 0 {
			// `configs` is Optional and not Computed, so a deployment the API reports no configs
			// for has to map back to null: an empty map would not match the null config value and
			// Terraform would reject the apply with "inconsistent values for sensitive attribute".
			data.Configs = types.MapNull(types.StringType)
		} else {
			mapValue, diags := types.MapValue(types.StringType, configs)
			if diags.HasError() {
				tflog.Error(ctx, "Error creating configs map when mapping application deployment response")
			}
			data.Configs = mapValue
		}
		// For Connector deployments, KSML- and Flink-specific fields should be null
		data.Definition = types.StringNull()
		data.DeploymentSize = types.StringNull()
		data.RestartPolicy = types.StringNull()
		data.SqlScript = types.StringNull()
		data.GenerateTablesSql = types.BoolNull()
	}
}

func isKSML(deploymentType string) bool {
	return deploymentType == "Ksml"
}

func isFlinkSQL(deploymentType string) bool {
	return deploymentType == webclient.FlinkSQLApplicationType
}

// isConnector reports whether the deployment is a Connector deployment: anything that is neither
// a KSML nor a Flink SQL deployment.
func isConnector(deploymentType string) bool {
	return !isKSML(deploymentType) && !isFlinkSQL(deploymentType)
}

// countConfigs counts the entries in a `configs` map, treating null and unknown as empty.
func countConfigs(configs types.Map) int {
	if configs.IsNull() || configs.IsUnknown() {
		return 0
	}
	return len(configs.Elements())
}

// countActivePrincipals counts principals whose API-returned active flag is true.
// A nil Active pointer means the field was absent from the response (treated as inactive).
func countActivePrincipals(response *webclient.ApplicationPrincipalFindByApplicationAndEnvironmentResponse) int {
	count := 0
	for _, p := range response.Embedded.ApplicationPrincipalResponses {
		if p.Active != nil && *p.Active {
			count++
		}
	}
	return count
}

// countScramCredentials counts the Application Credentials that support SCRAM-SHA-512, the only
// mechanism a Flink SQL job can authenticate with (FlinkKafkaConnectionOptionsProvider).
func countScramCredentials(credentials []webclient.ApplicationCredentialFindByApplicationAndEnvironmentResponse) int {
	count := 0
	for _, credential := range credentials {
		for _, authType := range credential.Types {
			if authType.Type == "SCRAM_SHA_512" {
				count++
				break
			}
		}
	}
	return count
}

// isFlinkTaskSizeOnlyChange reports whether the only difference between state and plan is a
// FLINK_SQL deployment's size. Such a change is sent as a `flink_task_size`-only patch, which the
// API applies through `saveTaskSizeOnly` instead of pushing the SQL to Ververica again.
func isFlinkTaskSizeOnlyChange(state *ApplicationDeploymentResourceData, plan *ApplicationDeploymentResourceData) bool {
	if !isFlinkSQL(plan.Type.ValueString()) {
		return false
	}
	// An unknown value is not comparable, so it cannot be ruled out as a change.
	if plan.DeploymentSize.IsUnknown() || plan.SqlScript.IsUnknown() || plan.GenerateTablesSql.IsUnknown() {
		return false
	}
	if plan.DeploymentSize.Equal(state.DeploymentSize) {
		return false
	}
	return plan.SqlScript.Equal(state.SqlScript) && plan.GenerateTablesSql.Equal(state.GenerateTablesSql)
}

// stopDeploymentAndWait stops the deployment when the API still offers a `stop` action, then waits
// for the per-type status to report a terminal state - the rels cannot answer that, because both
// `stop` and `delete` are already gone or offered while a Flink job is still draining.
// Running out of attempts is not an error: the caller lets the API decide, as Delete always did.
func (r *applicationDeploymentResource) stopDeploymentAndWait(ctx context.Context, id string, deploymentType string) error {
	status, err := r.provider.client.GetApplicationDeploymentStatus(id)
	if err != nil {
		return fmt.Errorf("unable to get Application Deployment status, got error: %s", err)
	}
	if !shouldStopDeployment(status) {
		return nil
	}

	stopRequest := webclient.ApplicationDeploymentOperationRequest{Action: "STOP"}
	if err := r.provider.client.OperateApplicationDeployment(id, "STOP", stopRequest); err != nil {
		return fmt.Errorf("unable to stop Application Deployment, got error: %s", err)
	}

	tflog.Info(ctx, fmt.Sprintf("Stopped Application Deployment %s, waiting for it to stop running", id))
	for i := 0; i < stopWaitAttempts; i++ {
		time.Sleep(stopWaitDelay)
		status, err := r.provider.client.GetApplicationDeploymentStatus(id)
		if err != nil {
			return fmt.Errorf("unable to get Application Deployment status while waiting for it to stop, got error: %s", err)
		}
		if isDeploymentStopped(deploymentType, status) {
			tflog.Info(ctx, fmt.Sprintf("Application Deployment %s stopped (%s)", id, describeDeploymentStatus(status)))
			return nil
		}
	}
	tflog.Warn(ctx, fmt.Sprintf("Application Deployment %s did not report a stopped state within %s, continuing", id, time.Duration(stopWaitAttempts)*stopWaitDelay))
	return nil
}

// startApplicationDeployment starts or resumes the deployment and returns nil once it is running or
// once the action was accepted; the caller decides whether the error fails the apply. A stopped
// FLINK_SQL deployment offers `resume`, not `start`, so the rel decides which action is sent.
func (r *applicationDeploymentResource) startApplicationDeployment(ctx context.Context, id string, deploymentType string, retryDelay time.Duration) error {
	status, err := r.provider.client.GetApplicationDeploymentStatus(id)
	if err != nil {
		return fmt.Errorf("unable to get Application Deployment status before starting it: %s", err)
	}

	// Wait for the deployment to become startable. Update() stops a running deployment before
	// patching it, and the stop is asynchronous: the deployment reports `Stopping` for a while,
	// offering neither a start action (it is not stopped yet) nor a running status, and only
	// advertises one once it has come to a halt.
	action := startActionFor(status)
	for i := 0; i < startWaitAttempts && action == ""; i++ {
		// A deployment that is already running needs no start: a previous one succeeded, possibly
		// one whose response timed out on an earlier apply.
		if isDeploymentRunning(deploymentType, status) {
			tflog.Info(ctx, "Application Deployment is already running, skipping START")
			return nil
		}
		time.Sleep(startWaitDelay)
		if status, err = r.provider.client.GetApplicationDeploymentStatus(id); err != nil {
			return fmt.Errorf("unable to get Application Deployment status while waiting for it to become startable: %s", err)
		}
		action = startActionFor(status)
	}

	if action == "" {
		if isDeploymentRunning(deploymentType, status) {
			tflog.Info(ctx, "Application Deployment is already running, skipping START")
			return nil
		}
		return fmt.Errorf("the API does not offer a START or RESUME action for this deployment (%s)", describeDeploymentStatus(status))
	}

	var applicationStartRequest = webclient.ApplicationDeploymentOperationRequest{
		Action: action,
	}
	err = Retry(startAttempts, retryDelay, func() error {
		return r.provider.client.OperateApplicationDeployment(id, action, applicationStartRequest)
	})
	if err == nil {
		return nil
	}

	// The START may still have gone through on a request whose response was lost, so a deployment
	// that reports itself running counts as success rather than a failure.
	if currentStatus, statusErr := r.provider.client.GetApplicationDeploymentStatus(id); statusErr == nil && isDeploymentRunning(deploymentType, currentStatus) {
		tflog.Info(ctx, "START reported an error but the Application Deployment is running")
		return nil
	}

	return fmt.Errorf("could not start the deployment after %d attempts: %s", startAttempts, err)
}

// startActionFor is the action that brings the deployment up, or an empty string when the API
// offers neither. A stopped FLINK_SQL deployment resumes from its last checkpoint and is only
// startable again after a reset, so `resume` is what it advertises; every other type advertises
// `start`. START is preferred when both are offered.
func startActionFor(status *webclient.ApplicationDeploymentStatusResponse) string {
	switch {
	case status.Links.Has(webclient.RelStart):
		return "START"
	case status.Links.Has(webclient.RelResume):
		return "RESUME"
	}
	return ""
}

// describeDeploymentStatus renders the per-type status the API reported plus the actions it
// advertises, so a failure to start says what state the deployment was actually in.
func describeDeploymentStatus(status *webclient.ApplicationDeploymentStatusResponse) string {
	var reported []string
	if status.ConnectorState.State != "" {
		reported = append(reported, fmt.Sprintf("connectorState=%s", status.ConnectorState.State))
	}
	if status.KsmlStatus.Status != "" {
		reported = append(reported, fmt.Sprintf("ksmlStatus=%s", status.KsmlStatus.Status))
	}
	if status.FlinkStatus.Status != "" {
		reported = append(reported, fmt.Sprintf("flinkStatus=%s", status.FlinkStatus.Status))
	}

	actions := make([]string, 0, len(status.Links))
	for rel := range status.Links {
		actions = append(actions, rel)
	}
	sort.Strings(actions)

	return fmt.Sprintf("reported status: %s; available actions: %s",
		strings.Join(reported, ", "), strings.Join(actions, ", "))
}

// isDeploymentStopped reports whether the deployment has reached a state it can be deleted from.
// Unlike the stop-before-update/delete decision, this cannot be read from the `_links`: the rels
// stop advertising STOP - and start advertising DELETE - while a Flink job is still shutting down,
// which is exactly the window in which Ververica rejects the DELETE.
func isDeploymentStopped(deploymentType string, status *webclient.ApplicationDeploymentStatusResponse) bool {
	if isFlinkSQL(deploymentType) {
		return status.FlinkStatus.Status == "Undeployed" || status.FlinkStatus.Status == "Failed"
	}
	if isKSML(deploymentType) {
		return status.KsmlStatus.Status == "Undeployed"
	}
	// A Connector that never ran reports `Undefined` rather than `Stopped`.
	return status.ConnectorState.State == "Stopped" || status.ConnectorState.State == "Undefined"
}

// isDeploymentRunning reports whether the deployment reports itself as running. Like
// isDeploymentStopped this reads the per-type status: the `_links` cannot answer it, because a
// deployment that is still shutting down offers no `start` either.
func isDeploymentRunning(deploymentType string, status *webclient.ApplicationDeploymentStatusResponse) bool {
	if isFlinkSQL(deploymentType) {
		return status.FlinkStatus.Status == "Running"
	}
	if isKSML(deploymentType) {
		return status.KsmlStatus.Status == "Running"
	}
	return status.ConnectorState.State == "Running"
}

// shouldStopDeployment reports whether the deployment has to be stopped before it can be
// updated or deleted. The status endpoint advertises a `stop` link exactly when STOP is a valid
// action for the deployment's current state, for every application type - so the provider does
// not have to mirror the per-type state machines (Connector states, KSML statuses, Flink job
// states), nor be released again when the platform adds a state.
func shouldStopDeployment(status *webclient.ApplicationDeploymentStatusResponse) bool {
	return status.Links.Has(webclient.RelStop)
}
