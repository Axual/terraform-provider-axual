package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &applicationCredentialResource{}
var _ resource.ResourceWithImportState = &applicationCredentialResource{}

func NewApplicationCredentialResource(provider AxualProvider) resource.Resource {
	return &applicationCredentialResource{
		provider: provider,
	}
}

type applicationCredentialResource struct {
	provider AxualProvider
}

type ApplicationCredentialResponse struct {
	AuthData authData `json:"auth_data"`
}

type authData struct {
	Password string `json:"password"`
	Provider string `json:"provider"`
	Clusters string `json:"clusters"`
	UserName string `json:"username"`
}

type applicationCredentialResourceData struct {
	Id            types.String   `tfsdk:"id"`
	ApplicationId types.String   `tfsdk:"application"`
	EnvironmentId types.String   `tfsdk:"environment"`
	Target        types.String   `tfsdk:"target"`
	UserName      types.String   `tfsdk:"username"`
	Password      types.String   `tfsdk:"password"`
	Clusters      types.String   `tfsdk:"clusters"`
	Description   types.String   `tfsdk:"description"`
	AuthProvider  types.String   `tfsdk:"auth_provider"`
	Types         []types.String `tfsdk:"types"`
}

func (r *applicationCredentialResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_credential"
}

func (r *applicationCredentialResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "An Application Credential is a security credential (SASL) that uniquely authenticates an Application in an Environment. Read more: https://docs.axual.io/axual/2024.4/self-service/application-management.html#configuring-application-securityauthentication",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application Credential Id",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application": schema.StringAttribute{
				MarkdownDescription: "A valid Id of an existing application",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "A valid Id of an existing environment",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Username associated with the credentials",
			},
			"password": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true, // hide password
				MarkdownDescription: "Password for the credentials. This value is sensitive and will not be printed in terraform apply output.",
			},
			"target": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The authentication credential provider (e.g., Apache Kafka, Schema Registry).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("KAFKA"),
				},
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description information for the credentials.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"clusters": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster information for the credentials.",
			},
			"auth_provider": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The authentication provider (e.g., Apache Kafka, Schema Registry).",
			},
			"types": schema.ListAttribute{
				MarkdownDescription: "List of authentication types.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *applicationCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data applicationCredentialResourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	applicationCredentialCreateRequest, err := createApplicationCredentialRequestFromData(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CREATE request struct for application credential resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Create application credential request %q", applicationCredentialCreateRequest))
	applicationCredential, err := r.provider.client.CreateApplicationCredential(applicationCredentialCreateRequest)

	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for application credential resource", fmt.Sprintf("Error message: %s %s", applicationCredential, err))
		return
	}

	data.UserName = types.StringValue(applicationCredential.AuthData.Username)
	data.Password = types.StringValue(applicationCredential.AuthData.Password) // Will be stored but not printed
	data.Clusters = types.StringValue(applicationCredential.AuthData.Clusters)
	data.AuthProvider = types.StringValue(applicationCredential.AuthData.Provider)

	tflog.Trace(ctx, "Created an application credential resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data applicationCredentialResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentials, err := r.provider.client.FindApplicationCredentialByApplicationAndEnvironment(data.ApplicationId.ValueString(), data.EnvironmentId.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Error querying for Application Credential for this application and environment", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	if len(credentials) == 0 {
		tflog.Info(ctx, "No credentials found, removing resource")
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Info(ctx, "Checking for matching credentials")
	var foundMatch bool
	var matchedCredential webclient.ApplicationCredentialFindByApplicationAndEnvironmentResponse

	for _, credential := range credentials {
		if credential.Username == data.UserName.ValueString() &&
			credential.Application.ID == data.ApplicationId.ValueString() &&
			credential.Environment.ID == data.EnvironmentId.ValueString() {
			tflog.Info(ctx, "Found matching credential, updating")
			matchedCredential = credential
			foundMatch = true
			break
		}
	}

	if !foundMatch {
		tflog.Info(ctx, "No matching credentials found, removing resource")
		resp.State.RemoveResource(ctx)
		return
	}

	applicationCredentialJSON, err := json.MarshalIndent(credentials, "", "  ")
	tflog.Info(ctx, "Application Credential Data:", map[string]interface{}{
		"data": string(applicationCredentialJSON),
	})
	mapApplicationCredentialResponseToData(ctx, &data, &matchedCredential)

	tflog.Info(ctx, "Saving updated credential")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

}

func (r *applicationCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data applicationCredentialResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var usernameConfig = webclient.NameConfig{
		Username: data.UserName.ValueString(),
	}

	applicationCredentialDeleteRequest := webclient.ApplicationCredentialDeleteRequest{
		ApplicationId: data.ApplicationId.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		Target:        data.Target.ValueString(),
		Configs:       usernameConfig,
	}

	err := r.provider.client.DeleteApplicationCredential(applicationCredentialDeleteRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete application principal, got error: %s", err))
		return
	}

}

func (r *applicationCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func createApplicationCredentialRequestFromData(ctx context.Context, data *applicationCredentialResourceData) (webclient.ApplicationCredentialCreateRequest, error) {
	rawEnvironmentId, err := data.EnvironmentId.ToTerraformValue(ctx)
	if err != nil {
		return webclient.ApplicationCredentialCreateRequest{}, err
	}
	var environmentId string
	err = rawEnvironmentId.As(&environmentId)
	if err != nil {
		return webclient.ApplicationCredentialCreateRequest{}, err
	}

	rawApplicationId, err := data.ApplicationId.ToTerraformValue(ctx)
	if err != nil {
		return webclient.ApplicationCredentialCreateRequest{}, err
	}
	var applicationId string
	err = rawApplicationId.As(&applicationId)
	if err != nil {
		return webclient.ApplicationCredentialCreateRequest{}, err
	}

	applicationCredentialRequest := webclient.ApplicationCredentialCreateRequest{
		ApplicationId: applicationId,
		EnvironmentId: environmentId,
		Target:        data.Target.ValueString(),
	}

	return applicationCredentialRequest, err
}

func mapApplicationCredentialResponseToData(_ context.Context, data *applicationCredentialResourceData, applicationCredential *webclient.ApplicationCredentialFindByApplicationAndEnvironmentResponse) {
	data.Id = types.StringValue(applicationCredential.ID)
	data.Description = types.StringValue(applicationCredential.Description)
	data.UserName = types.StringValue(applicationCredential.Username)
	data.Types = convertAuthTypeListToTypesStringList(applicationCredential.Types)
	data.Clusters = types.StringValue(applicationCredential.Metadata.Clusters)
}

func convertAuthTypeListToTypesStringList(input []webclient.AuthType) []types.String {
	var result []types.String
	for _, v := range input {
		result = append(result, types.StringValue(v.Type))
	}
	return result
}
