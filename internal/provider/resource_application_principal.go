package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
)

var _ tfsdk.ResourceType = applicationPrincipalResourceType{}
var _ tfsdk.Resource = applicationPrincipalResource{}
var _ tfsdk.ResourceWithImportState = applicationPrincipalResource{}

type applicationPrincipalResourceType struct{}

func (t applicationPrincipalResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An ApplicationPrincipal is a security principal (certificate or comparable) that uniquely authenticates an Application on an Environment. Read more: https://docs.axual.io/axual/2023.2/self-service/application-management.html#configuring-application-securityauthentication",

		Attributes: map[string]tfsdk.Attribute{
			"principal": {
				MarkdownDescription: "The principal of an Application for an Environment",
				Required:            true,
				Type:                types.StringType,
			},
			"application": {
				MarkdownDescription: "A valid UID of an existing application",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"environment": {
				MarkdownDescription: "A valid UID of an existing environment",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"id": {
				Computed:            true,
				MarkdownDescription: "Application Principal identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
			"custom": {
				Optional:            true,
				MarkdownDescription: "A boolean identifying whether we are creating a custom principal. If true, the custom principal will be stored in principal property.  Custom principal allows an application with SASL+OAUTHBEARER to produce/consume a topic. Custom Application Principal certificate is used to authenticate your application with an IAM provider using the custom ApplicationPrincipal as Client ID",
				Type:                types.BoolType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (t applicationPrincipalResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return applicationPrincipalResource{
		provider: provider,
	}, diags
}

type applicationPrincipalResourceData struct {
	Principal   types.String `tfsdk:"principal"`
	Application types.String `tfsdk:"application"`
	Environment types.String `tfsdk:"environment"`
	Custom      types.Bool   `tfsdk:"custom"`
	Id          types.String `tfsdk:"id"`
}

type applicationPrincipalResource struct {
	provider provider
}

func (r applicationPrincipalResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
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
	tflog.Info(ctx, fmt.Sprintf("Create application principal request %q", applicationPrincipalRequest))
	applicationPrincipal, err := r.provider.client.CreateApplicationPrincipal(applicationPrincipalRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for application principal resource", fmt.Sprintf("Error message: %s %s", applicationPrincipal, err))
		return
	}

	var trimmedResponse = strings.Trim(string(applicationPrincipal), "\"")
	returnedUid := strings.ReplaceAll(trimmedResponse, fmt.Sprintf("%s/%s", r.provider.client.ApiURL, "application_principals/"), "")

	data.Id = types.String{Value: returnedUid}

	tflog.Trace(ctx, "Created an application principal resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r applicationPrincipalResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data applicationPrincipalResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationPrincipal, err := r.provider.client.ReadApplicationPrincipal(data.Id.Value)
	if err != nil {
		if strings.Contains(err.Error(), statusNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Application Principal not found. Id: %s", data.Id.Value))
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

func (r applicationPrincipalResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data applicationPrincipalResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var oldApplication string
	var oldEnvironment string
	req.State.GetAttribute(ctx, tftypes.NewAttributePath().WithAttributeName("application"), &oldApplication)
	req.State.GetAttribute(ctx, tftypes.NewAttributePath().WithAttributeName("environment"), &oldEnvironment)

	if data.Application.Value != oldApplication || data.Environment.Value != oldEnvironment {
		resp.Diagnostics.AddError(
			"Application Principal's environment UID or application UID cannot be updated",
			fmt.Sprint("To update Application Principal's environment UID or application UID resource please create new axual_application_principal resource"),
		)
		return
	}
	var applicationPrincipalUpdateRequest webclient.ApplicationPrincipalUpdateRequest
	applicationPrincipalUpdateRequest = webclient.ApplicationPrincipalUpdateRequest{
		Principal: data.Principal.Value,
	}
	tflog.Info(ctx, fmt.Sprintf("Update application principal request %q", applicationPrincipalUpdateRequest))
	applicationPrincipal, err := r.provider.client.UpdateApplicationPrincipal(data.Id.Value, applicationPrincipalUpdateRequest)
	if err != nil {
		resp.Diagnostics.AddError("PATCH request error for application principal resource", fmt.Sprintf("Error message: %s %s", applicationPrincipal, err))
		return
	}
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r applicationPrincipalResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data applicationPrincipalResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteApplicationPrincipal(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete application principal, got error: %s", err))
		return
	}
}

func (r applicationPrincipalResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func createApplicationPrincipalRequestFromData(ctx context.Context, data *applicationPrincipalResourceData, r applicationPrincipalResource) ([1]webclient.ApplicationPrincipalRequest, error) {
	rawEnvironment, err := data.Environment.ToTerraformValue(ctx)
	if err != nil {
		return [1]webclient.ApplicationPrincipalRequest{}, err
	}
	var environment string
	err = rawEnvironment.As(&environment)
	if err != nil {
		return [1]webclient.ApplicationPrincipalRequest{}, err
	}
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
			Principal:   data.Principal.Value,
			Application: application,
			Environment: environment,
		}
	// optional fields
	if !data.Custom.Null && data.Custom.Value {
		applicationPrincipalRequestArray[0].Custom = data.Custom.Value
	}
	return applicationPrincipalRequestArray, err
}

func mapApplicationPrincipalResponseToData(_ context.Context, data *applicationPrincipalResourceData, applicationPrincipal *webclient.ApplicationPrincipalResponse) {
	data.Id = types.String{Value: applicationPrincipal.Uid}
	data.Environment = types.String{Value: applicationPrincipal.Embedded.Environment.Uid}
	data.Application = types.String{Value: applicationPrincipal.Embedded.Application.Uid}
}
