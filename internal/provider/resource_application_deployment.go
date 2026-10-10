package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	_ resource.ResourceWithModifyPlan  = &applicationDeploymentResource{}
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
// Connector deployment that has none stored (AXPD-12050), standing for the legacy Axual Connect
// runtime (ConnectorDeploymentTargetResolver.AXUAL_CONNECT_ID_PREFIX). It is not a registered Kafka
// Connect cluster and is rejected when sent back as a `targetId`, so it is left out of the create.
const legacyAxualConnectTargetPrefix = "axualconnect-"

// startAttempts is how often a START is retried before the apply is failed.
const startAttempts = 3

// stopAttempts and stopRetryDelay retry a failed STOP. Variables so the unit tests can shorten them.
var (
	stopAttempts   = 3
	stopRetryDelay = 5 * time.Second
	// conflictAttempts and conflictDelay retry a delete that hit a concurrent change of the same row.
	conflictAttempts = 3
	conflictDelay    = 2 * time.Second
)

// stopWaitAttempts and stopWaitDelay bound how long an update or a delete waits for a stopped
// deployment to stop running before it PATCHes or DELETEs it. Variables so the unit tests can
// shorten them.
var (
	stopWaitAttempts = 10
	stopWaitDelay    = 3 * time.Second
)

// flinkStopWaitAttempts and flinkStopWaitDelay are the same bound for a FLINK_SQL deployment.
// Ververica cancels the job asynchronously and keeps reporting `Stopping` well past the 30s the
// other types need; deleting before it is terminal is refused with "Deleting a deployment which
// has job is not terminal status is not allowed".
const (
	flinkStopWaitAttempts = 150
	flinkStopWaitDelay    = 2 * time.Second
)

// flinkDeleteAttempts and flinkDeleteDelay retry a FLINK_SQL DELETE that Ververica refuses because
// its job has not reached a terminal state yet.
const (
	flinkDeleteAttempts = 6
	flinkDeleteDelay    = 10 * time.Second
)

// startWaitAttempts and startWaitDelay bound how long a START waits for a deployment that is
// still stopping to become startable.
var (
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
	TargetVersion     types.String `tfsdk:"target_version"`
	SqlScript         types.String `tfsdk:"sql_script"`
	GenerateTablesSql types.Bool   `tfsdk:"generate_tables_sql"`
	Autostart         types.Bool   `tfsdk:"autostart"`
}

func (r *applicationDeploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_deployment"
}
func (r *applicationDeploymentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An Application Deployment stores the configuration an Application runs with on an Environment. Supported application types are `Connector`, `Ksml` and `FLINK_SQL`.",

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
				MarkdownDescription: "The t-shirt size of the deployment. Optional for KSML and FLINK_SQL deployments; for FLINK_SQL it sizes the Flink TaskManager. The accepted sizes are configured per Platform Manager install (`axual.application-deployment.flink.deployment-sizes`, `...ksml.deployment-sizes`) and default to `XS`, `S`, `M`, `L` and `XL`, so your instance may accept a different set; the match is case-insensitive. No endpoint lists them, so an unknown size is only rejected once the apply reaches the API. If not specified, the Platform Manager will assign a default value.",
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
				// Only FLINK_SQL has to be replaced for a changed target: it rejects the change
				// outright (AXPD-11759), while the API patches the target of the other types.
				MarkdownDescription: "The id of the deployment target to deploy to. Required for FLINK_SQL deployments, where it must be the id of an `axual_flink_cluster` registered for the environment. For other deployment types the Platform Manager assigns a default target if not specified. Changing this value replaces a FLINK_SQL Application Deployment, whose deployment target can only be set when the deployment is created; for the other types the target is updated in place. Available targets can be listed via `GET /applications/{applicationId}/deployment-targets?environmentId={environmentId}`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIf(
						func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
							var deploymentType types.String
							resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("type"), &deploymentType)...)
							resp.RequiresReplace = isFlinkSQL(deploymentType.ValueString())
						},
						"A FLINK_SQL Application Deployment must be replaced to change its deployment target.",
						"A FLINK_SQL Application Deployment must be replaced to change its deployment target.",
					),
				},
			},
			"target_version": schema.StringAttribute{
				// UseStateForUnknown keeps an omitted target_version on the value Read last saw.
				MarkdownDescription: "The plugin version to deploy on the deployment target named by `target_id`. Only meaningful for a Connector deployment targeting a registered Kafka Connect cluster; omit it for legacy Axual Connect and for other deployment types - for legacy Axual Connect this value is always overwritten on read with the actually-deployed plugin version, so setting it explicitly can produce a diff that never clears. Changing this value updates the deployment in place, the same way changing `target_id` does for every type except FLINK_SQL.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sql_script": schema.StringAttribute{
				MarkdownDescription: "The user-authored Flink SQL script. Required for FLINK_SQL deployments. Each statement must be a `CREATE TEMPORARY TABLE` or an `INSERT INTO ... SELECT`; two or more top-level `INSERT INTO` statements must be wrapped in `BEGIN STATEMENT SET; ... END;`. With `generate_tables_sql = true` the platform generates the `CREATE TEMPORARY TABLE` statements from the application's approved topic access and the script carries only the `INSERT INTO ... SELECT`; with `false` you write the table DDL yourself. Kafka and schema registry connection options (`bootstrap.servers`, `properties.security.protocol`, `properties.sasl.*`, `ssl.*`, and the `avro-confluent` `url`/`basic-auth.*`/`bearer-auth.*` options) are injected by the platform and are rejected here, `connector` must stay `kafka` or `upsert-kafka`, and each side is declared with `key.format`/`value.format` rather than Flink's `format` shorthand. This field is Sensitive and will not be displayed in server log outputs when using Terraform commands.",
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
			"autostart": schema.BoolAttribute{
				MarkdownDescription: "Whether this resource starts the deployment. Defaults to `true`: the deployment is created only after an approved `axual_application_access_grant` and an Application Principal or Credential exist, and it is started on create and after every update. " +
					"Set it to `false` to only store the deployment target and configs, the same way the Self-Service UI does when a deployment target is confirmed. The deployment is then created before the principal or credential and is not started; an `axual_application_deployment_state` resource starts and stops it. " +
					"`false` is required for a Connector on a registered Kafka Connect cluster (`target_id` set), on MTLS and SASL_SCRAM clusters alike. A legacy Axual Connect Connector supports both values. " +
					"With `false`, an update that has to stop the deployment starts it again only if it was running before. Changing only this value changes nothing on the platform.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
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

// deploymentTypePlanModifier keeps the prior `type` when it is recomputed, so a change applies in
// place instead of replacing the deployment. Changing `application` already forces a replace on
// its own. Connector deployments need a Platform Manager with the `validateConfig` NPE fix to
// update in place at all - see the CHANGELOG.
type deploymentTypePlanModifier struct{}

func (m deploymentTypePlanModifier) Description(_ context.Context) string {
	return "Every Application Deployment type is updated in place; none is replaced just because `type` is recomputed."
}

func (m deploymentTypePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m deploymentTypePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Nothing to decide when the deployment is being created or destroyed.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	if req.PlanValue.IsUnknown() {
		resp.PlanValue = req.StateValue
	}
}

// ModifyPlan refuses autostart = true on a Kafka Connect target at plan time. It can only do that
// when the application already exists; otherwise Create and Update still refuse it at apply time.
func (r *applicationDeploymentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.provider.client == nil {
		return
	}
	var plan ApplicationDeploymentResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || plan.Autostart.IsUnknown() || !isAutostart(plan.Autostart) || plan.Application.IsUnknown() {
		return
	}
	if plan.TargetId.IsNull() || plan.TargetId.IsUnknown() || plan.TargetId.ValueString() == "" ||
		strings.HasPrefix(plan.TargetId.ValueString(), legacyAxualConnectTargetPrefix) {
		return
	}
	application, err := r.provider.client.GetApplication(plan.Application.ValueString())
	if err != nil {
		// Not decidable now; the apply-time check still applies.
		tflog.Debug(ctx, fmt.Sprintf("Skipping the plan-time autostart check, unable to read application: %s", err))
		return
	}
	plan.Type = types.StringValue(application.ApplicationType)
	if isKafkaConnectTarget(&plan) {
		addKafkaConnectAutostartError(&resp.Diagnostics)
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

	// With autostart = false this resource only stores the target and configs, like the Self-Service
	// UI does when a deployment target is confirmed: the platform needs no principal, credential or
	// grant for that, and the axual_application_deployment_state resource starts the deployment later.
	autostart := isAutostart(data.Autostart)
	canStart := false
	if autostart {
		var startDiags diag.Diagnostics
		canStart, startDiags = r.checkStartPrerequisites(&data)
		resp.Diagnostics.Append(startDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
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

	// The Uid of the created deployment comes from the Location header of the response, which the
	// API returns for every application type.
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
	// A FLINK_SQL deployment is created with a target only, so the response carries no configs yet
	// and mapping it would null the planned Flink values. Keep them - the follow-up PATCH is what
	// stores them - or an apply that does not reach the re-read below fails the consistency check.
	plannedSqlScript, plannedDeploymentSize, plannedGenerateTablesSql := data.SqlScript, data.DeploymentSize, data.GenerateTablesSql
	mapApplicationDeploymentByIdResponseToData(ctx, &data, createdDeployment)
	if isFlinkSQL(data.Type.ValueString()) {
		// `deployment_size` and `generate_tables_sql` are Computed, so an unset one is unknown in the
		// plan; only a known value may be carried over, the rest stays as the response mapped it.
		data.SqlScript = plannedSqlScript
		if !plannedDeploymentSize.IsUnknown() {
			data.DeploymentSize = plannedDeploymentSize
		}
		if !plannedGenerateTablesSql.IsUnknown() {
			data.GenerateTablesSql = plannedGenerateTablesSql
		}
	}

	// Save state as soon as the Id is known: a failure after this leaves the deployment trackable,
	// so a retried apply goes through Update() instead of orphaning it.
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
	// A deployment that cannot start yet is not asked to: the START only waits out its full budget
	// for a `start` rel the API will not offer, and then repeats the warning above.
	if canStart {
		if err := startApplicationDeployment(ctx, r.provider.client, data.Id.ValueString(), data.Type.ValueString(), 10*time.Second); err != nil {
			resp.Diagnostics.AddWarning(
				"Deployment created but START failed",
				fmt.Sprintf("The deployment was created and saved to state, but it is not running: %s. "+
					"Run 'terraform apply' again to retry starting the deployment.", err),
			)
			return
		}
		tflog.Info(ctx, "Successfully created and started Application Deployment")
		return
	}

	tflog.Info(ctx, "Successfully created Application Deployment, which cannot be started yet")
}

// isAutostart reads the `autostart` attribute. A state written by a provider version without it
// has no value, which keeps the behaviour those versions had: start the deployment.
func isAutostart(autostart types.Bool) bool {
	return autostart.IsNull() || autostart.IsUnknown() || autostart.ValueBool()
}

// checkStartPrerequisites is what `autostart = true` needs before the deployment is created: the
// principal or credential the START will use, and an approved access grant. It returns whether the
// deployment can be started right after it is created; a Connector without configs cannot.
func (r *applicationDeploymentResource) checkStartPrerequisites(data *ApplicationDeploymentResourceData) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	applicationURL := fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, data.Application.ValueString())
	environmentURL := fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, data.Environment.ValueString())

	if isFlinkSQL(data.Type.ValueString()) {
		// A FLINK_SQL job runs on the application's SASL/SCRAM Kafka credential, which the platform
		// injects into the generated SQL at deploy time. An mTLS Application Principal cannot be used.
		// The credential search endpoint takes plain ids (`applicationId=`/`environmentId=`), unlike the
		// principal search below, which takes resource URLs (`application=`/`environment=`).
		credentials, err := r.provider.client.FindApplicationCredentialByApplicationAndEnvironment(data.Application.ValueString(), data.Environment.ValueString())
		if err != nil {
			diags.AddError("Error querying for Application Credential for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
			return false, diags
		}
		if countScramCredentials(credentials) == 0 {
			diags.AddError(
				"Missing SASL/SCRAM Application Credential",
				"A FLINK_SQL Application Deployment requires an axual_application_credential supporting SCRAM_SHA_512 for this application and environment. "+
					"An axual_application_principal cannot be used: the platform rejects the deployment with "+
					"\"a Flink SQL application requires SASL/SCRAM Kafka credentials\".",
			)
			return false, diags
		}
	} else {
		if isKafkaConnectTarget(data) {
			addKafkaConnectAutostartError(&diags)
			return false, diags
		}

		// we count if there is at least one authentication defined for these application and environment
		authenticationCount := 0
		// We check if Application Principal exists for this environment and application
		applicationPrincipalsResponse, err := r.provider.client.FindApplicationPrincipalByApplicationAndEnvironment(applicationURL, environmentURL)
		if err != nil {
			diags.AddError("Error querying for Application Principal for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
			return false, diags
		}
		authenticationCount += len(applicationPrincipalsResponse.Embedded.ApplicationPrincipalResponses)
		if isKSML(data.Type.ValueString()) {
			// For KSML applications, we check if Application Credential exists for this environment and application
			applicationCredentialsResponse, err := r.provider.client.FindApplicationCredentialByApplicationAndEnvironment(data.Application.ValueString(), data.Environment.ValueString())
			if err != nil {
				diags.AddError("Error querying for Application Credential for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
				return false, diags
			}
			authenticationCount += len(applicationCredentialsResponse)
		}

		if authenticationCount == 0 {
			diags.AddError("Error from Terraform Provider validation", "Please first create an Application Principal or Application Credential for this application and environment")
			return false, diags
		}

		// Connector deployments require at least one active Application Principal: the deployment START
		// will fail at the API otherwise. The API does NOT auto-activate, so we surface a clear error here
		// instead of letting deployment creation fail with a generic platform error.
		if isConnector(data.Type.ValueString()) && countActivePrincipals(applicationPrincipalsResponse) == 0 {
			diags.AddError(
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
			return false, diags
		}
	}

	// We check if Approved Application Access Grant exists for this environment and application
	accessGrantRequest := webclient.ApplicationAccessGrantAttributes{
		ApplicationId: data.Application.ValueString(),
		EnvironmentId: data.Environment.ValueString(),
		Statuses:      "APPROVED",
	}
	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrantsByAttributes(accessGrantRequest)
	if err != nil {
		diags.AddError("Error querying for Application Access Grant for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
		return false, diags
	}
	// We do not allow creating Application Deployment if there is no Approved Application Access Grant, because we can't start the connector without it
	if len(applicationAccessGrant.Embedded.ApplicationAccessGrantResponses) == 0 {
		diags.AddError("Error from Terraform Provider validation", "Please first create and approve Application Access Grant for this application and environment. "+
			"To create the deployment before the grant, set `autostart = false` and start it with an axual_application_deployment_state resource.")
		return false, diags
	}

	// Registering a Connector deployment target and adding the configs later is a supported flow
	// (ConnectorDeploymentManager.validateConfig returns early on empty configs), so this only
	// warns: the deployment is created but offers no START action until `configs` are set.
	if isConnector(data.Type.ValueString()) && countConfigs(data.Configs) == 0 {
		diags.AddWarning(
			"Connector Application Deployment has no configs",
			"The deployment is created but cannot be started until `configs` are set.",
		)
		return false, diags
	}
	return true, diags
}

// isKafkaConnectTarget reports whether a Connector deployment targets a registered Kafka Connect
// cluster. No target, or the synthesized `axualconnect-` one, means legacy Axual Connect.
func isKafkaConnectTarget(data *ApplicationDeploymentResourceData) bool {
	return isConnector(data.Type.ValueString()) && !data.TargetId.IsNull() && !data.TargetId.IsUnknown() &&
		data.TargetId.ValueString() != "" && !strings.HasPrefix(data.TargetId.ValueString(), legacyAxualConnectTargetPrefix)
}

// addKafkaConnectAutostartError refuses autostart = true on a Kafka Connect cluster. That flow needs
// the principal or credential before the deployment, but the platform only writes a principal's key
// to the Connect cluster's Vault, and only creates a SASL_SCRAM credential, once the target is stored.
func addKafkaConnectAutostartError(diags *diag.Diagnostics) {
	diags.AddError(
		"Kafka Connect cluster needs autostart = false",
		"This Connector targets a registered Kafka Connect cluster, which is only supported with `autostart = false`. "+
			"With `autostart = true` the principal or credential has to exist before the deployment, but Platform Manager only stores a principal's private key in the Connect cluster's Vault, "+
			"and only creates a SASL_SCRAM credential, once the deployment and its target_id exist. "+
			"Set `autostart = false` on this axual_application_deployment, make the principal or credential depend on it, and start the deployment with an axual_application_deployment_state resource "+
			"that depends on the access grant approval. "+
			"If an axual_application_principal for this deployment was already created, replace it after that change (`terraform apply -replace=<principal address>`) so its key is stored for the Connect cluster.",
	)
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
	if err != nil && !errors.Is(err, webclient.NotFoundError) {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to read Application Deployment, got error: %s", err))
		return
	}
	// Deleted outside Terraform: plan to create it again, like the other resources do.
	if err != nil || len(ApplicationDeploymentFindByApplicationAndEnvironmentResponse.Embedded.ApplicationDeploymentResponses) == 0 {
		tflog.Warn(ctx, fmt.Sprintf("Application Deployment %s not found, removing it from the state", data.Id.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	err = mapApplicationDeploymentByApplicationAndEnvironmentResponseToData(ctx, &data, ApplicationDeploymentFindByApplicationAndEnvironmentResponse)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to map Application Deployment, got error: %s", err))
		return
	}
	// A state written before `autostart` existed has no value; those versions always started.
	data.Autostart = types.BoolValue(isAutostart(data.Autostart))
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

	if isAutostart(planData.Autostart) && isKafkaConnectTarget(&planData) {
		addKafkaConnectAutostartError(&resp.Diagnostics)
		return
	}

	// `autostart` only decides what this resource does, so changing nothing else is not a change on
	// the platform. This is also how a state written before `autostart` existed picks up its value.
	if isOnlyAutostartChange(&stateData, &planData) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
		return
	}

	status, err := r.provider.client.GetApplicationDeploymentStatus(planData.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Application Deployment status, got error: %s", err))
		return
	}
	wasRunning := shouldStopDeployment(planData.Type.ValueString(), status)
	// With autostart = false an axual_application_deployment_state resource owns whether the
	// deployment runs, so an update that has to stop it puts it back the way it was.
	restartAfterUpdate := isAutostart(planData.Autostart) || wasRunning

	// A running deployment cannot be updated: `beforeUpdateImpl` rejects a save while the desired
	// state is RUNNING, and `saveFlinkSql` rejects one while a Flink job actually is. Stop it and
	// wait, exactly as Delete does.
	if err := stopDeploymentAndWait(ctx, r.provider.client, planData.Id.ValueString(), planData.Type.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// A stopped connector is only paused, and stays on its Connect cluster with its consumers in the
	// group. Moved to another cluster, it would keep the partitions from the new connector, so reset
	// it first: that removes it from the old cluster and keeps its offsets.
	if isConnector(planData.Type.ValueString()) && isConnectTargetChange(stateData.TargetId, planData.TargetId) {
		warning, err := resetConnector(ctx, r.provider.client, planData.Id.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		if warning != "" {
			resp.Diagnostics.AddWarning("Application Deployment was not reset before its target changed", warning)
		}
	}

	ApplicationDeploymentUpdateRequest, err := createApplicationUpdateDeploymentRequestFromData(ctx, &planData, &stateData)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request struct for application deployment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	_, err = r.provider.client.UpdateApplicationDeployment(planData.Id.ValueString(), ApplicationDeploymentUpdateRequest)
	if err != nil {
		message := fmt.Sprintf("Unable to update Application Deployment, got error: %s", err)
		// The update was refused, so the deployment still has its old settings: start it again
		// instead of leaving a deployment that was running stopped by a failed apply.
		if wasRunning {
			if startErr := startApplicationDeployment(ctx, r.provider.client, planData.Id.ValueString(), planData.Type.ValueString(), 5*time.Second); startErr != nil {
				message += fmt.Sprintf(". It was stopped for the update and could not be started again: %s", startErr)
			} else {
				message += ". It was started again with its previous settings."
			}
		}
		resp.Diagnostics.AddError("Client Error", message)
		return
	}
	tflog.Info(ctx, "Successfully updated Application Deployment")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)

	if !restartAfterUpdate {
		tflog.Info(ctx, "Application Deployment was not running before the update, leaving it stopped")
		return
	}

	// Unlike Create, Update fails the apply when the deployment cannot be started: the resource
	// is already tracked, so an error here does not taint it and the next apply retries Update.
	if err := startApplicationDeployment(ctx, r.provider.client, planData.Id.ValueString(), planData.Type.ValueString(), 5*time.Second); err != nil {
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

	if err := stopDeploymentAndWait(ctx, r.provider.client, data.Id.ValueString(), data.Type.ValueString()); err != nil {
		if deploymentGone(r.provider.client, data.Id.ValueString()) {
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// No `delete` rel pre-check (AXPD-11714): a draining Flink job still offers it, a failed
	// deployment does not.
	if err := deleteWithRetry(data.Type.ValueString(), flinkDeleteDelay, func() error {
		return ignoreGone(r.provider.client.DeleteApplicationDeployment(data.Id.ValueString()))
	}); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Application Deployment, got error: %s", err))
		return
	}
}

func mapApplicationDeploymentByApplicationAndEnvironmentResponseToData(
	ctx context.Context,
	data *ApplicationDeploymentResourceData,
	applicationDeploymentResponse *webclient.ApplicationDeploymentFindByApplicationAndEnvironmentResponse) error {
	if len(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses) > 1 {
		return fmt.Errorf("error processing mapping application deployment response, multiple application deployments found for the application and environment")
	} else {
		data.Id = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Uid)
		data.Environment = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Environment.Uid)
		data.Application = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Application.Uid)
		mapTargetIdToData(data, applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].TargetId)
		mapTargetVersionToData(data, applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].TargetVersion)
		mapResponseConfigsToData(ctx, data, applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Application.ApplicationType, applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Configs)
		return nil
	}
}

func mapApplicationDeploymentByIdResponseToData(ctx context.Context, data *ApplicationDeploymentResourceData, applicationDeploymentResponse *webclient.ApplicationDeploymentResponse) {
	data.Id = types.StringValue(applicationDeploymentResponse.Uid)
	data.Environment = types.StringValue(applicationDeploymentResponse.Embedded.Environment.Uid)
	data.Application = types.StringValue(applicationDeploymentResponse.Embedded.Application.Uid)

	mapTargetIdToData(data, applicationDeploymentResponse.TargetId)
	mapTargetVersionToData(data, applicationDeploymentResponse.TargetVersion)
	mapResponseConfigsToData(ctx, data, applicationDeploymentResponse.Embedded.Application.ApplicationType, applicationDeploymentResponse.Configs)
}

func mapTargetIdToData(data *ApplicationDeploymentResourceData, targetId string) {
	data.TargetId = stringOrNull(targetId)
}

func mapTargetVersionToData(data *ApplicationDeploymentResourceData, targetVersion string) {
	data.TargetVersion = stringOrNull(targetVersion)
}

// isConnectTargetChange reports whether a Connector moves to another Connect runtime. No target and
// the synthesized `axualconnect-` target both mean legacy Axual Connect.
func isConnectTargetChange(state, plan types.String) bool {
	if plan.IsUnknown() {
		return false
	}
	return connectTargetKey(state) != connectTargetKey(plan)
}

func connectTargetKey(target types.String) string {
	if target.IsNull() || target.IsUnknown() || strings.HasPrefix(target.ValueString(), legacyAxualConnectTargetPrefix) {
		return ""
	}
	return target.ValueString()
}

// stringOrNull maps an empty API string to null, so an unset optional attribute shows no diff.
func stringOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
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
	// A Connector deployment with no stored target reads back as `axualconnect-<instance>`, which
	// is not a registered Connect cluster. Leave it out and let the platform resolve the target.
	if !data.TargetId.IsNull() && !data.TargetId.IsUnknown() &&
		!strings.HasPrefix(data.TargetId.ValueString(), legacyAxualConnectTargetPrefix) {
		ApplicationDeploymentRequest.TargetId = data.TargetId.ValueString()
	}
	if !data.TargetVersion.IsNull() && !data.TargetVersion.IsUnknown() {
		ApplicationDeploymentRequest.TargetVersion = data.TargetVersion.ValueString()
	}

	tflog.Info(ctx, fmt.Sprintf("Application request completed: %q", ApplicationDeploymentRequest))
	return ApplicationDeploymentRequest, nil
}

// createApplicationUpdateDeploymentRequestFromData builds the update request. The target is sent
// only when it changes, because Platform Manager 15.0.x refuses it. FLINK_SQL and the synthesized
// `axualconnect-` target are never sent; a Connector moved back to Axual Connect clears the target.
func createApplicationUpdateDeploymentRequestFromData(ctx context.Context, plan *ApplicationDeploymentResourceData, state *ApplicationDeploymentResourceData) (webclient.ApplicationDeploymentUpdateRequest, error) {
	configs, err := createConfigsForDeploymentType(plan)

	if err != nil {
		return webclient.ApplicationDeploymentUpdateRequest{}, err
	}

	ApplicationDeploymentUpdateRequest := webclient.ApplicationDeploymentUpdateRequest{
		Configs: configs,
		ClearTarget: isConnector(plan.Type.ValueString()) &&
			connectTargetKey(state.TargetId) != "" && connectTargetKey(plan.TargetId) == "" && !plan.TargetId.IsUnknown(),
	}
	targetChanged := !plan.TargetId.Equal(state.TargetId) || !plan.TargetVersion.Equal(state.TargetVersion)
	sendTarget := targetChanged && !isFlinkSQL(plan.Type.ValueString())
	if sendTarget && !plan.TargetId.IsNull() && !plan.TargetId.IsUnknown() &&
		!strings.HasPrefix(plan.TargetId.ValueString(), legacyAxualConnectTargetPrefix) {
		ApplicationDeploymentUpdateRequest.TargetId = plan.TargetId.ValueString()
	}
	if sendTarget && !plan.TargetVersion.IsNull() && !plan.TargetVersion.IsUnknown() {
		ApplicationDeploymentUpdateRequest.TargetVersion = plan.TargetVersion.ValueString()
	}

	tflog.Info(ctx, fmt.Sprintf("Application update request completed: %+v", ApplicationDeploymentUpdateRequest))
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

	// Any state can be imported: a deployment created with `autostart = false` is not running until
	// an axual_application_deployment_state starts it. `autostart` is not stored on the platform, so
	// the default is imported; a configuration that says `false` then only updates the state.
	var data ApplicationDeploymentResourceData
	mapApplicationDeploymentByIdResponseToData(ctx, &data, applicationDeployment)
	data.Autostart = types.BoolValue(true)

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

// isOnlyAutostartChange reports whether `autostart` is the only difference between state and plan.
// An unknown value is not comparable, so it cannot be ruled out as a change.
func isOnlyAutostartChange(state *ApplicationDeploymentResourceData, plan *ApplicationDeploymentResourceData) bool {
	for _, v := range []attr.Value{plan.Configs, plan.Definition, plan.DeploymentSize, plan.RestartPolicy,
		plan.TargetId, plan.TargetVersion, plan.SqlScript, plan.GenerateTablesSql} {
		if v.IsUnknown() {
			return false
		}
	}
	return plan.Configs.Equal(state.Configs) &&
		plan.Definition.Equal(state.Definition) &&
		plan.DeploymentSize.Equal(state.DeploymentSize) &&
		plan.RestartPolicy.Equal(state.RestartPolicy) &&
		plan.TargetId.Equal(state.TargetId) &&
		plan.TargetVersion.Equal(state.TargetVersion) &&
		plan.SqlScript.Equal(state.SqlScript) &&
		plan.GenerateTablesSql.Equal(state.GenerateTablesSql)
}

// stopDeploymentAndWait stops the deployment when the API still offers a `stop` action, then waits
// for the per-type status to report a terminal state - the rels cannot answer that, because both
// `stop` and `delete` are already gone or offered while a Flink job is still draining.
// Running out of attempts is not an error: the caller lets the API decide, as Delete always did.
func stopDeploymentAndWait(ctx context.Context, client *webclient.Client, id string, deploymentType string) error {
	status, err := client.GetApplicationDeploymentStatus(id)
	if err != nil {
		return fmt.Errorf("unable to get Application Deployment status, got error: %s", err)
	}
	if !shouldStopDeployment(deploymentType, status) {
		return nil
	}

	if err := stopWithRetry(ctx, client, id, deploymentType); err != nil {
		return fmt.Errorf("unable to stop Application Deployment, got error: %s", err)
	}

	attempts, delay := stopWaitBudget(deploymentType)
	tflog.Info(ctx, fmt.Sprintf("Stopped Application Deployment %s, waiting for it to stop running", id))
	started := time.Now()
	for i := 0; i < attempts; i++ {
		time.Sleep(delay)
		status, err := client.GetApplicationDeploymentStatus(id)
		if err != nil {
			return fmt.Errorf("unable to get Application Deployment status while waiting for it to stop, got error: %s", err)
		}
		if isDeploymentStopped(deploymentType, status) {
			tflog.Info(ctx, fmt.Sprintf("Application Deployment %s stopped after %s (%s)", id, time.Since(started).Round(time.Second), describeDeploymentStatus(status)))
			return nil
		}
	}
	tflog.Warn(ctx, fmt.Sprintf("Application Deployment %s did not report a stopped state within %s, continuing", id, time.Duration(attempts)*delay))
	return nil
}

// stopWithRetry retries a failed STOP, for example a 500 during a Connect rebalance (APCS-3090).
// Before a retry it reads the status, so a STOP that went through is not sent again.
func stopWithRetry(ctx context.Context, client *webclient.Client, id string, deploymentType string) error {
	var err error
	for i := 0; i < stopAttempts; i++ {
		if i > 0 {
			time.Sleep(stopRetryDelay)
			status, statusErr := client.GetApplicationDeploymentStatus(id)
			if statusErr == nil && !shouldStopDeployment(deploymentType, status) {
				return nil
			}
		}
		if err = operateDeployment(client, id, actionStop); err == nil {
			return nil
		}
		if isAlreadyStopped(err) {
			return nil
		}
		if !isRetryableStopError(err) {
			if status, statusErr := client.GetApplicationDeploymentStatus(id); statusErr == nil && !shouldStopDeployment(deploymentType, status) {
				return nil
			}
			return err
		}
		tflog.Warn(ctx, fmt.Sprintf("STOP of Application Deployment %s failed (attempt %d of %d): %s", id, i+1, stopAttempts, err))
	}
	return fmt.Errorf("after %d attempts: %s", stopAttempts, err)
}

// isRetryableStopError reports whether a STOP may succeed when sent again: a 5xx or a failed request
// can, a 4xx cannot.
func isRetryableStopError(err error) bool {
	if errors.Is(err, webclient.NotFoundError) {
		return false
	}
	if isConflict(err) {
		return true
	}
	var httpErr *webclient.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= http.StatusInternalServerError
	}
	return true
}

// stopWaitBudget is how long stopDeploymentAndWait waits for the given type to come to a halt.
func stopWaitBudget(deploymentType string) (int, time.Duration) {
	if isFlinkSQL(deploymentType) {
		return flinkStopWaitAttempts, flinkStopWaitDelay
	}
	return stopWaitAttempts, stopWaitDelay
}

// deleteWithRetry retries only a FLINK_SQL delete: the stop wait gives up rather than failing, and
// Ververica refuses the DELETE for as long as its job is not in a terminal state. Every other type
// is deleted once and its error returned unchanged.
func deleteWithRetry(deploymentType string, delay time.Duration, deleteDeployment func() error) error {
	if isFlinkSQL(deploymentType) {
		return Retry(flinkDeleteAttempts, delay, deleteDeployment)
	}
	// A concurrent change of the same deployment (409, optimistic lock) is retried.
	err := deleteDeployment()
	for attempt := 1; attempt < conflictAttempts && isConflict(err); attempt++ {
		time.Sleep(conflictDelay)
		err = deleteDeployment()
	}
	return err
}

// startApplicationDeployment starts or resumes the deployment and returns nil once it is running or
// once the action was accepted; the caller decides whether the error fails the apply. A stopped
// FLINK_SQL deployment offers `resume`, not `start`, so the rel decides which action is sent.
func startApplicationDeployment(ctx context.Context, client *webclient.Client, id string, deploymentType string, retryDelay time.Duration) error {
	status, err := client.GetApplicationDeploymentStatus(id)
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
		if status, err = client.GetApplicationDeploymentStatus(id); err != nil {
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
		return client.OperateApplicationDeployment(id, action, applicationStartRequest)
	})
	if err == nil {
		return nil
	}

	// The START may still have gone through on a request whose response was lost, so a deployment
	// that reports itself running counts as success rather than a failure.
	if currentStatus, statusErr := client.GetApplicationDeploymentStatus(id); statusErr == nil && isDeploymentRunning(deploymentType, currentStatus) {
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
		return actionStart
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
	live := liveDeploymentStatus(deploymentType, status)
	switch {
	case isFlinkSQL(deploymentType):
		// A stopped job reports `Stopped`; `Undeployed` is one that never ran or was reset.
		return live == "Stopped" || live == "Undeployed" || live == "Failed"
	case isKSML(deploymentType):
		// KSML never reports `Stopped`: a stopped app reads as `Undeployed`. `Failed` and `Completed`
		// are end states with nothing running, so waiting for them to change wastes the full budget.
		return live == "Undeployed" || live == "Failed" || live == "Completed"
	}
	// A Connector that never ran reports `Undefined` rather than `Stopped`, and a failed one stays `Failed`.
	return live == "Stopped" || live == "Undefined" || live == "Failed"
}

// isDeploymentRunning reports whether the deployment reports itself as running. Like
// isDeploymentStopped this reads the per-type status: the `_links` cannot answer it, because a
// deployment that is still shutting down offers no `start` either.
func isDeploymentRunning(deploymentType string, status *webclient.ApplicationDeploymentStatusResponse) bool {
	return liveDeploymentStatus(deploymentType, status) == "Running"
}

// liveDeploymentStatus is the status string the API reports for the deployment's type. It is the
// one place that picks the per-type field.
func liveDeploymentStatus(deploymentType string, status *webclient.ApplicationDeploymentStatusResponse) string {
	switch {
	case isFlinkSQL(deploymentType):
		return status.FlinkStatus.Status
	case isKSML(deploymentType):
		return status.KsmlStatus.Status
	}
	return status.ConnectorState.State
}

const (
	actionStart   = "START"
	actionStop    = "STOP"
	actionRestart = "RESTART"
	actionReset   = "RESET"

	connectorApplicationType = "Connector"
)

// operateDeployment sends one lifecycle action, so the action name is written once.
func operateDeployment(client *webclient.Client, id string, action string) error {
	return client.OperateApplicationDeployment(id, action, webclient.ApplicationDeploymentOperationRequest{Action: action})
}

// shouldStopDeployment reports whether the deployment has to be stopped before it can be
// updated or deleted. The status endpoint advertises a `stop` link exactly when STOP is a valid
// action for the deployment's current state, for every application type - so the provider does
// not have to mirror the per-type state machines (Connector states, KSML statuses, Flink job
// states), nor be released again when the platform adds a state.
func shouldStopDeployment(deploymentType string, status *webclient.ApplicationDeploymentStatusResponse) bool {
	if len(status.Links) > 0 {
		return status.Links.Has(webclient.RelStop)
	}
	// No rels at all is not the same as "already stopped": the caller may lack
	// APPLICATION_DEPLOYMENT_UPDATE, the KSML provisioner may be unavailable, or Ververica may report
	// a state Axual does not know. Fall back on what the deployment says it is doing, so a running
	// deployment is still stopped rather than left for the API to refuse the PATCH or DELETE.
	return isDeploymentRunning(deploymentType, status)
}
