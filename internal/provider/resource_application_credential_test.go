package provider

import (
	webclient "axual-webclient"
	"testing"
)

func TestDeriveTargetFromTypes(t *testing.T) {
	tests := map[string]struct {
		types  []string
		target string
	}{
		"schema registry basic auth": {[]string{"SCHEMA_REGISTRY_BASIC_AUTH"}, "SCHEMA_REGISTRY"},
		"schema registry oauth":      {[]string{"SCHEMA_REGISTRY_OAUTH"}, "SCHEMA_REGISTRY"},
		"scram":                      {[]string{"SCRAM_SHA_512"}, "KAFKA"},
		"no types":                   {nil, "KAFKA"},
		"schema registry and scram":  {[]string{"SCRAM_SHA_512", "SCHEMA_REGISTRY_OAUTH"}, "SCHEMA_REGISTRY"},
	}
	for name, test := range tests {
		var authTypes []webclient.AuthType
		for _, authType := range test.types {
			authTypes = append(authTypes, webclient.AuthType{Type: authType})
		}
		if got := deriveTargetFromTypes(authTypes); got != test.target {
			t.Errorf("%s: deriveTargetFromTypes = %s, expected %s", name, got, test.target)
		}
	}
}
