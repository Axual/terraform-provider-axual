package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ tfsdk.DataSourceType = applicationAccessGrantDataSourceType{}
var _ tfsdk.DataSource = applicationAccessGrantDataSource{}

type applicationAccessGrantDataSourceType struct{}

func (t applicationAccessGrantDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Application Access Grant resource. Purpose of a grant is to request access to a topic in an environment. Read more: https://docs.axual.io/axual/2023.2/self-service/application-management.html#requesting-topic-access",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "Application Access Grant Unique Identifier",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
			"status": {
				MarkdownDescription: "Status of Application Access Grant",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
			"environment": {
				MarkdownDescription: "Environment Unique Identifier",
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (t applicationAccessGrantDataSourceType) NewDataSource(ctx context.Context, in tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return applicationAccessGrantDataSource{
		provider: provider,
	}, diags
}

type applicationAccessGrantDataSourceData struct {
	Id            types.String `tfsdk:"id"`
	EnvironmentId types.String `tfsdk:"environment"`
	Status        types.String `tfsdk:"status"`
}

type applicationAccessGrantDataSource struct {
	provider provider
}

func (d applicationAccessGrantDataSource) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var data applicationAccessGrantDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := d.provider.client.GetApplicationAccessGrant(data.Id.Value)
	if err != nil {
	    resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application access grant, got error: %s", err))
	return
	}

    mapApplicationAccessGrantDataSourceResponseToData(ctx, &data, applicationAccessGrant)
	
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func mapApplicationAccessGrantDataSourceResponseToData(ctx context.Context, data *applicationAccessGrantDataSourceData, applicationAccessGrant *webclient.ApplicationAccessGrant) {

	data.Id = types.String{Value: applicationAccessGrant.Uid}
	data.Status = types.String{Value: applicationAccessGrant.Status}
	data.EnvironmentId = types.String{Value: applicationAccessGrant.Embedded.Environment.Uid}
}
