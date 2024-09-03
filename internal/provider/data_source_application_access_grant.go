package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &applicationAccessGrantDataSource{}

func NewApplicationAccessGrantDataSource(provider AxualProvider) datasource.DataSource {
	return &applicationAccessGrantDataSource{
		provider: provider,
	}
}

type applicationAccessGrantDataSource struct {
	provider AxualProvider
}

type applicationAccessGrantDataSourceData struct {
	Id            types.String `tfsdk:"id"`
	ApplicationId types.String `tfsdk:"application"`
	TopicId       types.String `tfsdk:"topic"`
	EnvironmentId types.String `tfsdk:"environment"`
	Status        types.String `tfsdk:"status"`
	AccessType    types.String `tfsdk:"access_type"`
}

func (d *applicationAccessGrantDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_access_grant"
}

func (d *applicationAccessGrantDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Application Access Grant resource. Purpose of a grant is to request access to a topic in an environment. Read more: https://docs.axual.io/axual/2024.2/self-service/application-management.html#requesting-topic-access",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Application Access Grant Unique Identifier",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of Application Access Grant",
				Computed:            true,
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "Environment Unique Identifier",
				Required:            true,
			},
			"topic": schema.StringAttribute{
				MarkdownDescription: "Topic Unique Identifier",
				Required:            true,
			},
			"application": schema.StringAttribute{
				MarkdownDescription: "Application Unique Identifier",
				Required:            true,
			},
			"access_type": schema.StringAttribute{
				MarkdownDescription: "Application Access Type. Accepted values: CONSUMER, PRODUCER",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("CONSUMER", "PRODUCER"),
				},
			},
		},
	}
}

func (d applicationAccessGrantDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data applicationAccessGrantDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	accessGrantRequest := webclient.ApplicationAccessGrantAttributes{
		TopicId:       data.TopicId.ValueString(),
		ApplicationId: data.ApplicationId.ValueString(),
		EnvironmentId: data.EnvironmentId.ValueString(),
		AccessType:    data.AccessType.ValueString(),
	}

	applicationAccessGrant, err := d.provider.client.GetApplicationAccessGrantsByAttributes(accessGrantRequest)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application access grant, got error: %s", err))
		return
	}

	mapApplicationAccessGrantDataSourceResponseToData(ctx, &data, applicationAccessGrant)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapApplicationAccessGrantDataSourceResponseToData(ctx context.Context, data *applicationAccessGrantDataSourceData, applicationAccessGrant *webclient.GetApplicationAccessGrantsByAttributeResponse) {
	applicationAccessResponse := applicationAccessGrant.Embedded.ApplicationAccessGrantResponses[0]
	data.Id = types.StringValue(applicationAccessResponse.Uid)
	data.Status = types.StringValue(applicationAccessResponse.Status)
	data.AccessType = types.StringValue(applicationAccessResponse.AccessType)
	data.EnvironmentId = types.StringValue(applicationAccessResponse.Embedded.Environment.Uid)
	data.TopicId = types.StringValue(applicationAccessResponse.Embedded.Stream.Uid)
	data.ApplicationId = types.StringValue(applicationAccessResponse.Embedded.Application.Uid)
}
