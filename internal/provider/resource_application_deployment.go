package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &applicationDeploymentResource{}
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

	// Validate prerequisites
	if err := r.validatePrerequisites(ctx, &data, resp); err != nil {
		return
	}

	// Create Application Deployment
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

	// Find the created Application Deployment to get its UID
	ApplicationDeploymentFindByApplicationAndEnvironmentResponse, err := r.provider.client.FindApplicationDeploymentByApplicationAndEnvironment(applicationURL, environmentURL)
	if err != nil {
		resp.Diagnostics.AddError("Error finding application deployment", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	// Start the Connector Application
	if err := r.startApplication(ctx, ApplicationDeploymentFindByApplicationAndEnvironmentResponse.Embedded.ApplicationDeploymentResponses[0].Uid, resp); err != nil {
		return
	}

	mapApplicationDeploymentResponseToData(ctx, &data, ApplicationDeploymentFindByApplicationAndEnvironmentResponse)
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
			tflog.Warn(ctx, fmt.Sprintf("Application Deployment with ID: %s not found, removing from state", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to read Application Deployment, got error: %s", err))
			return
		}
	}

	// Verify the deployment still exists
	if len(ApplicationDeploymentFindByApplicationAndEnvironmentResponse.Embedded.ApplicationDeploymentResponses) == 0 {
		tflog.Warn(ctx, fmt.Sprintf("Application Deployment with ID: %s no longer exists, removing from state", data.Id.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	mapApplicationDeploymentResponseToData(ctx, &data, ApplicationDeploymentFindByApplicationAndEnvironmentResponse)
	tflog.Debug(ctx, "Successfully read Application Deployment")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData ApplicationDeploymentResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Stop the application if it's running
	if err := r.stopApplicationIfRunning(ctx, planData.Id.ValueString(), resp); err != nil {
		return
	}

	// Update the application deployment
	ApplicationDeploymentUpdateRequest, err := createApplicationUpdateDeploymentRequestFromData(ctx, &planData, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating update request", fmt.Sprintf("Error message: %s", err.Error()))
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

	// Start the application again
	if err := r.startApplication(ctx, planData.Id.ValueString(), resp); err != nil {
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

	// Stop the application if it's running
	if err := r.stopApplicationIfRunning(ctx, data.Id.ValueString(), resp); err != nil {
		return
	}

	// Delete the application deployment
	err := r.provider.client.DeleteApplicationDeployment(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Application Deployment, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Successfully deleted Application Deployment")
}

func (r *applicationDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, fmt.Sprintf("Starting import of Application Deployment with ID: %s", req.ID))
	
	idParts := strings.Split(req.ID, "/")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import Identifier",
			fmt.Sprintf("Expected import identifier with format: application_id/environment_id. Got: %s\n\nExample: terraform import axual_application_deployment.example abc123/def456", req.ID),
		)
		return
	}

	applicationId := idParts[0]
	environmentId := idParts[1]

	tflog.Debug(ctx, fmt.Sprintf("Importing Application Deployment for Application ID: %s, Environment ID: %s", applicationId, environmentId))

	// Validate that the application exists
	applicationWithUrl := fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, applicationId)
	environmentWithUrl := fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, environmentId)
	
	// Find the Application Deployment
	ApplicationDeploymentFindByApplicationAndEnvironmentResponse, err := r.provider.client.FindApplicationDeploymentByApplicationAndEnvironment(applicationWithUrl, environmentWithUrl)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			resp.Diagnostics.AddError(
				"Application Deployment Not Found",
				fmt.Sprintf("Unable to find Application Deployment for Application ID: %s and Environment ID: %s.\n\nPlease verify that:\n1. The Application ID exists\n2. The Environment ID exists\n3. An Application Deployment exists for this Application-Environment combination\n\nError: %s", applicationId, environmentId, err),
			)
		} else {
			resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to find Application Deployment, got error: %s", err))
		}
		return
	}

	deploymentResponse := ApplicationDeploymentFindByApplicationAndEnvironmentResponse.Embedded.ApplicationDeploymentResponses

	// Verify we found exactly one deployment per application and environment
	if len(deploymentResponse) == 0 {
		resp.Diagnostics.AddError(
			"Application Deployment Not Found",
			fmt.Sprintf("No Application Deployment found for Application ID: %s and Environment ID: %s", applicationId, environmentId),
		)
		return
	}

	if len(deploymentResponse) > 1 {
		resp.Diagnostics.AddWarning(
			"Multiple Application Deployments Found",
			fmt.Sprintf("Found %d Application Deployments for Application ID: %s and Environment ID: %s. Using the first one.", 
				len(ApplicationDeploymentFindByApplicationAndEnvironmentResponse.Embedded.ApplicationDeploymentResponses), applicationId, environmentId),
		)
	}

	// Map the response to Terraform state
	var data ApplicationDeploymentResourceData
	mapApplicationDeploymentResponseToData(ctx, &data, ApplicationDeploymentFindByApplicationAndEnvironmentResponse)

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

func (r *applicationDeploymentResource) validatePrerequisites(ctx context.Context, data *ApplicationDeploymentResourceData, resp *resource.CreateResponse) error {
	applicationURL := fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, data.Application.ValueString())
	environmentURL := fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, data.Environment.ValueString())

	// Check if Application Principal exists
	ApplicationPrincipalFindByApplicationAndEnvironmentResponse, err := r.provider.client.FindApplicationPrincipalByApplicationAndEnvironment(applicationURL, environmentURL)
	if err != nil {
		resp.Diagnostics.AddError("Error querying for Application Principal for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
		return err
	}

	if len(ApplicationPrincipalFindByApplicationAndEnvironmentResponse.Embedded.ApplicationPrincipalResponses) == 0 {
		resp.Diagnostics.AddError("Error from Terraform Provider validation", "Please first create Application Principal for this application and environment")
		return errors.New("missing application principal")
	}

	// Check if Approved Application Access Grant exists
	accessGrantRequest := webclient.ApplicationAccessGrantAttributes{
		ApplicationId: data.Application.ValueString(),
		EnvironmentId: data.Environment.ValueString(),
		Statuses:      "APPROVED",
	}
	
	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrantsByAttributes(accessGrantRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error querying for Application Access Grant for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
		return err
	}

	if len(applicationAccessGrant.Embedded.ApplicationAccessGrantResponses) == 0 {
		resp.Diagnostics.AddError("Error from Terraform Provider validation", "Please first create and approve Application Access Grant for this application and environment")
		return errors.New("missing approved application access grant")
	}

	return nil
}

func (r *applicationDeploymentResource) stopApplicationIfRunning(ctx context.Context, deploymentId string, resp interface{}) error {
	// Get the current status of the application deployment
	applicationDeploymentStatus, err := r.provider.client.GetApplicationDeploymentStatus(deploymentId)
	if err != nil {
		switch v := resp.(type) {
		case *resource.UpdateResponse:
			v.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Application Deployment status, got error: %s", err))
		case *resource.DeleteResponse:
			v.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Application Deployment status, got error: %s", err))
		}
		return err
	}

	// Stop the application if it's running
	if applicationDeploymentStatus.ConnectorState.State == "Running" {
		var applicationStopRequest = webclient.ApplicationDeploymentOperationRequest{
			Action: "STOP",
		}
		err := r.provider.client.OperateApplicationDeployment(deploymentId, "STOP", applicationStopRequest)
		if err != nil {
			switch v := resp.(type) {
			case *resource.UpdateResponse:
				v.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to stop Application, got error: %s", err))
			case *resource.DeleteResponse:
				v.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to stop Application, got error: %s", err))
			}
			return err
		}
	}

	return nil
}

func (r *applicationDeploymentResource) startApplication(ctx context.Context, deploymentId string, resp interface{}) error {
	var applicationStartRequest = webclient.ApplicationDeploymentOperationRequest{
		Action: "START",
	}
	
	err := r.provider.client.OperateApplicationDeployment(deploymentId, "START", applicationStartRequest)
	if err != nil {
		switch v := resp.(type) {
		case *resource.CreateResponse:
			v.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to start Application, got error: %s", err))
		case *resource.UpdateResponse:
			v.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to start Application, got error: %s", err))
		}
		return err
	}

	return nil
}

func mapApplicationDeploymentResponseToData(ctx context.Context, data *ApplicationDeploymentResourceData, applicationDeploymentResponse *webclient.ApplicationDeploymentFindByApplicationAndEnvironmentResponse) {
	data.Id = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Uid)
	data.Environment = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Environment.Uid)
	data.Application = types.StringValue(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Application.Uid)

	// Initialize the map for configs
	configs := make(map[string]attr.Value)

	// Map the configs of the first ApplicationDeploymentResponse
	if len(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses) > 0 {
		firstDeploymentResponse := applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0]
		tflog.Debug(ctx, fmt.Sprintf("firstDeploymentResponse: %+v", firstDeploymentResponse))
		
		// Iterate through the Configs and add them to the map
		for _, config := range firstDeploymentResponse.Configs {
			configs[config.ConfigKey] = types.StringValue(config.ConfigValue)
		}
	}

	mapValue, diags := types.MapValue(types.StringType, configs)
	if diags.HasError() {
		tflog.Error(ctx, "Error creating configs map when mapping deployment response")
	}
	
	// Set the Configs in the ApplicationDeploymentResourceData
	data.Configs = mapValue
	tflog.Debug(ctx, fmt.Sprintf("data.Configs: %+v", data.Configs))
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

	tflog.Debug(ctx, fmt.Sprintf("Application request completed: %+v", ApplicationDeploymentRequest))
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

	tflog.Debug(ctx, fmt.Sprintf("Application update request completed: %+v", ApplicationDeploymentUpdateRequest))
	return ApplicationDeploymentUpdateRequest, nil
}