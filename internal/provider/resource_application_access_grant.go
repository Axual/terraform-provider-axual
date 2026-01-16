package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"time"

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

var _ resource.Resource = &applicationAccessGrantResource{}
var _ resource.ResourceWithImportState = &applicationAccessGrantResource{}

func NewApplicationAccessGrantResource(provider AxualProvider) resource.Resource {
	return &applicationAccessGrantResource{
		provider: provider,
	}
}

type applicationAccessGrantResource struct {
	provider AxualProvider
}
type applicationAccessGrantData struct {
	Id            types.String `tfsdk:"id"`
	ApplicationId types.String `tfsdk:"application"`
	TopicId       types.String `tfsdk:"topic"`
	EnvironmentId types.String `tfsdk:"environment"`
	Status        types.String `tfsdk:"status"`
	AccessType    types.String `tfsdk:"access_type"`
}

func (r *applicationAccessGrantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_access_grant"
}

func (r *applicationAccessGrantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Application Access Grant resource. Purpose of a grant is to request access to a topic in an environment. Read more: https://docs.axual.io/axual/2025.3/self-service/application-management.html#requesting-topic-access",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Application Access Grant Unique Identifier",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status of Application Access Grant",
				Computed:            true,
			},
			"application": schema.StringAttribute{
				MarkdownDescription: "Application Unique Identifier",
				Required:            true,
			},
			"topic": schema.StringAttribute{
				MarkdownDescription: "Topic Unique Identifier",
				Required:            true,
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "Environment Unique Identifier",
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

func (r *applicationAccessGrantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data applicationAccessGrantData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrantRequestData := webclient.ApplicationAccessGrantRequest{
		EnvironmentId: data.EnvironmentId.ValueString(),
		StreamId:      data.TopicId.ValueString(),
		ApplicationId: data.ApplicationId.ValueString(),
		AccessType:    data.AccessType.ValueString(),
	}

	ApplicationAccessGrant, err := r.provider.client.CreateApplicationAccessGrant(applicationAccessGrantRequestData)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	data.Id = types.StringValue(ApplicationAccessGrant.Uid)
	data.Status = types.StringValue(ApplicationAccessGrant.Status)
	data.TopicId = types.StringValue(data.TopicId.ValueString())
	data.EnvironmentId = types.StringValue(ApplicationAccessGrant.Environment.Id)
	data.ApplicationId = types.StringValue(data.ApplicationId.ValueString())

	tflog.Info(ctx, "Saving Application Access Grant resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *applicationAccessGrantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data applicationAccessGrantData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Change to use GET /application_access_grants/{id} once the API returns application, topic, and access_type fields.
	// Currently using search endpoint as a workaround because GET by ID doesn't return these fields.
	tflog.Info(ctx, fmt.Sprintf("Reading Application Access Grant via search endpoint. Id: %s", data.Id.ValueString()))
	applicationAccessGrant, err := r.findGrantById(ctx, data.Id.ValueString())
	if err != nil {
		if errors.Is(err, errGrantNotFound) {
			tflog.Warn(ctx, fmt.Sprintf("Application Access Grant not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	tflog.Info(ctx, "mapping the resource")
	data.Id = types.StringValue(applicationAccessGrant.Uid)
	data.Status = types.StringValue(applicationAccessGrant.Status)
	data.ApplicationId = types.StringValue(applicationAccessGrant.ApplicationUid)
	data.TopicId = types.StringValue(applicationAccessGrant.StreamUid)
	data.EnvironmentId = types.StringValue(applicationAccessGrant.EnvironmentUid)
	data.AccessType = types.StringValue(applicationAccessGrant.AccessType)

	tflog.Info(ctx, "Saving Application Access Grant resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// grantSearchResult holds the extracted grant data from search results
type grantSearchResult struct {
	Uid            string
	Status         string
	ApplicationUid string
	StreamUid      string
	EnvironmentUid string
	AccessType     string
}

// errGrantNotFound is returned when a grant cannot be found
var errGrantNotFound = errors.New("grant not found")

// findGrantById searches for a grant by its UID using the search endpoint.
// TODO: Replace with GET /application_access_grants/{id} once the API returns application, topic, and access_type fields.
// This is an anti-pattern (fetching all to find one), but necessary as a workaround until the API is updated.
func (r *applicationAccessGrantResource) findGrantById(ctx context.Context, grantId string) (*grantSearchResult, error) {
	// Note: Not filtering by status - the API returns all grants when statuses is not provided.
	// Possible grant statuses: PENDING, APPROVED, REVOKED, REJECTED, CANCELLED
	searchAttrs := webclient.ApplicationAccessGrantAttributes{
		Size: 9999,
	}
	result, err := r.provider.client.GetApplicationAccessGrantsByAttributes(searchAttrs)
	if err != nil {
		return nil, err
	}

	for _, grant := range result.Embedded.ApplicationAccessGrantResponses {
		if grant.Uid == grantId {
			return &grantSearchResult{
				Uid:            grant.Uid,
				Status:         grant.Status,
				ApplicationUid: grant.Embedded.Application.Uid,
				StreamUid:      grant.Embedded.Stream.Uid,
				EnvironmentUid: grant.Embedded.Environment.Uid,
				AccessType:     grant.AccessType,
			}, nil
		}
	}

	return nil, errGrantNotFound
}

func (r *applicationAccessGrantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Application Access Grant cannot be updated",
		`The Axual API does not support updating grant attributes. To change access_type, application, topic, or environment:

1. Delete the axual_application_access_grant_approval resource (this revokes the grant)
2. Delete the axual_application_access_grant resource
3. Recreate the axual_application_access_grant with new attributes
4. Recreate the axual_application_access_grant_approval

For detailed instructions, see: https://registry.terraform.io/providers/Axual/axual/latest/docs/guides/manage-application-access-to-topics`)
}

func (r *applicationAccessGrantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data applicationAccessGrantData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	applicationAccessGrant, err := r.provider.client.GetApplicationAccessGrant(data.Id.ValueString())
	if err != nil {
		// If grant not found, it's already deleted - success
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Info(ctx, fmt.Sprintf("Grant already deleted. Id: %s", data.Id.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Failed to get Application Access Grant", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	// Terminal states - grant is already "destroyed", just remove from state
	// We check this BEFORE attempting any API calls to avoid "invalid state" errors
	// This also handles the case where approval resource revoked the grant at the same time.
	if applicationAccessGrant.Status == "Revoked" ||
		applicationAccessGrant.Status == "Rejected" ||
		applicationAccessGrant.Status == "Cancelled" {
		tflog.Info(ctx, fmt.Sprintf("Grant is in terminal state (%s), removing from state. Id: %s",
			applicationAccessGrant.Status, data.Id.ValueString()))
		return
	}

	// Pending grants - can be cancelled
	if applicationAccessGrant.Links.Cancel.Href != "" {
		tflog.Info(ctx, fmt.Sprintf("Cancelling pending grant. Id: %s", data.Id.ValueString()))
		// Retry logic for cancelling the grant to give time for Kafka to propagate changes
		err1 := Retry(3, 3*time.Second, func() error {
			return r.provider.client.CancelGrant(data.Id.ValueString())
		})
		if err1 != nil {
			resp.Diagnostics.AddError("Unable to cancel Application Access Grant", fmt.Sprintf("Error message after retries: %s", err1))
			return
		}
		return
	}

	// Approved grants - revoke first, then remove from state
	// This enables terraform destroy to work after import, when the dependency
	// between grant and approval resources is lost and they're destroyed in parallel
	if applicationAccessGrant.Status == "Approved" && applicationAccessGrant.Links.Revoke.Href != "" {
		tflog.Info(ctx, fmt.Sprintf("Revoking approved grant before deletion. Id: %s", data.Id.ValueString()))
		err := r.provider.client.RevokeOrDenyGrant(data.Id.ValueString(), "Revoked during terraform destroy")
		if err != nil {
			// Handle race condition: approval resource may have been revoked at the same time
			// The API is not idempotent - revoking an already-revoked grant throws an error
			// Re-fetch grant to check if it's now in a terminal state
			tflog.Warn(ctx, fmt.Sprintf("Revoke failed, checking if grant was revoked by another process. Id: %s, Error: %s",
				data.Id.ValueString(), err.Error()))

			updatedGrant, fetchErr := r.provider.client.GetApplicationAccessGrant(data.Id.ValueString())
			if fetchErr != nil {
				if errors.Is(fetchErr, webclient.NotFoundError) {
					tflog.Info(ctx, fmt.Sprintf("Grant was deleted by another process. Id: %s", data.Id.ValueString()))
					return
				}
				resp.Diagnostics.AddError("Failed to revoke grant during deletion",
					fmt.Sprintf("Original error: %s", err.Error()))
				return
			}

			// If grant is now in terminal state, someone else handled it - success
			if updatedGrant.Status == "Revoked" ||
				updatedGrant.Status == "Rejected" ||
				updatedGrant.Status == "Cancelled" {
				tflog.Info(ctx, fmt.Sprintf("Grant was revoked by another process (status: %s). Id: %s",
					updatedGrant.Status, data.Id.ValueString()))
				return
			}

			// Still not terminal - report the original error
			resp.Diagnostics.AddError("Failed to revoke grant during deletion",
				fmt.Sprintf("Error message: %s", err.Error()))
			return
		}
		tflog.Info(ctx, fmt.Sprintf("Grant revoked successfully, removing from state. Id: %s", data.Id.ValueString()))
		return
	}

	// Unexpected state - shouldn't reach here
	resp.Diagnostics.AddError(
		"Application Access Grant cannot be deleted",
		fmt.Sprintf("Grant is in unexpected state: %s", applicationAccessGrant.Status))
}

func (r *applicationAccessGrantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
