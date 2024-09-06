package provider

import (
	webclient "axual-webclient"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &topicBrowsePermissionsResource{}
var _ resource.ResourceWithImportState = &topicBrowsePermissionsResource{}

func NewTopicBrowsePermissionsResource(provider AxualProvider) resource.Resource {
	return &topicBrowsePermissionsResource{
		provider: provider,
	}
}

type topicBrowsePermissionsResource struct {
	provider AxualProvider
}

type topicBrowsePermissionsResourceData struct {
	TopicConfig types.String `tfsdk:"topic_config"`
	Users       types.Set    `tfsdk:"users"`
	Groups      types.Set    `tfsdk:"groups"`
}

func (r *topicBrowsePermissionsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topic_browse_permissions"
}

func (r *topicBrowsePermissionsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "With this resource you can configure who can browse topic's messages in a specified environment. Only works if the Environment's Instance has Granular Stream Browse Permissions turned on. Granular Stream browse permissions are disabled in private environments and in public environments with the authorization issuer set to \"auto\". Read more: https://docs.axual.io/axual/2024.2/self-service/stream-browse.html#controlling-permissions-to-browse-a-stream",
		Attributes: map[string]schema.Attribute{
			"topic_config": schema.StringAttribute{
				MarkdownDescription: "UID of the Topic configuration.",
				Required:            true,
			},
			"users": schema.SetAttribute{
				MarkdownDescription: "Set of users who are given Topic Browse permissions. User can't give Granular Stream Browse Permissions to the user he is logged in as.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"groups": schema.SetAttribute{
				MarkdownDescription: "Set of groups who are given Topic Browse permissions. User can't add a group where he himself is a member, because user can't give himself Granular Stream Browse Permissions.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *topicBrowsePermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data topicBrowsePermissionsResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	permissionRequest, err := createPermissionRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating permission request", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	err = r.provider.client.AddTopicConfigPermissions(data.TopicConfig.ValueString(), *permissionRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error adding browse permissions", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *topicBrowsePermissionsResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
}

func (r *topicBrowsePermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data topicBrowsePermissionsResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	perms, err := r.provider.client.GetTopicConfigPermissions(data.TopicConfig.ValueString(), "browse")
	if err != nil {
		resp.Diagnostics.AddError("Error reading browse permissions", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapPermissionsResponseToData(ctx, &data, perms)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *topicBrowsePermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data topicBrowsePermissionsResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	users := setToStringSlice(data.Users)
	groups := setToStringSlice(data.Groups)

	// If both users and groups are empty, don't proceed with the delete operation
	if len(users) == 0 && len(groups) == 0 {
		// Log the decision not to proceed with deletion
		resp.Diagnostics.AddWarning("No users or groups to delete", "Both users and groups are empty. Skipping deletion.")
		return
	}

	permissionRequest, err := createPermissionRequestFromData(ctx, &data, r)
	if err != nil {
		resp.Diagnostics.AddError("Error creating permission request", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	err = r.provider.client.DeleteTopicConfigPermissions(data.TopicConfig.ValueString(), *permissionRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting browse permissions", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
}
func (r *topicBrowsePermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper to create PermissionRequest object from resource data
func createPermissionRequestFromData(ctx context.Context, data *topicBrowsePermissionsResourceData, r *topicBrowsePermissionsResource) (*webclient.PermissionRequest, error) {
	users := setToStringSlice(data.Users)
	groups := setToStringSlice(data.Groups)

	// Check if both users and groups are empty
	if len(users) == 0 && len(groups) == 0 {
		return nil, fmt.Errorf("either 'users' or 'groups' must be provided")
	}

	return &webclient.PermissionRequest{
		Type:   "browse",
		Users:  users,
		Groups: groups,
	}, nil
}

// Mapping function from API response to Terraform data
func mapPermissionsResponseToData(ctx context.Context, data *topicBrowsePermissionsResourceData, perms []webclient.PermissionResponse) {
	users := []attr.Value{}
	groups := []attr.Value{}

	for _, perm := range perms {
		if perm.Type == "USER" {
			users = append(users, types.StringValue(perm.Uid))
		} else if perm.Type == "GROUP" {
			groups = append(groups, types.StringValue(perm.Uid))
		}
	}

	data.Users, _ = types.SetValue(types.StringType, users)
	data.Groups, _ = types.SetValue(types.StringType, groups)
}

// Utility function to convert a Set to a slice of strings
func setToStringSlice(set types.Set) []string {
	var result []string
	for _, elem := range set.Elements() {
		result = append(result, elem.(types.String).ValueString())
	}
	return result
}
