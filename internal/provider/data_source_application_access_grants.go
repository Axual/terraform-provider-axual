package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigValidators = &applicationAccessGrantsDataSource{}

func NewApplicationAccessGrantsDataSource(provider AxualProvider) datasource.DataSource {
	return &applicationAccessGrantsDataSource{provider: provider}
}

type applicationAccessGrantsDataSource struct {
	provider AxualProvider
}

type applicationAccessGrantsData struct {
	Topic       types.String           `tfsdk:"topic"`
	Environment types.String           `tfsdk:"environment"`
	Application types.String           `tfsdk:"application"`
	AccessType  types.String           `tfsdk:"access_type"`
	Statuses    []types.String         `tfsdk:"statuses"`
	Grants      []applicationGrantItem `tfsdk:"grants"`
}

type applicationGrantItem struct {
	Id              types.String `tfsdk:"id"`
	Status          types.String `tfsdk:"status"`
	AccessType      types.String `tfsdk:"access_type"`
	Application     types.String `tfsdk:"application"`
	ApplicationName types.String `tfsdk:"application_name"`
	Topic           types.String `tfsdk:"topic"`
	Environment     types.String `tfsdk:"environment"`
	RequestedAt     types.String `tfsdk:"requested_at"`
	decisionFields
}

func (d *applicationAccessGrantsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_access_grants"
}

func (d *applicationAccessGrantsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	computed := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: desc, Computed: true}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists application access grants, for example the open requests on a topic. Works on every Terraform version; for generating config, see `terraform query` (Terraform 1.14).",
		Attributes: map[string]schema.Attribute{
			"topic":       schema.StringAttribute{MarkdownDescription: "Topic UID.", Optional: true},
			"environment": schema.StringAttribute{MarkdownDescription: "Environment UID.", Optional: true},
			"application": schema.StringAttribute{MarkdownDescription: "Application UID or name.", Optional: true},
			"access_type": schema.StringAttribute{
				MarkdownDescription: "`CONSUMER` or `PRODUCER`.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf("CONSUMER", "PRODUCER")},
			},
			"statuses": schema.ListAttribute{
				MarkdownDescription: "Only grants with these statuses: `Pending`, `Approved`, `Rejected`, `Revoked`, `Cancelled`.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.List{listvalidator.ValueStringsAre(
					stringvalidator.OneOfCaseInsensitive("Pending", "Approved", "Rejected", "Revoked", "Cancelled"))},
			},
			"grants": schema.ListNestedAttribute{
				MarkdownDescription: "The grants that match, as far as the signed-in user may see them.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":               computed("Grant UID."),
					"status":           computed("Grant status."),
					"access_type":      computed("`CONSUMER` or `PRODUCER`."),
					"application":      computed("Application UID."),
					"application_name": computed("Application name."),
					"topic":            computed("Topic UID."),
					"environment":      computed("Environment UID."),
					"requested_at":     computed("When the application team last requested access."),
					"decision_reason":  computed(decisionReasonAttribute().MarkdownDescription),
					"decided_at":       computed(decidedAtAttribute().MarkdownDescription),
					"decided_by":       computed(decidedByAttribute().MarkdownDescription),
				}},
			},
		},
	}
}

func (d *applicationAccessGrantsDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf(path.MatchRoot("topic"), path.MatchRoot("application"), path.MatchRoot("environment")),
	}
}

func (d *applicationAccessGrantsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data applicationAccessGrantsData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	application := data.Application.ValueString()
	applicationFilter := ""
	if uidPattern.MatchString(strings.ToLower(application)) {
		applicationFilter = application
	}
	var statuses []string
	for _, s := range data.Statuses {
		statuses = append(statuses, strings.ToUpper(s.ValueString()))
	}
	grants, err := d.provider.client.SearchGrants(data.Topic.ValueString(), data.Environment.ValueString(), data.AccessType.ValueString(), applicationFilter, statuses)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list application access grants", fmt.Sprintf("Error message: %s", err))
		return
	}
	data.Grants = []applicationGrantItem{}
	for _, g := range grants {
		if application != "" && applicationFilter == "" && g.ApplicationName != application {
			continue
		}
		data.Grants = append(data.Grants, applicationGrantItem{
			Id:              types.StringValue(g.Uid),
			Status:          types.StringValue(g.Status),
			AccessType:      types.StringValue(g.AccessType),
			Application:     types.StringValue(g.ApplicationUid),
			ApplicationName: types.StringValue(g.ApplicationName),
			Topic:           stringOrNull(g.TopicUid),
			Environment:     stringOrNull(g.EnvironmentUid),
			RequestedAt:     stringOrNull(g.RequestedAt),
			decisionFields:  decisionFieldsOf(g),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
