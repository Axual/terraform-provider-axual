package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"

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
	Id             types.String `tfsdk:"id"`
	Application    types.String `tfsdk:"application"`
	Environment    types.String `tfsdk:"environment"`
	Configs        types.Map    `tfsdk:"configs"`
	Type           types.String `tfsdk:"type"`
	Definition     types.String `tfsdk:"definition"`
	DeploymentSize types.String `tfsdk:"deployment_size"`
	RestartPolicy  types.String `tfsdk:"restart_policy"`
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
				MarkdownDescription: "Connector config for Application Deployment. Required for Connector deployments. This field is Sensitive and will not be displayed in server log outputs when using Terraform commands. All available application plugin class names, plugin types and plugin configs are listed here in API- `GET: /api/connect_plugins?page=0&size=9999&sort=pluginClass` and in Axual Connect Docs: https://docs.axual.io/connect/Axual-Connect/developer/connect-plugins-catalog/connect-plugins-catalog.html",
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
				MarkdownDescription: "The deployment size for KSML applications. Optional for KSML deployments.",
				Optional:            true,
			},
			"restart_policy": schema.StringAttribute{
				MarkdownDescription: "The restart policy for KSML applications. Valid values are 'on_exit' and 'never'. Optional for KSML deployments.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("on_exit", "never"),
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

	// we count if there is at least one authentication defined for these application and environment
	authenticationCount := 0
	// We check if Application Principal exists for this environment and application
	applicationPrincipalsResponse, err := r.provider.client.FindApplicationPrincipalByApplicationAndEnvironment(applicationURL, environmentURL)
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
	ApplicationDeploymentRequest, err := createApplicationDeploymentRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request struct for application deployment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	_, err = r.provider.client.CreateApplicationDeployment(ApplicationDeploymentRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for application deployment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	// We search for the Application Deployment we just created because we need to save its UID, because creating it did not respond with UID.
	ApplicationDeploymentFindByApplicationAndEnvironmentResponse, err := r.provider.client.FindApplicationDeploymentByApplicationAndEnvironment(applicationURL, environmentURL)

	if err != nil {
		resp.Diagnostics.AddError("Error finding application deployment", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	var applicationStartRequest = webclient.ApplicationDeploymentOperationRequest{
		Action: "START",
	}

	// We start the Application Deployment
	err = r.provider.client.OperateApplicationDeployment(ApplicationDeploymentFindByApplicationAndEnvironmentResponse.Embedded.ApplicationDeploymentResponses[0].Uid, "START", applicationStartRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to start Application, got error: %s", err))
		return
	}
	mapApplicationDeploymentByApplicationAndEnvironmentResponseToData(ctx, &data, ApplicationDeploymentFindByApplicationAndEnvironmentResponse)
	tflog.Info(ctx, "Successfully created Application Deployment")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
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
	mapApplicationDeploymentByApplicationAndEnvironmentResponseToData(ctx, &data, ApplicationDeploymentFindByApplicationAndEnvironmentResponse)
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

	ApplicationDeploymentUpdateRequest, err := createApplicationUpdateDeploymentRequestFromData(ctx, &planData, r)

	_, err = r.provider.client.UpdateApplicationDeployment(planData.Id.ValueString(), ApplicationDeploymentUpdateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Application Deployment, got error: %s", err))
		return
	}
	tflog.Info(ctx, "Successfully updated Application Deployment")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)

	var applicationStartRequest = webclient.ApplicationDeploymentOperationRequest{
		Action: "START",
	}
	err = r.provider.client.OperateApplicationDeployment(planData.Id.ValueString(), "START", applicationStartRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to start Application, got error: %s", err))
		return
	}
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
	}

	err = r.provider.client.DeleteApplicationDeployment(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Application Deployment, got error: %s", err))
		return
	}
}

func mapApplicationDeploymentByApplicationAndEnvironmentResponseToData(ctx context.Context, data *ApplicationDeploymentResourceData, applicationDeploymentResponse *webclient.ApplicationDeploymentFindByApplicationAndEnvironmentResponse) {
	if len(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses) == 0 {
		tflog.Error(ctx, "Error processing mapping application deployment response, no application deployment found for the application and environment")
		return
	} else if len(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses) > 1 {
		tflog.Error(ctx, "Error processing mapping application deployment response, multiple application deployments found for the application and environment")
		return
	} else {
		data.Id = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Uid)
		data.Environment = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Environment.Uid)
		data.Application = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Application.Uid)
		mapResponseConfigsToData(ctx, data, applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Application.ApplicationType, applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Configs)
	}
}

func mapApplicationDeploymentByIdResponseToData(ctx context.Context, data *ApplicationDeploymentResourceData, applicationDeploymentResponse *webclient.ApplicationDeploymentResponse) {
	data.Id = types.StringValue(applicationDeploymentResponse.Uid)
	data.Environment = types.StringValue(applicationDeploymentResponse.Embedded.Environment.Uid)
	data.Application = types.StringValue(applicationDeploymentResponse.Embedded.Application.Uid)

	mapResponseConfigsToData(ctx, data, applicationDeploymentResponse.Embedded.Application.ApplicationType, applicationDeploymentResponse.Configs)
}

func createApplicationDeploymentRequestFromData(ctx context.Context, data *ApplicationDeploymentResourceData, r *applicationDeploymentResource) (webclient.ApplicationDeploymentCreateRequest, error) {
	configs, err := createConfigsForDeploymentType(data)

	if err != nil {
		return webclient.ApplicationDeploymentCreateRequest{}, err
	}

	ApplicationDeploymentRequest := webclient.ApplicationDeploymentCreateRequest{
		Application: data.Application.ValueString(),
		Environment: data.Environment.ValueString(),
		Configs:     configs,
	}

	tflog.Info(ctx, fmt.Sprintf("Application request completed: %q", ApplicationDeploymentRequest))
	return ApplicationDeploymentRequest, nil
}

func createApplicationUpdateDeploymentRequestFromData(ctx context.Context, data *ApplicationDeploymentResourceData, r *applicationDeploymentResource) (webclient.ApplicationDeploymentUpdateRequest, error) {
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

	// We iterate through the Configs and extract KSML-specific ones
	for _, config := range responseConfigs {
		switch config.ConfigKey {
		case "ksml_definition":
			ksmlDefinition = config.ConfigValue
		case "ksml_deployment_size":
			ksmlDeploymentSize = config.ConfigValue
		case "ksml_restart_policy":
			ksmlRestartPolicy = config.ConfigValue
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
	} else {
		if data.Type.IsNull() || data.Type.IsUnknown() {
			data.Type = types.StringValue("Connector")
		}
		mapValue, diags := types.MapValue(types.StringType, configs)
		if diags.HasError() {
			tflog.Error(ctx, "Error creating configs map when mapping application deployment response")
		}
		data.Configs = mapValue
	}
}

func isKSML(deploymentType string) bool {
	return deploymentType == "Ksml"
}

func shouldStopDeployment(data ApplicationDeploymentResourceData, status *webclient.ApplicationDeploymentStatusResponse) bool {
	// Check the connectorState.state before deciding to stop or delete directly
	// if the connectorState.state is not `Stopped`
	// or if the ksmlStatus.Status is not `Undeployed`,
	// we stop the deployment first
	if (isKSML(data.Type.ValueString()) && status.KsmlStatus.Status != "Undeployed") || (!isKSML(data.Type.ValueString()) && status.ConnectorState.State != "Stopped") {
		return true
	}
	return false
}
