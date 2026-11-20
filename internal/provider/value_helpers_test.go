package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Platform Manager removes the characters up to U+0020 at both ends (Java's String.trim), and the provider
// reads connect_urls back with Go's strings.TrimSpace, so both kinds of white space are refused there.
func TestNoSurroundingWhitespace(t *testing.T) {
	for value, wantError := range map[string]bool{
		"Amsterdam": false, "two words": false, "line\nbreak": false, "": false,
		"ends with a space ": true, " starts with a space": true, "heredoc\n": true, "\ttab": true,
		"crlf\r\n": true, "vertical tab\v": true, "control\x01": true, "no-break space ": true,
	} {
		resp := validator.StringResponse{}
		noSurroundingWhitespace.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("description"), ConfigValue: types.StringValue(value)}, &resp)
		if resp.Diagnostics.HasError() != wantError {
			t.Errorf("%q: error = %v, expected %v", value, resp.Diagnostics.HasError(), wantError)
		}
	}
}

// The attributes Platform Manager trims, and connect_urls, have the validator.
func TestTrimmedAttributesRefuseSurroundingWhitespace(t *testing.T) {
	ctx := context.Background()
	want := noSurroundingWhitespace.Description(ctx)
	for name, test := range map[string]struct {
		resource   resource.Resource
		attributes []string
	}{
		"cluster":  {&clusterResource{}, []string{"description", "location"}},
		"instance": {&instanceResource{}, []string{"name", "description", "connect_urls"}},
	} {
		resp := resource.SchemaResponse{}
		test.resource.Schema(ctx, resource.SchemaRequest{}, &resp)
		for _, attribute := range test.attributes {
			var descriptions []string
			switch a := resp.Schema.Attributes[attribute].(type) {
			case schema.StringAttribute:
				for _, v := range a.Validators {
					descriptions = append(descriptions, v.Description(ctx))
				}
			case schema.ListAttribute:
				for _, v := range a.Validators {
					descriptions = append(descriptions, v.Description(ctx))
				}
			}
			if !strings.Contains(strings.Join(descriptions, "|"), want) {
				t.Errorf("%s %s has no white-space validator: %v", name, attribute, descriptions)
			}
		}
	}
}
