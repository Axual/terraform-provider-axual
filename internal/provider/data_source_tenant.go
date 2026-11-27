package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &tenantDataSource{}

func NewTenantDataSource(provider AxualProvider) datasource.DataSource {
	return &tenantDataSource{provider: provider}
}

type tenantDataSource struct {
	provider AxualProvider
}

func (d *tenantDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant"
}

func (d *tenantDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id":         schema.StringAttribute{MarkdownDescription: "The ID of the tenant.", Computed: true},
		"name":       schema.StringAttribute{MarkdownDescription: "The name of the tenant.", Computed: true},
		"short_name": schema.StringAttribute{MarkdownDescription: "The short name of the tenant. It is part of topic and group names on Clusters whose pattern has the tenant placeholder.", Computed: true},
		"authentication_methods": schema.ListAttribute{
			MarkdownDescription: "The login methods that the Instances of the tenant may use, by rank: `MUTUAL_TLS`, `SCRAM_SHA_256`, `SCRAM_SHA_512`, `OAUTH_BEARER` or `PLAIN`.",
			ElementType:         types.StringType,
			Computed:            true,
		},
		"history_retention_days": schema.Int64Attribute{
			MarkdownDescription: "How many days the history (audit events) of the tenant is kept. `0` keeps it forever. `null` when the tenant uses the platform value.",
			Computed:            true,
		},
		"onboarding_completed": schema.BoolAttribute{
			MarkdownDescription: "Whether the onboarding wizard of the Self-Service UI is finished for the tenant.",
			Computed:            true,
		},
		"unique_consumer_group": schema.BoolAttribute{
			MarkdownDescription: "Whether every application ID (the consumer group of an application) must be unique in the tenant.",
			Computed:            true,
		},
		"data_classification_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether topics can be given a data classification.",
			Computed:            true,
		},
	}
	for _, s := range tenantBoolSettings {
		attrs[s.attr] = schema.BoolAttribute{MarkdownDescription: s.description, Computed: true}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "The tenant of the login and its settings. Any user or service account of the tenant can read it. To change the settings, use `axual_tenant_settings`.",
		Attributes:          attrs,
	}
}

func (d *tenantDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tenant, err := d.provider.client.GetTenant()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read the tenant of the login: %s", err))
		return
	}
	runtimeSupported, err := tenantRuntimeSupported(d.provider.client)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read the Platform Manager version: %s", err))
		return
	}
	values, diags := tenantSettingsFromAnswer(tenant, runtimeSupported)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	set := func(name string, value interface{}) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(name), value)...)
	}
	set("id", types.StringValue(tenant.Uid))
	set("name", types.StringValue(tenant.Name))
	set("short_name", types.StringValue(tenant.ShortName))
	set("authentication_methods", values.AuthenticationMethods)
	set("history_retention_days", values.HistoryRetentionDays)
	set("onboarding_completed", types.BoolValue(tenant.WizardCompleted))
	set("unique_consumer_group", types.BoolValue(tenant.UniqueConsumerGroup))
	set("data_classification_enabled", types.BoolValue(tenant.DataClassificationEnabled))
	for _, s := range tenantBoolSettings {
		set(s.attr, values.Bools[s.attr])
	}
}
