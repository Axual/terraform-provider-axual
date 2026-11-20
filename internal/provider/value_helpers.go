package provider

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// noSurroundingWhitespace refuses a value that starts or ends with white space. Platform Manager removes
// a space, a tab, a line break or another control character there (Java's String.trim), so the stored
// value differs from the configuration and the apply fails.
var noSurroundingWhitespace validator.String = surroundingWhitespaceValidator{}

type surroundingWhitespaceValidator struct{}

func (v surroundingWhitespaceValidator) Description(_ context.Context) string {
	return "must not start or end with a space, a tab, a line break or other white space"
}

func (v surroundingWhitespaceValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v surroundingWhitespaceValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	javaTrimmed := strings.TrimFunc(value, func(r rune) bool { return r <= ' ' })
	if value != javaTrimmed || value != strings.TrimSpace(value) {
		resp.Diagnostics.AddAttributeError(req.Path, "Value starts or ends with white space",
			"The value "+v.Description(ctx)+": it is removed when the value is stored, and the apply would then fail. "+
				"For a heredoc, which ends with a line break, use trimspace().")
	}
}

// notEmpty refuses "": Platform Manager stores it, or the provider leaves it out of the request, and the
// value is then read back as null, so the apply fails.
var notEmpty validator.String = notEmptyValidator{}

type notEmptyValidator struct{}

func (v notEmptyValidator) Description(_ context.Context) string {
	return "must not be empty: leave it out instead"
}

func (v notEmptyValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v notEmptyValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() && req.ConfigValue.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Empty value", "The value "+v.Description(ctx)+".")
	}
}

// mtlsToAPI sends MTLS as "mTLS", the spelling the Self-Service UI writes and compares with. Platform
// Manager ignores the case.
func mtlsToAPI(value string) string {
	if strings.EqualFold(value, "MTLS") {
		return "mTLS"
	}
	return value
}

// spellingOf returns the value of known that equals value ignoring case, as Platform Manager compares
// them, or value when none does.
func spellingOf(value string, known []string) string {
	for _, k := range known {
		if strings.EqualFold(k, value) {
			return k
		}
	}
	return value
}

// putIfSet adds value to m under key, unless it is null, unknown or "".
func putIfSet(m map[string]string, key string, value types.String) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return
	}
	m[key] = value.ValueString()
}

// nonEmptyPointer returns nil for "", so the field is left out of a request.
func nonEmptyPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// stringPointerOrNil returns nil for a null or unknown value, which a PATCH sends as an explicit null.
func stringPointerOrNil(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	s := value.ValueString()
	return &s
}

// stringMapFromData returns the map's elements, or an empty map for a null or unknown map.
func stringMapFromData(value types.Map) map[string]string {
	out := map[string]string{}
	if value.IsNull() || value.IsUnknown() {
		return out
	}
	for key, element := range value.Elements() {
		if s, ok := element.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
			out[key] = s.ValueString()
		}
	}
	return out
}

// stringMapToData maps an API map to state. An absent map becomes an empty map, not null, so
// import and create agree.
func stringMapToData(m map[string]string) types.Map {
	elements := make(map[string]attr.Value, len(m))
	for key, value := range m {
		elements[key] = types.StringValue(value)
	}
	return types.MapValueMust(types.StringType, elements)
}

// Platform Manager keeps certificates, private keys and truststores base64-encoded, the way the
// Self-Service UI sends them, and parts of it decode them again. The provider takes PEM, so a PEM
// value is encoded; a value that is already base64 is sent as it is.
func pemToAPI(value string) string {
	if strings.Contains(value, "-----BEGIN") {
		return base64.StdEncoding.EncodeToString([]byte(value))
	}
	return value
}

// pemFromAPI turns a stored value back into the form of the configuration: PEM, unless the state
// holds the base64 form.
func pemFromAPI(value string, state types.String) string {
	if !state.IsNull() && !state.IsUnknown() && state.ValueString() != "" && !strings.Contains(state.ValueString(), "-----BEGIN") {
		return value
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil || !strings.Contains(string(decoded), "-----BEGIN") {
		return value
	}
	return string(decoded)
}

// putPEMIfSet is putIfSet for a PEM value, which Platform Manager wants base64-encoded.
func putPEMIfSet(m map[string]string, key string, value types.String) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return
	}
	m[key] = pemToAPI(value.ValueString())
}
