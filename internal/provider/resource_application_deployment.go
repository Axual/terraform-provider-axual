package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &applicationDeploymentResource{}

func NewApplicationDeploymentResource(provider AxualProvider) resource.Resource {
	return &applicationDeploymentResource{
		provider: provider,
	}
}

type applicationDeploymentResource struct {
	provider AxualProvider
}

type ApplicationDeploymentResourceData struct {
	Id          types.String `tfsdk:"id"`
	Application types.String `tfsdk:"application"`
	Environment types.String `tfsdk:"environment"`
	Configs     types.Map    `tfsdk:"configs"`
}

func (r *applicationDeploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_deployment"
}
func (r *applicationDeploymentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An Application Deployment stores the configs for connector application type that is saved for an Application on an Environment.",

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
			"configs": schema.MapAttribute{
				MarkdownDescription: "Connector config for Application Deployment. This field is Sensitive and will not be displayed in server log outputs when using Terraform commands. All available application plugin class names, plugin types and plugin configs are listed here in API- `GET: /api/connect_plugins?page=0&size=9999&sort=pluginClass` and in Axual Connect Docs: https://docs.axual.io/connect/Axual-Connect/developer/connect-plugins-catalog/connect-plugins-catalog.html",
				Required:            true,
				ElementType:         types.StringType,
				Sensitive:           true,
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
	applicationURL := fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, data.Application.ValueString())
	environmentURL := fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, data.Environment.ValueString())

	// We check if Application Principal exists for this environment and application
	ApplicationPrincipalFindByApplicationAndEnvironmentResponse, err := r.provider.client.FindApplicationPrincipalByApplicationAndEnvironment(applicationURL, environmentURL)
	if err != nil {
		resp.Diagnostics.AddError("Error querying for Application Principal for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	// We do not allow creating Application Deployment if there is no Application Principal because we can't start the connector without it
	if len(ApplicationPrincipalFindByApplicationAndEnvironmentResponse.Embedded.ApplicationPrincipalResponses) == 0 {
		resp.Diagnostics.AddError("Error from Terraform Provider validation", "Please first create Application Principal for this application and environment")
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

	// We search for the Application Deployment we just created, because we need to save its UID, because creating it did not respond with UID.
	ApplicationDeploymentFindByApplicationAndEnvironmentResponse, err := r.provider.client.FindApplicationDeploymentByApplicationAndEnvironment(applicationURL, environmentURL)

	if err != nil {
		resp.Diagnostics.AddError("Error finding application deployment", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	// Map response to data
	mapApplicationDeploymentResponseToData(ctx, &data, ApplicationDeploymentFindByApplicationAndEnvironmentResponse)

	// Save state IMMEDIATELY after creation, BEFORE starting
	// This prevents state loss if the START operation times out
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
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
		// This means a previous START succeeded but the response timed out
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
	mapApplicationDeploymentResponseToData(ctx, &data, ApplicationDeploymentFindByApplicationAndEnvironmentResponse)
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

	// Check the connectorState.state before deciding to stop or delete directly
	if applicationDeploymentStatus.ConnectorState.State == "Running" {
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

	// START deployment with retry logic
	var applicationStartRequest = webclient.ApplicationDeploymentOperationRequest{
		Action: "START",
	}
	err = Retry(3, 5*time.Second, func() error {
		return r.provider.client.OperateApplicationDeployment(planData.Id.ValueString(), "START", applicationStartRequest)
	})
	if err != nil {
		// Check if the error indicates the deployment is already running
		// This means a previous START succeeded but the response timed out
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

	// Check the connectorState.state before deciding to stop or delete directly
	if applicationDeploymentStatus.ConnectorState.State == "Running" {
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

func mapApplicationDeploymentResponseToData(ctx context.Context, data *ApplicationDeploymentResourceData, applicationDeploymentResponse *webclient.ApplicationDeploymentFindByApplicationAndEnvironmentResponse) {
	data.Id = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Uid)
	data.Environment = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Environment.Uid)
	data.Application = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Application.Uid)

	// Initialize the map for configs
	configs := make(map[string]attr.Value)

	// We want to map the configs of the first ApplicationDeploymentResponse
	// We check if there is at least one ApplicationDeploymentResponse
	if len(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses) > 0 {
		firstDeploymentResponse := applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0]
		fmt.Printf("firstDeploymentResponse: %+v\n", firstDeploymentResponse)
		// We iterate through the Configs and add them to the map
		for _, config := range firstDeploymentResponse.Configs {
			configs[config.ConfigKey] = types.StringValue(config.ConfigValue)
		}
	}
	mapValue, diags := types.MapValue(types.StringType, configs)
	if diags.HasError() {
		tflog.Error(ctx, "Error creating members slice when mapping group response")
	}
	// Set the Configs in the ApplicationDeploymentResourceData
	data.Configs = mapValue

	fmt.Printf("data.Configs: %+v\n", data.Configs)
}

func createApplicationDeploymentRequestFromData(ctx context.Context, data *ApplicationDeploymentResourceData, r *applicationDeploymentResource) (webclient.ApplicationDeploymentCreateRequest, error) {
	configs := make(map[string]string)

	for key, value := range data.Configs.Elements() {
		strValue, ok := value.(types.String)
		if !ok {
			return webclient.ApplicationDeploymentCreateRequest{}, fmt.Errorf("type assertion to types.String failed for key: %s", key)
		}
		configs[key] = strValue.ValueString()
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
	configs := make(map[string]string)

	for key, value := range data.Configs.Elements() {
		strValue, ok := value.(types.String)
		if !ok {
			return webclient.ApplicationDeploymentUpdateRequest{}, fmt.Errorf("type assertion to types.String failed for key: %s", key)
		}
		configs[key] = strValue.ValueString()
	}

	ApplicationDeploymentUpdateRequest := webclient.ApplicationDeploymentUpdateRequest{
		Configs: configs,
	}

	tflog.Info(ctx, fmt.Sprintf("Application update request completed: %q", ApplicationDeploymentUpdateRequest))
	return ApplicationDeploymentUpdateRequest, nil
}
