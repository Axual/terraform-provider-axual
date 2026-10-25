package provider

import (
	webclient "axual-webclient"
	custom_validator "axual.com/terraform-provider-axual/internal/custom-validator"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"regexp"
	"strings"
)

var _ resource.Resource = &groupResource{}
var _ resource.ResourceWithImportState = &groupResource{}
var _ resource.ResourceWithModifyPlan = &groupResource{}

type groupResourceType struct{}

func NewGroupResource(provider AxualProvider) resource.Resource {
	return &groupResource{
		provider: provider,
	}
}

type groupResource struct {
	provider AxualProvider
}

// iamReferenceFormat is the value of the group claim in the token. The API gateway joins the claim values
// with commas, so a comma can never match. A space at the start or the end, and a control character
// (a tab or a new line), can never match either, so the provider refuses them.
var iamReferenceFormat = regexp.MustCompile(`^[^\s\p{Z}\p{Cc},](?:[^,\p{Cc}]*[^\s\p{Z}\p{Cc},])?$`)

type groupResourceData struct {
	Name         types.String `tfsdk:"name"`
	EmailAddress types.String `tfsdk:"email_address"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	Members      types.Set    `tfsdk:"members"`
	Managers     types.Set    `tfsdk:"managers"`
	IamReference types.String `tfsdk:"iam_reference"`
	Id           types.String `tfsdk:"id"`
}

func (r *groupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Group resource. Read more: https://docs.axual.io/axual/2026.1/self-service/user-group-management.html#groups",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Group's name",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 80),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._ -]*$`), "can only contain letters, numbers, dots, dashes and underscores, but cannot begin with an underscore, dot or dash"),
				},
			},
			"email_address": schema.StringAttribute{
				MarkdownDescription: "Group's email address",
				Optional:            true,
			},
			"phone_number": schema.StringAttribute{
				MarkdownDescription: "Group's phone number",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 16),
				},
			},
			"members": schema.SetAttribute{
				MarkdownDescription: "Group's members. Each entry is the uid of a user or of a service account. Adding a service account requires Tenant Admin; once it is a member, a group manager may promote or remove it. Cannot be set when `iam_reference` is set.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"managers": schema.SetAttribute{
				MarkdownDescription: "A Group Manager can edit this group, including adding or removing users and other group managers. Each entry is the uid of a user or of a service account, and must also be listed in `members`. Cannot be set when `iam_reference` is set. Read more: https://docs.axual.io/axual/2026.1/self-service/user-group-management.html#making-a-group-member-manager-of-the-group",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					custom_validator.NewNonEmptySetValidator(),
				},
			},
			"iam_reference": schema.StringAttribute{
				MarkdownDescription: "The group's value in your identity system, as it is in the token (for example an Azure AD group object id). Setting it makes the group IAM-managed: the identity system decides who is a member. Removing it makes the group Axual-managed again. Must be unique in the tenant. See the guide \"IAM-managed groups\".",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(iamReferenceFormat,
						"must not start or end with a space, and must not contain a comma or a control character"),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Group's unique identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data groupResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupRequest, err := createGroupRequestFromData(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CREATE request struct for group resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	group, err := r.provider.client.CreateGroup(groupRequest)
	if err != nil {
		resp.Diagnostics.AddError("CREATE request error for group resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}

	mapGroupResponseToData(ctx, &data, group)
	tflog.Trace(ctx, "created a resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data groupResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.provider.client.GetGroup(data.Id.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Warn(ctx, fmt.Sprintf("Group not found. Id: %s", data.Id.ValueString()))
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read group, got error: %s", err))
		}
		return
	}

	tflog.Info(ctx, "mapping the resource")
	mapGroupResponseToData(ctx, &data, group)

	tflog.Info(ctx, "saving the resource to state")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// ModifyPlan refuses members or managers on an IAM-managed group, because the provider does not send
// them, and warns when removing iam_reference makes the group Axual-managed again.
func (r *groupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var data groupResourceData
	if diags := req.Plan.Get(ctx, &data); diags.HasError() {
		return
	}
	if isIamManaged(data) && (!data.Members.IsNull() || !data.Managers.IsNull()) {
		resp.Diagnostics.AddAttributeError(path.Root("iam_reference"),
			"Members and managers not supported",
			"This group has an iam_reference, so the IAM system decides who is a member. Remove members and managers from the configuration.")
	}

	// Leaving out iam_reference on a group that has one makes it Axual-managed again, and the members
	// Platform Manager has stored for the group count again. Say so in the plan.
	var state groupResourceData
	if req.State.Raw.IsNull() || req.State.Get(ctx, &state).HasError() {
		return
	}
	if state.IamReference.IsNull() && !data.IamReference.IsNull() && (!state.Members.IsNull() || !state.Managers.IsNull()) {
		resp.Diagnostics.AddAttributeWarning(path.Root("iam_reference"),
			"Stored members and managers stay in Platform Manager",
			"Linking does not remove the members and managers stored for this group. They still count for service accounts and for creating applications, topics and environments for the group. This plan shows them as removed from the Terraform state only. Do not empty them just to be clean: today Platform Manager lets only a stored member (or a Tenant Admin or Application Admin) create resources for the group.")
	}
	if !state.IamReference.IsNull() && data.IamReference.IsNull() {
		resp.Diagnostics.AddAttributeWarning(path.Root("iam_reference"),
			"The group becomes Axual-managed again",
			fmt.Sprintf("The group is linked to IAM group %q, and iam_reference is not in the configuration. Applying makes the group Axual-managed. Its stored members and managers count again, and the next plan shows them: add them to members and managers to keep them.", state.IamReference.ValueString()))
	}
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state groupResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Link or unlink the IAM group first: Platform Manager checks the tenant settings before it
	// changes anything, so a refused link leaves the group as it was. Changing the reference of an
	// IAM-managed group is a normal update.
	id := data.Id.ValueString()
	switch {
	case state.IamReference.IsNull() && !data.IamReference.IsNull():
		if _, err := r.provider.client.ExternalizeGroup(id, data.IamReference.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to link the group to IAM group %q, got error: %s", data.IamReference.ValueString(), err))
			return
		}
	case !state.IamReference.IsNull() && data.IamReference.IsNull():
		if _, err := r.provider.client.InternalizeGroup(id); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to make the group Axual-managed again, got error: %s", err))
			return
		}
	}

	groupRequest, err := createGroupRequestFromData(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error creating UPDATE request struct for group resource", fmt.Sprintf("Error message: %s", err.Error()))
		return
	}
	// The state of an IAM-managed group has no members, so a plan cannot show what is stored. When the
	// group is unlinked, do not clear a list that the configuration does not have: keep it, and let
	// the next plan show it. Otherwise this apply would remove it, unseen.
	unlinked := !state.IamReference.IsNull() && data.IamReference.IsNull()
	keepMembers := unlinked && data.Members.IsNull()
	keepManagers := unlinked && data.Managers.IsNull()
	groupRequest.SkipMembers = groupRequest.SkipMembers || keepMembers
	groupRequest.SkipManagers = groupRequest.SkipManagers || keepManagers
	group, err := r.provider.client.UpdateGroup(id, groupRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update group, got error: %s", err))
		return
	}

	mapGroupResponseToData(ctx, &data, group)
	if keepMembers {
		data.Members = types.SetNull(types.StringType)
	}
	if keepManagers {
		data.Managers = types.SetNull(types.StringType)
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data groupResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("delete request for group %q", data.Id.ValueString()))

	err := ignoreGone(r.provider.client.DeleteGroup(data.Id.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete group, got error: %s", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("delete group successful for group: %q", data.Id.ValueString()))
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importByIDOrKey(ctx, req, resp, importByNamePrefix, func(key string) (string, error) {
		return findGroupIDByName(r.provider.client, key)
	})
}

func mapGroupResponseToData(ctx context.Context, data *groupResourceData, group *webclient.GroupResponse) {
	// Initialize diagnostics variable
	var diags diag.Diagnostics

	// mandatory fields first
	tflog.Info(ctx, "mapping response to data")
	data.Id = types.StringValue(group.Uid)
	data.Name = types.StringValue(group.Name)

	iamReference := strings.TrimSpace(group.IamReference)
	if iamReference == "" {
		data.IamReference = types.StringNull()
	} else {
		data.IamReference = types.StringValue(group.IamReference)
	}

	// The provider does not manage the lists Platform Manager stores for an IAM-managed group, so
	// they stay null (the configuration cannot have them).
	if iamReference != "" {
		mapGroupContactToData(ctx, data, group)
		return
	}

	// Handle members
	if group.Embedded.Members == nil || len(group.Embedded.Members) == 0 {
		data.Members = types.SetNull(types.StringType)
	} else {
		memberSet := make([]attr.Value, len(group.Embedded.Members))
		for i, member := range group.Embedded.Members {
			memberSet[i] = types.StringValue(member.Uid)
		}
		data.Members, diags = types.SetValue(types.StringType, memberSet)
		if diags.HasError() {
			tflog.Error(ctx, "Error creating members set")
		}
	}

	// Handle managers
	if group.Embedded.Managers == nil || len(group.Embedded.Managers) == 0 {
		data.Managers = types.SetNull(types.StringType)
	} else {
		managerSet := make([]attr.Value, len(group.Embedded.Managers))
		for i, manager := range group.Embedded.Managers {
			managerSet[i] = types.StringValue(manager.Uid)
		}
		data.Managers, diags = types.SetValue(types.StringType, managerSet)
		if diags.HasError() {
			tflog.Error(ctx, "Error creating managers set")
		}
	}

	mapGroupContactToData(ctx, data, group)
}

func mapGroupContactToData(ctx context.Context, data *groupResourceData, group *webclient.GroupResponse) {
	// optional fields
	if group.EmailAddress.Email == "" {
		data.EmailAddress = types.StringNull()
	} else {
		tflog.Info(ctx, fmt.Sprintf("email is %s", group.EmailAddress.Email))
		data.EmailAddress = types.StringValue(group.EmailAddress.Email)
	}
	if group.PhoneNumber == nil {
		data.PhoneNumber = types.StringNull()
	} else {
		data.PhoneNumber = types.StringValue(group.PhoneNumber.(string))
	}
}

func isIamManaged(data groupResourceData) bool {
	return !data.IamReference.IsNull() && !data.IamReference.IsUnknown()
}

func createGroupRequestFromData(ctx context.Context, data *groupResourceData) (webclient.GroupRequest, error) {
	// Create members list
	members := []string{}
	if !data.Members.IsNull() {
		var memberUIDs []string
		diags := data.Members.ElementsAs(ctx, &memberUIDs, false)
		if diags.HasError() {
			return webclient.GroupRequest{}, fmt.Errorf("failed to extract members: %v", diags)
		}

		// A bare uid: Platform Manager resolves whether it names a person or a service
		// account. A typed URI would have to claim one, and the uid alone does not say which.
		members = append(members, memberUIDs...)
	}

	tflog.Info(ctx, fmt.Sprintf("Desired members list size %d", len(data.Members.Elements())))
	tflog.Info(ctx, fmt.Sprintf("Creating new members list of size %d", len(members)))

	// Create managers list
	managers := []string{}
	if !data.Managers.IsNull() {
		var managerUIDs []string
		diags := data.Managers.ElementsAs(ctx, &managerUIDs, false)
		if diags.HasError() {
			return webclient.GroupRequest{}, fmt.Errorf("failed to extract managers: %v", diags)
		}

		// A manager is a member with an extra role, not a group, so it is named the same way.
		managers = append(managers, managerUIDs...)
	}

	groupRequest := webclient.GroupRequest{
		Name:     data.Name.ValueString(),
		Members:  members,
		Managers: managers,
	}

	if isIamManaged(*data) {
		groupRequest.IamReference = data.IamReference.ValueString()
		groupRequest.SkipMembers = true
		groupRequest.SkipManagers = true
	}

	// Optional fields
	if !data.PhoneNumber.IsNull() {
		tflog.Info(ctx, "phone number is not null")
		groupRequest.PhoneNumber = data.PhoneNumber.ValueString()
	}

	if !data.EmailAddress.IsNull() {
		tflog.Info(ctx, "email is not null")
		groupRequest.EmailAddress = data.EmailAddress.ValueString()
	}

	tflog.Info(ctx, fmt.Sprintf("group request %+v", groupRequest))
	return groupRequest, nil
}
