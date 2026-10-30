package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var _ list.ListResource = &axualListResource{}

// listFilter is one optional string attribute in the `config` block of a `list` block.
type listFilter struct {
	name        string
	description string
}

// filterValues are the values a filter accepts, in any case. The API takes them in upper case.
var filterValues = map[string][]string{
	"application_type": {"Custom", "Connector", "Ksml", "FLINK_SQL"},
	"access_type":      {"PRODUCER", "CONSUMER"},
	"status":           {"PENDING", "APPROVED", "REJECTED", "REVOKED", "CANCELLED"},
}

// listFound is one resource a list resource found: the value of its identity attribute and the
// name shown by `terraform query`.
type listFound struct {
	id          string
	displayName string
	// forbidden: the user may not manage it, so it is left out with a warning.
	forbidden bool
}

// listSpec describes the list resource of one managed resource type.
type listSpec struct {
	// typeName is the managed resource type, for example "axual_topic".
	typeName string
	// attr is the identity attribute, the same one the resource is wrapped with in Resources.
	attr        string
	description string
	newResource func(AxualProvider) resource.Resource
	filters     []listFilter
	// find returns the resources that match the filters. Filters the user did not set are not in
	// the map.
	find func(ctx context.Context, c *webclient.Client, filters map[string]string) ([]listFound, error)
	// skipReason, when set, is called with the state of each found resource. A non-empty reason
	// leaves the resource out, because its state cannot be written as configuration.
	skipReason func(ctx context.Context, state tfsdk.State) string
	// adjust, when set, changes the state of each found resource before it is written as configuration.
	adjust func(ctx context.Context, state *tfsdk.State) diag.Diagnostics
	// adjustWithClient is adjust for changes that need the API.
	adjustWithClient func(ctx context.Context, c *webclient.Client, state *tfsdk.State) diag.Diagnostics
	// ownIdentity is set for a resource with its own identity schema: it isn't wrapped with
	// withIdentity, attr is unused, and the identity of each result comes from its ImportState.
	ownIdentity bool
}

// axualListResource lets `terraform query` find existing resources of one type. It fills each result
// with the resource's own ImportState and Read, then the spec's adjust.
type axualListResource struct {
	provider AxualProvider
	spec     listSpec
}

func newListResource(provider AxualProvider, spec listSpec) list.ListResource {
	return &axualListResource{provider: provider, spec: spec}
}

func (l *axualListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + strings.TrimPrefix(l.spec.typeName, "axual")
}

func (l *axualListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	attrs := map[string]listschema.Attribute{}
	for _, f := range l.spec.filters {
		var validators []validator.String
		if values, ok := filterValues[f.name]; ok {
			validators = append(validators, stringvalidator.OneOfCaseInsensitive(values...))
		}
		attrs[f.name] = listschema.StringAttribute{Optional: true, Description: f.description, MarkdownDescription: f.description, Validators: validators}
	}
	resp.Schema = listschema.Schema{
		Description:         l.spec.description,
		MarkdownDescription: l.spec.description,
		Attributes:          attrs,
	}
}

func (l *axualListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	if l.provider.client == nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Unconfigured provider", "The Axual provider must be configured before a list resource can be used."),
		})
		return
	}

	filters := map[string]string{}
	var diags diag.Diagnostics
	for _, f := range l.spec.filters {
		var v types.String
		diags.Append(req.Config.GetAttribute(ctx, path.Root(f.name), &v)...)
		if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
			filters[f.name] = v.ValueString()
		}
	}
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	found, err := l.spec.find(ctx, l.provider.client, filters)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic(fmt.Sprintf("Unable to list %s", l.spec.typeName), err.Error()),
		})
		return
	}
	// The API does not always return results in the same order. A stable order keeps the generated
	// resource names (`<list>_0`, `<list>_1`, ...) the same from one export to the next.
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].displayName != found[j].displayName {
			return found[i].displayName < found[j].displayName
		}
		return found[i].id < found[j].id
	})
	var forbidden []string
	found = slices.DeleteFunc(found, func(f listFound) bool {
		if f.forbidden {
			forbidden = append(forbidden, f.displayName)
		}
		return f.forbidden
	})

	stream.Results = func(push func(list.ListResult) bool) {
		if len(forbidden) > 0 {
			var skipped list.ListResult
			skipped.Diagnostics.AddWarning(
				fmt.Sprintf("Skipped %d %s", len(forbidden), l.spec.typeName),
				fmt.Sprintf("You may not manage them, so no configuration is generated: %s.", shortList(forbidden, 10)),
			)
			if !push(skipped) {
				return
			}
		}
		// Terraform stops reading at the limit, so warn before the results, not after them.
		if req.Limit > 0 && int64(len(found)) > req.Limit {
			var cut list.ListResult
			cut.Diagnostics.AddWarning(
				fmt.Sprintf("Found %d %s, returning the first %d", len(found), l.spec.typeName, req.Limit),
				fmt.Sprintf("The list block stops at its limit of %d. Set `limit` in the list block to %d or more to get all of them; some can still be skipped with a warning.", req.Limit, len(found)),
			)
			if !push(cut) {
				return
			}
		}
		pushed := int64(0)
		for _, f := range found {
			if req.Limit > 0 && pushed >= req.Limit {
				return
			}
			result := req.NewListResult(ctx)
			result.DisplayName = f.displayName
			if l.spec.ownIdentity {
				l.importIdentity(ctx, req, f.id, &result)
			} else {
				result.Diagnostics.Append(result.Identity.SetAttribute(ctx, path.Root(l.spec.attr), f.id)...)
			}
			if req.IncludeResource && !result.Diagnostics.HasError() {
				var fillDiags diag.Diagnostics
				l.fillResource(ctx, req, f.id, &result, &fillDiags)
				if fillDiags.HasError() {
					// Skip with a warning, so one bad resource does not stop the export.
					var skipped list.ListResult
					skipped.Diagnostics.AddWarning(
						fmt.Sprintf("Skipped %s %s", l.spec.typeName, f.id),
						fmt.Sprintf("No configuration is generated for %s (%s). %s", f.displayName, f.id, errorSummary(fillDiags)),
					)
					if !push(skipped) {
						return
					}
					continue
				}
				result.Diagnostics.Append(fillDiags...)
			}
			if !push(result) {
				return
			}
			pushed++
		}
	}
}

// fillResource sets result.Resource to the state Terraform would have after importing id.
func (l *axualListResource) fillResource(ctx context.Context, req list.ListRequest, id string, result *list.ListResult, diags *diag.Diagnostics) {
	r := l.resource()

	emptyState := tfsdk.State{
		Schema: req.ResourceSchema,
		Raw:    tftypes.NewValue(req.ResourceSchema.Type().TerraformType(ctx), nil),
	}
	emptyIdentity := &tfsdk.ResourceIdentity{
		Schema: req.ResourceIdentitySchema,
		Raw:    tftypes.NewValue(req.ResourceIdentitySchema.Type().TerraformType(ctx), nil),
	}

	importResp := resource.ImportStateResponse{State: emptyState, Identity: emptyIdentity}
	r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &importResp)
	diags.Append(importResp.Diagnostics...)
	if diags.HasError() {
		return
	}

	readResp := resource.ReadResponse{State: importResp.State, Identity: importResp.Identity}
	r.Read(ctx, resource.ReadRequest{State: importResp.State, Identity: importResp.Identity}, &readResp)
	diags.Append(readResp.Diagnostics...)
	if diags.HasError() {
		return
	}
	if readResp.State.Raw.IsNull() {
		diags.AddError("Resource not found", fmt.Sprintf("%s %s was listed but could not be read; it may have been deleted.", l.spec.typeName, id))
		return
	}
	if l.spec.skipReason != nil {
		if reason := l.spec.skipReason(ctx, readResp.State); reason != "" {
			diags.AddError("Cannot be configured", reason)
			return
		}
	}
	if l.spec.adjust != nil {
		if diags.Append(l.spec.adjust(ctx, &readResp.State)...); diags.HasError() {
			return
		}
	}
	if l.spec.adjustWithClient != nil {
		if diags.Append(l.spec.adjustWithClient(ctx, l.provider.client, &readResp.State)...); diags.HasError() {
			return
		}
	}
	result.Resource.Raw = readResp.State.Raw
}

// errorSummary joins the summaries and details of the error diagnostics.
func errorSummary(diags diag.Diagnostics) string {
	var parts []string
	for _, d := range diags.Errors() {
		parts = append(parts, d.Summary()+": "+d.Detail())
	}
	return strings.Join(parts, "; ")
}

// shortList joins the first n names and says how many more there are.
func shortList(names []string, n int) string {
	if len(names) <= n {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s, and %d more", strings.Join(names[:n], ", "), len(names)-n)
}

// resource returns the managed resource of the spec, with the identity Terraform uses for it.
func (l *axualListResource) resource() resource.ResourceWithImportState {
	if l.spec.ownIdentity {
		return l.spec.newResource(l.provider).(resource.ResourceWithImportState)
	}
	return withIdentity(l.spec.newResource(l.provider), l.spec.attr).(resource.ResourceWithImportState)
}

// importIdentity sets the identity of a result from the resource's own ImportState.
func (l *axualListResource) importIdentity(ctx context.Context, req list.ListRequest, id string, result *list.ListResult) {
	importResp := resource.ImportStateResponse{
		State: tfsdk.State{Schema: req.ResourceSchema, Raw: tftypes.NewValue(req.ResourceSchema.Type().TerraformType(ctx), nil)},
		Identity: &tfsdk.ResourceIdentity{Schema: req.ResourceIdentitySchema,
			Raw: tftypes.NewValue(req.ResourceIdentitySchema.Type().TerraformType(ctx), nil)},
	}
	l.resource().ImportState(ctx, resource.ImportStateRequest{ID: id}, &importResp)
	result.Diagnostics.Append(importResp.Diagnostics...)
	if !result.Diagnostics.HasError() {
		result.Identity.Raw = importResp.Identity.Raw
	}
}
