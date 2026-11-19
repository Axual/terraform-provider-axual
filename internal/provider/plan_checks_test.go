package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Platform Manager supports OAuth only for an APICURIO registry, on every version, and reads it back as
// false for CONFLUENT.
func TestInstanceClusterConfigRefusesOAuthWithoutApicurio(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		registryType string
		oauth        types.Bool
		wantError    bool
	}{
		"CONFLUENT with OAuth":    {"CONFLUENT", types.BoolValue(true), true},
		"APICURIO with OAuth":     {"APICURIO", types.BoolValue(true), false},
		"CONFLUENT without OAuth": {"CONFLUENT", types.BoolValue(false), false},
		"OAuth not known yet":     {"CONFLUENT", types.BoolUnknown(), false},
	} {
		t.Run(name, func(t *testing.T) {
			data := registryWith(t, tc.registryType, "", "", nil, nil)
			var registry schemaRegistryData
			_ = data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})
			registry.OauthEnabled = tc.oauth
			data.SchemaRegistry, _ = types.ObjectValueFrom(ctx, schemaRegistryAttrTypes(), registry)
			r := &instanceClusterResource{}
			config := stateFor(t, r, data)
			resp := resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw}}, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Errorf("error = %v, expected %v: %v", resp.Diagnostics.HasError(), tc.wantError, resp.Diagnostics)
			}
		})
	}
}

// Platform Manager refuses keycloak for a CONFLUENT registry and with No-Auth, with the same rule on
// 15.0.9, 16.0.3 and 16.1.0.
func TestInstanceClusterConfigRefusesKeycloakWithoutApicurioAuthentication(t *testing.T) {
	ctx := context.Background()
	keycloak := types.ObjectValueMust(schemaRegistryKeycloakAttrTypes(), map[string]attr.Value{
		"url": types.StringValue("https://keycloak"), "master_realm": types.StringValue("master"),
		"admin_username": types.StringValue("admin"), "admin_password": types.StringValue("secret"),
		"admin_client_id": types.StringValue("admin-cli"), "apicurio_realm": types.StringValue("apicurio"),
	})
	for name, tc := range map[string]struct {
		registryType   string
		authentication types.String
		keycloak       types.Object
		wantError      bool
	}{
		"CONFLUENT with keycloak":      {"CONFLUENT", types.StringValue("Basic-Auth"), keycloak, true},
		"No-Auth with keycloak":        {"APICURIO", types.StringValue("No-Auth"), keycloak, true},
		"Basic-Auth with keycloak":     {"APICURIO", types.StringValue("Basic-Auth"), keycloak, false},
		"No-Auth without keycloak":     {"APICURIO", types.StringValue("No-Auth"), types.ObjectNull(schemaRegistryKeycloakAttrTypes()), false},
		"keycloak not known yet":       {"CONFLUENT", types.StringValue("No-Auth"), types.ObjectUnknown(schemaRegistryKeycloakAttrTypes()), false},
		"authentication not known yet": {"APICURIO", types.StringUnknown(), keycloak, false},
	} {
		t.Run(name, func(t *testing.T) {
			data := registryWith(t, tc.registryType, "", "", nil, nil)
			var registry schemaRegistryData
			_ = data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})
			registry.Authentication, registry.Keycloak = tc.authentication, tc.keycloak
			data.SchemaRegistry, _ = types.ObjectValueFrom(ctx, schemaRegistryAttrTypes(), registry)
			r := &instanceClusterResource{}
			config := stateFor(t, r, data)
			resp := resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw}}, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Errorf("error = %v, expected %v: %v", resp.Diagnostics.HasError(), tc.wantError, resp.Diagnostics)
			}
		})
	}
}

// Platform Manager removes the CA certificates of an Instance without MUTUAL_TLS. The plan refuses that
// only when authentication_methods is set; left out, the methods are not known.
func TestInstanceCaCertificatesNeedMutualTLS(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		methods, certificates types.Set
		wantError             bool
	}{
		"SCRAM only":              {stringSetToData([]string{"SCRAM_SHA_512"}), stringSetToData([]string{"pem"}), true},
		"MUTUAL_TLS and SCRAM":    {stringSetToData([]string{"MUTUAL_TLS", "SCRAM_SHA_512"}), stringSetToData([]string{"pem"}), false},
		"methods left out":        {types.SetNull(types.StringType), stringSetToData([]string{"pem"}), false},
		"methods not known yet":   {types.SetUnknown(types.StringType), stringSetToData([]string{"pem"}), false},
		"SCRAM, no certificates":  {stringSetToData([]string{"SCRAM_SHA_512"}), stringSetToData(nil), false},
		"SCRAM, certificates out": {stringSetToData([]string{"SCRAM_SHA_512"}), types.SetNull(types.StringType), false},
	} {
		t.Run(name, func(t *testing.T) {
			data := testInstanceData()
			data.AuthenticationMethods, data.CaCertificates = tc.methods, tc.certificates
			r := &instanceResource{}
			config := stateFor(t, r, data)
			resp := validator.SetResponse{}
			caCertificatesNeedMutualTLS{}.ValidateSet(ctx, validator.SetRequest{
				Path: path.Root("ca_certificates"), ConfigValue: tc.certificates, Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw},
			}, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Errorf("error = %v, expected %v: %v", resp.Diagnostics.HasError(), tc.wantError, resp.Diagnostics)
			}
		})
	}
}

// "" is stored as "" or left out of the request, and read back as null, so it is refused where that
// happens.
func TestEmptyStringsAreRefused(t *testing.T) {
	ctx := context.Background()
	resp := validator.StringResponse{}
	notEmpty.ValidateString(ctx, validator.StringRequest{Path: path.Root("description"), ConfigValue: types.StringValue("")}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Error(`"" is not refused`)
	}
	resp = validator.StringResponse{}
	notEmpty.ValidateString(ctx, validator.StringRequest{Path: path.Root("description"), ConfigValue: types.StringValue("x")}, &resp)
	if resp.Diagnostics.HasError() {
		t.Error(`"x" is refused`)
	}

	want := notEmpty.Description(ctx)
	for name, test := range map[string]struct {
		resource   resource.Resource
		attributes []string
	}{
		"cluster": {&clusterResource{}, []string{"description", "location", "kafka_version", "oauth_principals_prefix",
			"broker.certificate", "broker.username", "broker.truststore", "strimzi.provisioner_url", "strimzi.provisioner_username"}},
		"instance": {&instanceResource{}, []string{"description", "connect_urls"}},
		"instance cluster": {&instanceClusterResource{}, []string{"schema_registry.version", "schema_registry.username",
			"schema_registry.password", "schema_registry.certificate", "schema_registry.private_key", "schema_registry.truststore",
			"schema_registry.keycloak.url"}},
	} {
		schemaResp := resource.SchemaResponse{}
		test.resource.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		for _, attribute := range test.attributes {
			var descriptions []string
			switch a := attributeAt(t, schemaResp.Schema.Attributes, attribute).(type) {
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
				t.Errorf("%s %s does not refuse \"\": %v", name, attribute, descriptions)
			}
		}
	}
}

// Platform Manager stores connect_urls as one comma-separated text, so a URL with a comma would come back
// as two.
func TestConnectUrlsRefuseAComma(t *testing.T) {
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	(&instanceResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	urls := schemaResp.Schema.Attributes["connect_urls"].(schema.ListAttribute)
	for value, wantError := range map[string]bool{"http://a:1": false, "http://a:1,http://b:2": true} {
		resp := validator.ListResponse{}
		list := types.ListValueMust(types.StringType, []attr.Value{types.StringValue(value)})
		for _, v := range urls.Validators {
			v.ValidateList(ctx, validator.ListRequest{Path: path.Root("connect_urls"), ConfigValue: list}, &resp)
		}
		if resp.Diagnostics.HasError() != wantError {
			t.Errorf("%q: error = %v, expected %v: %v", value, resp.Diagnostics.HasError(), wantError, resp.Diagnostics)
		}
	}
}

// An Instance PATCH or DELETE that loses the optimistic lock to a parallel schema registry write
// (Platform Manager 16.1.0) is retried, with the lock error of Spring and of JPA.
func TestInstanceUpdateAndDeleteRetryALostLock(t *testing.T) {
	t.Cleanup(shorten(&instanceWriteDelay))
	ctx := context.Background()
	for name, lockError := range map[string]string{"Spring": lockLost, "JPA": lockLostJPA} {
		t.Run(name, func(t *testing.T) {
			calls := map[string]int{}
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				calls[r.Method]++
				if calls[r.Method] == 1 {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(lockError))
					return
				}
				if r.Method == http.MethodDelete {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				_, _ = w.Write([]byte(`{"uid":"i1","name":"Instance","shortName":"inst","instanceClusters":[{"cluster":{"uid":"c1"}}]}`))
			})
			r := &instanceResource{provider: AxualProvider{client: client}}
			plan := testInstanceData()
			plan.Description = types.StringValue("changed")
			planned := stateFor(t, r, plan)
			state := stateFor(t, r, testInstanceData())
			resp := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Schema: planned.Schema, Raw: planned.Raw}, State: state}, &resp)
			if resp.Diagnostics.HasError() || calls[http.MethodPatch] != 2 {
				t.Errorf("update = %v after %d PATCHes, expected success after 2", resp.Diagnostics, calls[http.MethodPatch])
			}
			if del := deleteResource(t, r, testInstanceData()); del.Diagnostics.HasError() || calls[http.MethodDelete] != 2 {
				t.Errorf("delete = %v after %d DELETEs, expected success after 2", del.Diagnostics, calls[http.MethodDelete])
			}
		})
	}
}

// The Strimzi settings Platform Manager 16.1.0 refuses at apply, and a strimzi block on an apache-kafka
// Cluster, are refused at plan time. The checks run through the identity wrapper too.
func TestClusterValidateConfigStrimzi(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		providerType types.String
		strimzi      bool
		sslAuthMode  string
		wantErrors   int
	}{
		"strimzi without the block":   {types.StringValue("strimzi"), false, "", 1},
		"block on apache-kafka":       {types.StringValue("apache-kafka"), true, "", 1},
		"strimzi with fullChain":      {types.StringValue("strimzi"), true, "fullChainCertificate", 1},
		"strimzi":                     {types.StringValue("strimzi"), true, "certificate", 0},
		"apache-kafka with fullChain": {types.StringValue("apache-kafka"), false, "fullChainCertificate", 0},
		"provider_type not known yet": {types.StringUnknown(), true, "", 0},
	} {
		t.Run(name, func(t *testing.T) {
			data := clusterWithStrimzi(t, "apache-kafka", tc.strimzi)
			data.ProviderType = tc.providerType
			data.SslAuthMode = types.StringNull()
			if tc.sslAuthMode != "" {
				data.SslAuthMode = types.StringValue(tc.sslAuthMode)
			}
			r := withIdentity(&clusterResource{}, "id")
			config := stateFor(t, r, data)
			resp := resource.ValidateConfigResponse{}
			r.(resource.ResourceWithValidateConfig).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw}}, &resp)
			if resp.Diagnostics.ErrorsCount() != tc.wantErrors {
				t.Errorf("errors = %d, expected %d: %v", resp.Diagnostics.ErrorsCount(), tc.wantErrors, resp.Diagnostics)
			}
		})
	}
}

// The rule of the runtime provisioner: a Kubernetes object name that is also a label value.
func TestStrimziClusterName(t *testing.T) {
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	(&clusterResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	name := attributeAt(t, schemaResp.Schema.Attributes, "strimzi.cluster_name").(schema.StringAttribute)
	for value, wantError := range map[string]bool{
		"alpha": false, "my-kafka.cluster1": false, strings.Repeat("a", 63): false,
		"Bad_Name": true, "a..b": true, "a.-b": true, "-a": true, "a-": true, "": true, strings.Repeat("a", 64): true,
	} {
		resp := validator.StringResponse{}
		for _, v := range name.Validators {
			v.ValidateString(ctx, validator.StringRequest{Path: path.Root("cluster_name"), ConfigValue: types.StringValue(value)}, &resp)
		}
		if resp.Diagnostics.HasError() != wantError {
			t.Errorf("%q: error = %v, expected %v", value, resp.Diagnostics.HasError(), wantError)
		}
	}
}

// Platform Manager refuses Basic-Auth without a username and password, and MTLS without a certificate
// and private key, so the plan does too.
func TestInstanceClusterConfigNeedsTheCredentialsOfTheAuthentication(t *testing.T) {
	ctx := context.Background()
	set, null := types.StringValue("x"), types.StringNull()
	for name, tc := range map[string]struct {
		authentication                              types.String
		username, password, certificate, privateKey types.String
		wantErrors                                  int
	}{
		"Basic-Auth complete":            {types.StringValue("Basic-Auth"), set, set, null, null, 0},
		"Basic-Auth without credentials": {types.StringValue("Basic-Auth"), null, null, null, null, 2},
		"MTLS complete":                  {types.StringValue("MTLS"), null, null, set, set, 0},
		"MTLS without a key":             {types.StringValue("MTLS"), null, null, set, null, 1},
		"No-Auth":                        {types.StringValue("No-Auth"), null, null, null, null, 0},
		"password not known yet":         {types.StringValue("Basic-Auth"), set, types.StringUnknown(), null, null, 0},
		"authentication not known yet":   {types.StringUnknown(), null, null, null, null, 0},
	} {
		t.Run(name, func(t *testing.T) {
			data := registryWith(t, "APICURIO", "", "", nil, nil)
			var registry schemaRegistryData
			_ = data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})
			registry.Authentication, registry.Username, registry.Password = tc.authentication, tc.username, tc.password
			registry.Certificate, registry.PrivateKey = tc.certificate, tc.privateKey
			data.SchemaRegistry, _ = types.ObjectValueFrom(ctx, schemaRegistryAttrTypes(), registry)
			r := &instanceClusterResource{}
			config := stateFor(t, r, data)
			resp := resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw}}, &resp)
			if resp.Diagnostics.ErrorsCount() != tc.wantErrors {
				t.Errorf("errors = %d, expected %d: %v", resp.Diagnostics.ErrorsCount(), tc.wantErrors, resp.Diagnostics)
			}
		})
	}
}
