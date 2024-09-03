package custom_validator

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
