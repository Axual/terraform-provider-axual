package provider

import (
	"encoding/base64"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// putIfSet adds value to m under key, unless it is null, unknown or "".
func putIfSet(m map[string]string, key string, value types.String) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return
	}
	m[key] = value.ValueString()
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
