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
		MarkdownDescription: "Environments are used typically to support the application lifecycle, as it is moving from Development to Production.  In Self Service, they also allow you to test a feature in isolation, by making the environment Private. Read more: https://docs.axual.io/axual/2023.2/self-service/environment-management.html#managing-environments",

		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "A suitable name identifying this environment. This must be in the format string-string (Alphabetical characters, digits and the following characters are allowed: `- `,` _` ,` .`)",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(3, 50),
					validation.RegexpMatch((`^[a-z0-9\.\-_]+$`)),
				},
			},
			"short_name": {
				MarkdownDescription: "A short name that will uniquely identify this environment. The short name should be between 3 and 20 characters. no special characters are allowed.",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(3, 50),
					validation.RegexpMatch((`^[a-z0-9]+$`)),
				},
			},
			"description": {
				MarkdownDescription: "A text describing the purpose of the environment.",
				Optional:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Length(0, 200),
				},
			},
			"color": {
				MarkdownDescription: "The color used display the environment",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{
						"#80affe", "#4686f0", "#3347e1", "#1a2dbc", "#fee492", "#fbd04e", "#c2a7f9", "#8b58f3",
						"#e9b105", "#d19e02", "#6bdde0", "#21ccd2", "#19b9be", "#069499", "#532cd", "#3b0d98",
					}),
				}},
			"visibility": {
				MarkdownDescription: "Thi Private environments are only visible to the owning group (your team). They are not included in dashboard visualisations.",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"Public", "Private"}),
				},
			},
			"authorization_issuer": {
				MarkdownDescription: "This indicates if any deployments on this environment should be AUTO approved or requires approval from Stream Owner. For private environments, only AUTO can be selected.",
				Required:            true,
				Type:                types.StringType,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.OneOf, []string{"Stream owner", "Auto"}),
				},
			},
			"owners": {
				MarkdownDescription: "The id of the team owning this environment.",
				Required:            true,
				Type:                types.StringType,
			},
			"instance": {
				MarkdownDescription: "The id of the instance where this environment should be deployed.",
				Required:            true,
				Type:                types.StringType,
			},
			"retention_time": {
				MarkdownDescription: "The time in milliseconds after which the messages can be deleted from all streams. This is an optional field. If not specified, default value is 7 days (604800000).",
				Optional:            true,
				Computed:            true,
				Type:                types.Int64Type,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.GreaterThanOrEqualTo, 1000),
					validation.Compare(validation.LessThanOrEqualTo, int64(160704000000)),
				},
			},

			"partitions": {
				MarkdownDescription: "Defines the number of partitions configured for every stream of this tenant. This is an optional field. If not specified, default value is 12",
				Optional:            true,
				Computed:            true,
				Type:                types.Int64Type,
				Validators: []tfsdk.AttributeValidator{
					validation.Compare(validation.GreaterThanOrEqualTo, 1),
					validation.Compare(validation.LessThanOrEqualTo, 120000),
				},
			},
			"properties": {
				MarkdownDescription: "Environment-wide properties for all topics and applications.",
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
		if strings.Contains(err.Error(), statusNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Environment not found. Id: %s", data.Id.Value))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("READ request error for environment resource", fmt.Sprintf("Error message: %s", err.Error()))
		}
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
