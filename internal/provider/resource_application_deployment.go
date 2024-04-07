package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"github.com/dcarbone/terraform-plugin-framework-utils/validation"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.ResourceType = applicationDeploymentResourceType{}
var _ tfsdk.Resource = applicationDeploymentResource{}

type applicationDeploymentResourceType struct{}

func (t applicationDeploymentResourceType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An Application Deployment stores the configs for connector application type that is saved for an Application on an Environment.",

		Attributes: map[string]tfsdk.Attribute{
			"application": {
				MarkdownDescription: "A valid Uid of an existing application",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"environment": {
				MarkdownDescription: "A valid Uid of an existing environment",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"configs": {
				MarkdownDescription: "Connector config for Application Deployment",
				Required:            true,
				Type:                types.MapType{ElemType: types.StringType},
				Sensitive:           true,
			},
			"id": {
				Type:     types.StringType,
				Computed: true,
				Validators: []tfsdk.AttributeValidator{
					validation.RegexpMatch(`^[0-9a-fA-F]{32}$`),
				},
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
		},
	}, nil
}

func (t applicationDeploymentResourceType) NewResource(_ context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return applicationDeploymentResource{
		provider: provider,
	}, diags
}

type ApplicationDeploymentResourceData struct {
	Id          types.String `tfsdk:"id"`
	Application types.String `tfsdk:"application"`
	Environment types.String `tfsdk:"environment"`
	Configs     types.Map    `tfsdk:"configs"`
}

type applicationDeploymentResource struct {
	provider provider
}

func (r applicationDeploymentResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data ApplicationDeploymentResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	applicationURL := fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, data.Application.Value)
	environmentURL := fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, data.Environment.Value)

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
		ApplicationId: data.Application.Value,
		EnvironmentId: data.Environment.Value,
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
	var applicationStartRequest = webclient.ApplicationDeploymentOperationRequest{
		Action: "START",
	}

	// We start the Connector Application
	err = r.provider.client.OperateApplicationDeployment(ApplicationDeploymentFindByApplicationAndEnvironmentResponse.Embedded.ApplicationDeploymentResponses[0].Uid, "START", applicationStartRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to start Application, got error: %s", err))
		return
	}
	mapApplicationDeploymentResponseToData(ctx, &data, ApplicationDeploymentFindByApplicationAndEnvironmentResponse)
	tflog.Info(ctx, "Successfully created Application Deployment")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r applicationDeploymentResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data ApplicationDeploymentResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationWithUrl := fmt.Sprintf("%s/applications/%v", r.provider.client.ApiURL, data.Application.Value)
	environmentWithUrl := fmt.Sprintf("%s/environments/%v", r.provider.client.ApiURL, data.Environment.Value)
	ApplicationDeploymentFindByApplicationAndEnvironmentResponse, err := r.provider.client.FindApplicationDeploymentByApplicationAndEnvironment(applicationWithUrl, environmentWithUrl)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to find Application Deployment with ID: %s, got error: %s", data.Id.Value, err))
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

func (r applicationDeploymentResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
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
	applicationDeploymentStatus, err := r.provider.client.GetApplicationDeploymentStatus(planData.Id.Value)
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
		err := r.provider.client.OperateApplicationDeployment(planData.Id.Value, "STOP", applicationStopRequest)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to stop Application, got error: %s", err))
			return
		}
	}

	ApplicationDeploymentUpdateRequest, err := createApplicationUpdateDeploymentRequestFromData(ctx, &planData, r)

	_, err = r.provider.client.UpdateApplicationDeployment(planData.Id.Value, ApplicationDeploymentUpdateRequest)
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
	err = r.provider.client.OperateApplicationDeployment(planData.Id.Value, "START", applicationStartRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to start Application, got error: %s", err))
		return
	}
}

func (r applicationDeploymentResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data ApplicationDeploymentResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the current status of the application deployment
	applicationDeploymentStatus, err := r.provider.client.GetApplicationDeploymentStatus(data.Id.Value)
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
		err := r.provider.client.OperateApplicationDeployment(data.Id.Value, "STOP", applicationStopRequest)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to stop Application, got error: %s", err))
			return
		}
	}

	err = r.provider.client.DeleteApplicationDeployment(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Application Deployment, got error: %s", err))
		return
	}
}

func mapApplicationDeploymentResponseToData(_ context.Context, data *ApplicationDeploymentResourceData, applicationDeploymentResponse *webclient.ApplicationDeploymentFindByApplicationAndEnvironmentResponse) {
	data.Id = types.String{Value: applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Uid}
	data.Environment = types.String{Value: applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Environment.Uid}
	data.Application = types.String{Value: applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0].Embedded.Application.Uid}

	// Initialize the map for configs
	configs := make(map[string]attr.Value)

	// We want to map the configs of the first ApplicationDeploymentResponse
	// We check if there is at least one ApplicationDeploymentResponse
	if len(applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses) > 0 {
		firstDeploymentResponse := applicationDeploymentResponse.Embedded.ApplicationDeploymentResponses[0]
		fmt.Printf("firstDeploymentResponse: %+v\n", firstDeploymentResponse)
		// We iterate through the Configs and add them to the map
		for _, config := range firstDeploymentResponse.Configs {
			configs[config.ConfigKey] = types.String{Value: config.ConfigValue}
		}
	}

	// Set the Configs in the ApplicationDeploymentResourceData
	data.Configs = types.Map{ElemType: types.StringType, Elems: configs}
	fmt.Printf("data.Configs: %+v\n", data.Configs)
}

func createApplicationDeploymentRequestFromData(ctx context.Context, data *ApplicationDeploymentResourceData, r applicationDeploymentResource) (webclient.ApplicationDeploymentCreateRequest, error) {
	configs := make(map[string]string)

	for key, value := range data.Configs.Elems {
		strValue, ok := value.(types.String)
		if !ok {
			return webclient.ApplicationDeploymentCreateRequest{}, fmt.Errorf("type assertion to types.String failed for key: %s", key)
		}
		configs[key] = strValue.Value
	}

	ApplicationDeploymentRequest := webclient.ApplicationDeploymentCreateRequest{
		Application: data.Application.Value,
		Environment: data.Environment.Value,
		Configs:     configs,
	}

	tflog.Info(ctx, fmt.Sprintf("Application request completed: %q", ApplicationDeploymentRequest))
	return ApplicationDeploymentRequest, nil
}

func createApplicationUpdateDeploymentRequestFromData(ctx context.Context, data *ApplicationDeploymentResourceData, r applicationDeploymentResource) (webclient.ApplicationDeploymentUpdateRequest, error) {
	configs := make(map[string]string)

	for key, value := range data.Configs.Elems {
		strValue, ok := value.(types.String)
		if !ok {
			return webclient.ApplicationDeploymentUpdateRequest{}, fmt.Errorf("type assertion to types.String failed for key: %s", key)
		}
		configs[key] = strValue.Value
	}

	ApplicationDeploymentUpdateRequest := webclient.ApplicationDeploymentUpdateRequest{
		Configs: configs,
	}

	tflog.Info(ctx, fmt.Sprintf("Application update request completed: %q", ApplicationDeploymentUpdateRequest))
	return ApplicationDeploymentUpdateRequest, nil
}
