package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
					stringplanmodifier.RequiresReplace(),
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
				MarkdownDescription: "The id of the deployment target to deploy to. Required for FLINK_SQL deployments, where it must be the id of an `axual_flink_cluster` registered for the environment. Available targets can be listed via `GET /applications/{applicationId}/deployment-targets`.",
				Optional:            true,
			},
			"sql_script": schema.StringAttribute{
				MarkdownDescription: "The transformation SQL for a FLINK_SQL deployment (an `INSERT INTO ... SELECT ...` statement, without credentials or fully-qualified topic names). Required for FLINK_SQL deployments. This field is Sensitive and will not be displayed in server log outputs when using Terraform commands.",
				Optional:            true,
				Sensitive:           true,
			},
			"generate_tables_sql": schema.BoolAttribute{
				MarkdownDescription: "For FLINK_SQL deployments, whether to auto-generate the `CREATE TABLE` statements for the topics referenced by `sql_script`. Optional for FLINK_SQL deployments.",
				Optional:            true,
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

	// FLINK_SQL applications have their Kafka credentials injected automatically by the
	// platform at deploy time, so there is no Application Principal or Application Credential
	// to check for and no active-principal precondition to enforce.
	var applicationPrincipalsResponse *webclient.ApplicationPrincipalFindByApplicationAndEnvironmentResponse
	if !isFlinkSQL(data.Type.ValueString()) {
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
			applicationCredentialsResponse, err := r.provider.client.FindApplicationCredentialByApplicationAndEnvironment(applicationURL, environmentURL)
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
		_, err = r.provider.client.UpdateApplicationDeployment(data.Id.ValueString(), data.Type.ValueString(), flinkUpdate)
		if err != nil {
			resp.Diagnostics.AddError("Error setting Flink SQL config for application deployment resource", fmt.Sprintf("Error message: %s", err.Error()))
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

	// START deployment with retry logic
	var applicationStartRequest = webclient.ApplicationDeploymentOperationRequest{
		Action: "START",
	}

	err = Retry(3, 10*time.Second, func() error {
		return r.provider.client.OperateApplicationDeployment(data.Id.ValueString(), "START", applicationStartRequest)
	})
	if err != nil {
		// Check if the error indicates the deployment is already running
		// This means a previous START succeeded but response timed out
		if strings.Contains(err.Error(), "Invalid action for this state of deployment") {
			tflog.Info(ctx, "Deployment appears to be already running - previous START may have succeeded")
			return
		}

		// Other errors - deployment created but START failed
		resp.Diagnostics.AddWarning(
			"Deployment created but START failed after retries",
			fmt.Sprintf("The deployment was created and saved to state, but could not be started after 3 attempts: %s. "+
				"Run 'terraform apply' again to retry starting the deployment.", err))
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
	var stateData ApplicationDeploymentResourceData
	var configData ApplicationDeploymentResourceData
	var planData ApplicationDeploymentResourceData

	diags := req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	diags = req.Config.Get(ctx, &configData)
	resp.Diagnostics.Append(diags...)
	diags = req.Plan.Get(ctx, &planData)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the current status of the application deployment
	applicationDeploymentStatus, err := r.provider.client.GetApplicationDeploymentStatus(planData.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Application Deployment status, got error: %s", err))
		return
	}

	if shouldStopDeployment(planData, applicationDeploymentStatus) {
		// If running, then stop the application deployment first
		var applicationStopRequest = webclient.ApplicationDeploymentOperationRequest{
			Action: "STOP",
		}
		err := r.provider.client.OperateApplicationDeployment(planData.Id.ValueString(), "STOP", applicationStopRequest)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to stop Application, got error: %s", err))
			return
		}
	}

	ApplicationDeploymentUpdateRequest, err := createApplicationUpdateDeploymentRequestFromData(ctx, &planData)

	// UpdateApplicationDeployment picks PUT or PATCH based on the application type: FLINK_SQL
	// deployments reject PUT ("PUT is not supported for Flink SQL deployments; use PATCH").
	_, err = r.provider.client.UpdateApplicationDeployment(planData.Id.ValueString(), planData.Type.ValueString(), ApplicationDeploymentUpdateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Application Deployment, got error: %s", err))
		return
	}
	tflog.Info(ctx, "Successfully updated Application Deployment")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)

	// START deployment with retry logic
	var applicationStartRequest = webclient.ApplicationDeploymentOperationRequest{
		Action: "START",
	}

	err = Retry(3, 5*time.Second, func() error {
		return r.provider.client.OperateApplicationDeployment(planData.Id.ValueString(), "START", applicationStartRequest)
	})
	if err != nil {
		// Check if the error indicates the deployment is already running
		// This means a previous START succeeded but response timed out
		if strings.Contains(err.Error(), "Invalid action for this state of deployment") {
			tflog.Info(ctx, "Deployment appears to be already running - previous START may have succeeded")
			return
		}

		// Other errors - deployment updated but START failed
		resp.Diagnostics.AddWarning(
			"Deployment updated but START failed after retries",
			fmt.Sprintf("The deployment was updated successfully but could not be started after 3 attempts: %s. "+
				"Run 'terraform apply' again to retry starting the deployment.", err))
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

	// Get the current status of the application deployment
	applicationDeploymentStatus, err := r.provider.client.GetApplicationDeploymentStatus(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Application Deployment status, got error: %s", err))
		return
	}

	if shouldStopDeployment(data, applicationDeploymentStatus) {
		// If running, then stop the application deployment first
		var applicationStopRequest = webclient.ApplicationDeploymentOperationRequest{
			Action: "STOP",
		}
		err := r.provider.client.OperateApplicationDeployment(data.Id.ValueString(), "STOP", applicationStopRequest)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to stop Application, got error: %s", err))
			return
		}

		// Poll until the deployment reaches a terminal state before attempting DELETE.
		// STOP is asynchronous: the platform only acknowledges the request, and a Flink job in
		// particular keeps draining for a while before it reports Undeployed/Failed. A DELETE
		// issued before that is rejected by the API, which would fail the destroy.
		//
		// TODO: the attempt count and delay below are not configurable. If this wait turns out to
		// be too short (or too long) in practice, expose it as a Terraform `timeouts` block on the
		// resource (timeouts.Block() + a context deadline) instead of hardcoding it here.
		maxRetries := 30
		retryDelay := 2 * time.Second
		for i := 0; i < maxRetries; i++ {
			time.Sleep(retryDelay)
			status, err := r.provider.client.GetApplicationDeploymentStatus(data.Id.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get deployment status while waiting for stop, got error: %s", err))
				return
			}

			// Check if deployment reached terminal state
			deploymentType := data.Type.ValueString()
			isTerminal := false
			if isFlinkSQL(deploymentType) {
				isTerminal = status.FlinkStatus.Status == "Undeployed" || status.FlinkStatus.Status == "Failed"
			} else if isKSML(deploymentType) {
				isTerminal = status.KsmlStatus.Status == "Undeployed"
			} else {
				isTerminal = status.ConnectorState.State == "Stopped"
			}

			if isTerminal {
				break
			}
		}
	}

	err = r.provider.client.DeleteApplicationDeployment(data.Id.ValueString())
	if err != nil {
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
	// A FLINK_SQL application deployment must be created with a deployment target only - the API
	// rejects `configs` on POST /application_deployments and requires the SQL config to be set via
	// a follow-up PATCH on the created deployment (see createFlinkConfigsUpdateRequest / Create()).
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
	if !data.TargetId.IsNull() && !data.TargetId.IsUnknown() {
		ApplicationDeploymentRequest.TargetId = data.TargetId.ValueString()
	}

	tflog.Info(ctx, fmt.Sprintf("Application request completed: %q", ApplicationDeploymentRequest))
	return ApplicationDeploymentRequest, nil
}

func createApplicationUpdateDeploymentRequestFromData(ctx context.Context, data *ApplicationDeploymentResourceData) (webclient.ApplicationDeploymentUpdateRequest, error) {
	configs, err := createConfigsForDeploymentType(data)

	if err != nil {
		return webclient.ApplicationDeploymentUpdateRequest{}, err
	}

	ApplicationDeploymentUpdateRequest := webclient.ApplicationDeploymentUpdateRequest{
		Configs: configs,
	}
	if !data.TargetId.IsNull() && !data.TargetId.IsUnknown() {
		ApplicationDeploymentUpdateRequest.TargetId = data.TargetId.ValueString()
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
		mapValue, diags := types.MapValue(types.StringType, configs)
		if diags.HasError() {
			tflog.Error(ctx, "Error creating configs map when mapping application deployment response")
		}
		data.Configs = mapValue
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

func shouldStopDeployment(data ApplicationDeploymentResourceData, status *webclient.ApplicationDeploymentStatusResponse) bool {
	// Check the connectorState.state before deciding to stop or delete directly
	// if the connectorState.state is not `Stopped`,
	// or if the ksmlStatus.Status is not `Undeployed`,
	// or if the flinkStatus.Status is not `Undeployed`,
	// we stop the deployment first
	deploymentType := data.Type.ValueString()
	if isKSML(deploymentType) {
		return status.KsmlStatus.Status != "Undeployed"
	}
	if isFlinkSQL(deploymentType) {
		// STOP is only a valid action from Starting/Running/Failing (per the platform's
		// Flink job state machine). Undeployed and Failed are both terminal states where
		// the API rejects STOP ("Invalid action STOP for deployment in state FAILED") -
		// from either of those, going straight to PATCH/START is correct.
		switch status.FlinkStatus.Status {
		case "Starting", "Running", "Failing":
			return true
		default:
			return false
		}
	}
	return status.ConnectorState.State != "Stopped"
}
