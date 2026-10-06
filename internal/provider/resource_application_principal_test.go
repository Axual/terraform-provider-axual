package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	certA = "-----BEGIN CERTIFICATE-----\nA\n-----END CERTIFICATE-----"
	certB = "-----BEGIN CERTIFICATE-----\nB\n-----END CERTIFICATE-----"
	keyA  = "-----BEGIN PRIVATE KEY-----\nA\n-----END PRIVATE KEY-----"
)

// principalValues builds a principal object from the resource schema. A value of nil is null, and
// tftypes.UnknownValue is unknown.
func principalValues(t *testing.T, principal, privateKey any) (tftypes.Value, resource.SchemaResponse) {
	t.Helper()
	var schemaResp resource.SchemaResponse
	(&applicationPrincipalResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, attrType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	values["principal"] = tftypes.NewValue(tftypes.String, principal)
	values["private_key"] = tftypes.NewValue(tftypes.String, privateKey)
	return tftypes.NewValue(objectType, values), schemaResp
}

func TestIsCertificateChanging(t *testing.T) {
	tests := []struct {
		name                     string
		statePrincipal, stateKey any
		planPrincipal, planKey   any
		want                     bool
	}{
		{name: "same certificate", statePrincipal: certA, planPrincipal: certA, want: false},
		{name: "new certificate", statePrincipal: certA, planPrincipal: certB, want: true},
		{name: "certificate only known at apply", statePrincipal: certA, planPrincipal: tftypes.UnknownValue, want: true},
		{name: "private key only known at apply", statePrincipal: certA, stateKey: keyA, planPrincipal: certA, planKey: tftypes.UnknownValue, want: true},
		{name: "Custom app without a private key", statePrincipal: certA, planPrincipal: certA, want: false},
		{name: "imported principal adopts its private key", statePrincipal: certA, planPrincipal: certA, planKey: keyA, want: false},
		{name: "trailing newline only", statePrincipal: certA, planPrincipal: certA + "\n", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateRaw, schemaResp := principalValues(t, tt.statePrincipal, tt.stateKey)
			planRaw, _ := principalValues(t, tt.planPrincipal, tt.planKey)
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}
			plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}
			if got := isCertificateChanging(context.Background(), plan, state); got != tt.want {
				t.Errorf("isCertificateChanging() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCertificateChangingOnCreate(t *testing.T) {
	planRaw, schemaResp := principalValues(t, tftypes.UnknownValue, nil)
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), nil)}
	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}
	if isCertificateChanging(context.Background(), plan, state) {
		t.Error("a create is not a certificate change")
	}
}
