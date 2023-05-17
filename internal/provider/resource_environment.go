package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"strings"

	"github.com/dcarbone/terraform-plugin-framework-utils/validation"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.ResourceType = environmentResourceType{}
var _ tfsdk.Resource = environmentResource{}
var _ tfsdk.ResourceWithImportState = environmentResource{}

type environmentResourceType struct{}

func (t environmentResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A environment represents a flow of information (messages), which is continuously updated. Read more: https://docs.axual.io/axual/2022.2/self-service/environment-management.html",

		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "The name of the environment. This must be in the format string-string (Needs to contain exactly one dash). The environment name is usually discussed and finalized as part of the Intake session or a follow up.",
				Required:            true,
				Type:                types.StringType,
			},
			"short_name": {
				MarkdownDescription: "The nshort ame of the environment. This must be in the format string-string (Needs to contain exactly one dash). The environment name is usually discussed and finalized as part of the Intake session or a follow up.",
				Required:            true,
				Type:                types.StringType,
			},
			"description": {
				MarkdownDescription: "A text describing the purpose of the environment.",
				Optional:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(1, -1),
				},
			},
			"color": {
				MarkdownDescription: "The nshort ame of the environment. This must be in the format string-string (Needs to contain exactly one dash). The environment name is usually discussed and finalized as part of the Intake session or a follow up.",
				Required:            true,
				Type:                types.StringType,
			},
			"authorization_issuer": {
				MarkdownDescription: "Fix (if applicable). Read more: https://docs.axual.io/axual/2022.2/self-service/environment-management.html#key-type",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"Stream owner", "Auto"}),
				}},
			"visibility": {
				MarkdownDescription: "The value type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2022.2/self-service/environment-management.html#value-type",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"Public", "Private"}),
				},
			},
			"owners": {
				MarkdownDescription: "The team owning this environment. Read more: https://docs.axual.io/axual/2022.2/self-service/environment-management.html#environment-owner",
				Required:            true,
				Type:                types.StringType,
			},
			"instance": {
				MarkdownDescription: "The nshort ame of the environment. This must be in the format string-string (Needs to contain exactly one dash). The environment name is usually discussed and finalized as part of the Intake session or a follow up.",
				Required:            true,
				Type:                types.StringType,
			},
			"retention_time": {
				MarkdownDescription: "Determines what to do with messages after a certain period. Read more: https://docs.axual.io/axual/2022.2/self-service/environment-management.html#retention-policy",
				Optional:            true,
				Computed:            true,
				Type:                types.Int64Type,
			},
			"partitions": {
				MarkdownDescription: "The nshort ame of the environment. This must be in the format string-string (Needs to contain exactly one dash). The environment name is usually discussed and finalized as part of the Intake session or a follow up.",
				Optional:            true,
				Computed:            true,
				Type:                types.Int64Type,
			},

			"properties": {
				MarkdownDescription: "Advanced (Kafka) properties for a environment in a given environment. Read more: https://docs.axual.io/axual/2022.2/self-service/advanced-features.html#configuring-environment-properties",
				Optional:            true,
				Computed:            true,
				Type:                types.MapType{ElemType: types.StringType},
			},
			"id": {
				Computed:            true,
				MarkdownDescription: "Environment unique identifier",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
		},
	}, nil
}

func (t environmentResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return environmentResource{
		provider: provider,
	}, diags
}

type environmentResourceData struct {
	Name                types.String `tfsdk:"name"`
	ShortName           types.String `tfsdk:"short_name"`
	Description         types.String `tfsdk:"description"`
	Color               types.String `tfsdk:"color"`
	AuthorizationIssuer types.String `tfsdk:"authorization_issuer"`
	Visibility          types.String `tfsdk:"visibility"`
	Owners              types.String `tfsdk:"owners"`
	RetentionTime       types.Int64  `tfsdk:"retention_time"`
	Instnce             types.String `tfsdk:"instance"`
	Id                  types.String `tfsdk:"id"`
	Partitions          types.Int64  `tfsdk:"partitions"`
	Properties          types.Map    `tfsdk:"properties"`
}

type environmentResource struct {
	provider provider
}

func (r environmentResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data environmentResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	environmentRequest, err := createEnvironmentRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CREATE request struct for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	properties := make(map[string]interface{})
	for key, value := range data.Properties.Elems {
		properties[key] = strings.Trim(value.String(), "\"")
	}
	environmentRequest.Properties = properties

	tflog.Info(ctx, fmt.Sprintf("Create environment request %q", environmentRequest))
	environment, err := r.provider.client.CreateEnvironment(environmentRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapEnvironmentResponseToData(ctx, &data, environment)
	tflog.Trace(ctx, "Created a environment resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r environmentResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data environmentResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	environment, err := r.provider.client.ReadEnvironment(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("READ request error for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapEnvironmentResponseToData(ctx, &data, environment)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r environmentResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data environmentResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	environmentRequest, err := createEnvironmentRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating UPDATE request struct for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	var oldPropertiesState map[string]string
	req.State.GetAttribute(ctx, tftypes.NewAttributePath().WithAttributeName("properties"), &oldPropertiesState)

	properties := make(map[string]interface{})

	for key, _ := range oldPropertiesState {
		properties[key] = nil
	}

	for key, value := range data.Properties.Elems {
		properties[key] = strings.Trim(value.String(), "\"")
	}

	environmentRequest.Properties = properties

	tflog.Info(ctx, fmt.Sprintf("Update environment request %q", environmentRequest))
	environment, err := r.provider.client.UpdateEnvironment(data.Id.Value, environmentRequest)
	if err != nil {
		resp.Diagnostics.AddError("UPDATE request error for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapEnvironmentResponseToData(ctx, &data, environment)
	tflog.Trace(ctx, "Updated a environment resource")
	tflog.Info(ctx, "Saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r environmentResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data environmentResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.provider.client.DeleteEnvironment(data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("DELETE request error for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
}

func (r environmentResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func createEnvironmentRequestFromData(ctx context.Context, data *environmentResourceData, r environmentResource) (webclient.EnvironmentRequest, error) {
	rawOwners, err := data.Owners.ToTerraformValue(ctx)
	if err != nil {
		return webclient.EnvironmentRequest{}, err
	}
	var owners string
	err = rawOwners.As(&owners)
	if err != nil {
		return webclient.EnvironmentRequest{}, err
	}
	owners = fmt.Sprintf("%s/groups/%v", r.provider.client.ApiURL, owners)
	instance := fmt.Sprintf("%s/instances/%v", r.provider.client.ApiURL, data.Instnce.Value)

	environmentRequest := webclient.EnvironmentRequest{
		Name:                data.Name.Value,
		ShortName:           data.ShortName.Value,
		Description:         data.Description.Value,
		Color:               data.Color.Value,
		AuthorizationIssuer: data.AuthorizationIssuer.Value,
		Visibility:          data.Visibility.Value,
		Owners:              owners,
		Instance:            instance,
		RetentionTime:       int(data.RetentionTime.Value),
		Partitions:          int(data.Partitions.Value),
	}

	// optional fields
	if !data.Description.Null {
		environmentRequest.Description = data.Description.Value
	}
	if !data.RetentionTime.Null {
		environmentRequest.RetentionTime = int(data.RetentionTime.Value)
	}
	if !data.Partitions.Null {
		environmentRequest.Partitions = int(data.Partitions.Value)
	}
	return environmentRequest, nil
}

func mapEnvironmentResponseToData(_ context.Context, data *environmentResourceData, environment *webclient.EnvironmentResponse) {
	data.Id = types.String{Value: environment.Uid}
	data.Name = types.String{Value: environment.Name}
	data.ShortName = types.String{Value: environment.ShortName}
	data.Description = types.String{Value: environment.Embedded.Instance.Description}
	data.Color = types.String{Value: environment.Color}
	data.Visibility = types.String{Value: environment.Visibility}
	data.AuthorizationIssuer = types.String{Value: environment.AuthorizationIssuer}
	data.Owners = types.String{Value: environment.Embedded.Owners.Uid}
	data.RetentionTime = types.Int64{Value: int64(environment.RetentionTime)}
	data.Partitions = types.Int64{Value: int64(environment.Partitions)}

	properties := make(map[string]attr.Value)
	for key, value := range environment.Properties {
		if value != nil {
			properties[key] = types.String{Value: value.(string)}
		}
	}
	data.Properties = types.Map{ElemType: types.StringType, Elems: properties}

	// optional fields
	if environment.Description == nil || len(environment.Description.(string)) == 0 {
		data.Description = types.String{Null: true}
	} else {
		data.Description = types.String{Value: environment.Description.(string)}
	}
}
