package custom_validator

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NonEmptySetValidator is a custom validator ensuring a set is not empty but can be null.
type NonEmptySetValidator struct{}

// Description returns the description of the validator.
func (v NonEmptySetValidator) Description(_ context.Context) string {
	return "Ensures that the set is not empty but can be null."
}

// MarkdownDescription returns the markdown description of the validator.
func (v NonEmptySetValidator) MarkdownDescription(_ context.Context) string {
	return v.Description(context.Background())
}

// NewNonEmptySetValidator creates a new instance of NonEmptySetValidator.
func NewNonEmptySetValidator() validator.Set {
	return NonEmptySetValidator{}
}

// ValidateSet validates the set ensuring it is not empty if it is not null or unknown.
func (v NonEmptySetValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() {
		return
	}

	if !req.ConfigValue.IsUnknown() && len(req.ConfigValue.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Empty Set",
			"The set must not be empty. It can be null(omitted) or contain elements, but it cannot be empty.",
		)
	}
}

// KsmlApplicationTypeValidator validates that the type field is null when application_type is Ksml.
type KsmlApplicationTypeValidator struct{}

// Description returns the description of the validator.
func (v KsmlApplicationTypeValidator) Description(_ context.Context) string {
	return "Ensures that the type field is null when application_type is Ksml."
}

// MarkdownDescription returns the markdown description of the validator.
func (v KsmlApplicationTypeValidator) MarkdownDescription(_ context.Context) string {
	return v.Description(context.Background())
}

// NewKsmlApplicationTypeValidator creates a new instance of KsmlApplicationTypeValidator.
func NewKsmlApplicationTypeValidator() validator.String {
	return KsmlApplicationTypeValidator{}
}

// ValidateString validates that the type is null when application_type is Ksml.
func (v KsmlApplicationTypeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the type is null or unknown, validation passes
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// Get the application_type value
	var applicationType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("application_type"), &applicationType)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the application_type is unknown, we can't validate yet
	if applicationType.IsUnknown() {
		return
	}

	// If the application_type is Ksml and the type is not null, return error
	if applicationType.ValueString() == "Ksml" && !req.ConfigValue.IsNull() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Type for KSML Application",
			fmt.Sprintf("When application_type is 'Ksml', the type field must be null/omitted. Got: %s", req.ConfigValue.ValueString()),
		)
	}
}
