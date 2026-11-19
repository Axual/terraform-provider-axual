package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func testBroker(t *testing.T, privateKey string) types.Object {
	t.Helper()
	object, diags := types.ObjectValueFrom(context.Background(), clusterBrokerAttrTypes(), clusterBrokerData{
		BootstrapServers: types.StringValue("old:9092"),
		Protocol:         types.StringValue("MTLS"),
		SecurityProtocol: types.StringNull(),
		Certificate:      types.StringValue("old-cert"),
		PrivateKey:       types.StringValue(privateKey),
		Username:         types.StringNull(),
		Password:         types.StringNull(),
		Truststore:       types.StringNull(),
		CaValidation:     types.BoolValue(true),
	})
	if diags.HasError() {
		t.Fatalf("broker: %v", diags)
	}
	return object
}

func testClusterData(t *testing.T) *clusterResourceData {
	return &clusterResourceData{
		Id:                            types.StringValue("c1"),
		Name:                          types.StringValue("c"),
		Shared:                        types.BoolValue(false),
		ProviderType:                  types.StringValue("apache-kafka"),
		Broker:                        testBroker(t, "secret-key"),
		Strimzi:                       types.ObjectNull(clusterStrimziAttrTypes()),
		BootstrapUrlsByAuthentication: types.MapNull(types.StringType),
		KafkaListeners:                types.MapNull(types.StringType),
		BillingCloudEnabled:           types.BoolValue(false),
	}
}

func testInstanceData() *instanceResourceData {
	return &instanceResourceData{
		Id:                    types.StringValue("i1"),
		Name:                  types.StringValue("Instance"),
		ShortName:             types.StringValue("inst"),
		Clusters:              stringSetToData([]string{"c1"}),
		AuthenticationMethods: types.SetNull(types.StringType),
		CaCertificates:        types.SetNull(types.StringType),
		Properties:            types.MapNull(types.StringType),
		ConnectUrls:           types.ListNull(types.StringType),
	}
}

func testSchemaRegistry(t *testing.T, registryType string, password string) types.Object {
	t.Helper()
	object, diags := types.ObjectValueFrom(context.Background(), schemaRegistryAttrTypes(), schemaRegistryData{
		Type:            types.StringValue(registryType),
		Url:             types.StringValue("http://registry/apis/registry/v3"),
		Version:         types.StringNull(),
		Authentication:  types.StringValue("Basic-Auth"),
		Username:        types.StringValue("user"),
		Password:        types.StringValue(password),
		Certificate:     types.StringNull(),
		PrivateKey:      types.StringNull(),
		Truststore:      types.StringNull(),
		OauthEnabled:    types.BoolValue(false),
		Keycloak:        types.ObjectNull(schemaRegistryKeycloakAttrTypes()),
		Listeners:       types.MapNull(types.StringType),
		Properties:      types.MapNull(types.StringType),
		Api:             types.StringNull(),
		ClientEndpoints: types.MapNull(schemaRegistryClientEndpointType()),
	})
	if diags.HasError() {
		t.Fatalf("schema registry: %v", diags)
	}
	return object
}

func testInstanceClusterData(t *testing.T) *instanceClusterResourceData {
	return &instanceClusterResourceData{
		Id:             types.StringValue("i1/c1"),
		Instance:       types.StringValue("i1"),
		Cluster:        types.StringValue("c1"),
		SchemaRegistry: testSchemaRegistry(t, "APICURIO", "secret"),
	}
}

// pmInstanceGone answers like Platform Manager for a cluster, instance or instance-cluster that does
// not exist: a missing instance gets a 400, not a 404.
func pmInstanceGone(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/instances/") && !strings.Contains(r.URL.Path, "/schema-registry") {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"Required URI template variable 'uid' for method parameter type is present but converted to null"}`))
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func TestClusterAndInstanceDeleteAcceptsAGoneResource(t *testing.T) {
	provider := AxualProvider{client: fakePM(t, pmInstanceGone)}
	tests := map[string]struct {
		resource resource.Resource
		data     any
	}{
		"cluster":          {&clusterResource{provider: provider}, testClusterData(t)},
		"instance":         {&instanceResource{provider: provider}, testInstanceData()},
		"instance cluster": {&instanceClusterResource{provider: provider}, testInstanceClusterData(t)},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if resp := deleteResource(t, test.resource, test.data); resp.Diagnostics.HasError() {
				t.Errorf("delete error = %v, expected none for a resource that is gone", resp.Diagnostics)
			}
		})
	}
}

func TestClusterAndInstanceReadDropsAGoneResource(t *testing.T) {
	emptyRegistry := func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"url":null,"schemaRegistryType":null,"credentials":{"authenticationType":null}}`))
	}
	// Platform Manager 16.1.0 looks up the instance and the cluster of the schema registry itself.
	instanceNotFound := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"Instance not found","status":400}`))
	}
	tests := map[string]struct {
		handler  http.HandlerFunc
		resource func(AxualProvider) resource.Resource
		data     any
	}{
		"cluster":                       {pmInstanceGone, func(p AxualProvider) resource.Resource { return &clusterResource{provider: p} }, testClusterData(t)},
		"instance":                      {pmInstanceGone, func(p AxualProvider) resource.Resource { return &instanceResource{provider: p} }, testInstanceData()},
		"instance cluster, link gone":   {pmInstanceGone, func(p AxualProvider) resource.Resource { return &instanceClusterResource{provider: p} }, testInstanceClusterData(t)},
		"instance cluster, no registry": {emptyRegistry, func(p AxualProvider) resource.Resource { return &instanceClusterResource{provider: p} }, testInstanceClusterData(t)},
		"instance gone (PM 16.1.0)":     {instanceNotFound, func(p AxualProvider) resource.Resource { return &instanceClusterResource{provider: p} }, testInstanceClusterData(t)},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			r := test.resource(AxualProvider{client: fakePM(t, test.handler)})
			state := stateFor(t, r, test.data)
			resp := resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
				t.Errorf("read = %v, state null = %t; expected the resource removed from the state", resp.Diagnostics, resp.State.Raw.IsNull())
			}
		})
	}
}

// Platform Manager answers the private key as "", so a read keeps the one in state and takes
// everything else from the answer.
func TestClusterReadKeepsTheSecrets(t *testing.T) {
	client := fakePM(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"uid":"c1","name":"c","providerType":"apache-kafka","sharedCluster":false,
			"authenticationInfo":{"Axl-Bootstrap-Server-ApacheKafka":"new:9092","Axl-Broker-Protocol-Type-ApacheKafka":"MTLS",
			"Axl-Broker-Cert-ApacheKafka":"new-cert","Axl-Broker-Private-Key-ApacheKafka":"","Axl-Broker-CA-Validation-ApacheKafka":"false"},
			"settings":{"sslAuthMode":"fullChainCertificate"}}`))
	})
	r := &clusterResource{provider: AxualProvider{client: client}}
	state := stateFor(t, r, testClusterData(t))
	resp := resource.ReadResponse{State: state}

	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

	var data clusterResourceData
	resp.Diagnostics.Append(resp.State.Get(context.Background(), &data)...)
	var broker clusterBrokerData
	resp.Diagnostics.Append(data.Broker.As(context.Background(), &broker, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	if broker.PrivateKey.ValueString() != "secret-key" {
		t.Errorf("private_key = %q, expected the one in state", broker.PrivateKey.ValueString())
	}
	if broker.BootstrapServers.ValueString() != "new:9092" || broker.Certificate.ValueString() != "new-cert" || broker.CaValidation.ValueBool() {
		t.Errorf("broker = %+v, expected the answer's bootstrap servers, certificate and ca_validation false", broker)
	}
	if data.SslAuthMode.ValueString() != "fullChainCertificate" {
		t.Errorf("ssl_auth_mode = %q", data.SslAuthMode.ValueString())
	}
}

func TestClusterRequestSendsTheSecretsAndOwnershipOnlyOnCreate(t *testing.T) {
	data := testClusterData(t)
	create, diags := createClusterRequestFromData(context.Background(), data, true)
	update, _ := createClusterRequestFromData(context.Background(), data, false)
	if diags.HasError() {
		t.Fatalf("request: %v", diags)
	}
	if create.ProviderInfo[webclient.ProviderInfoBrokerPrivateKey] != "secret-key" || update.ProviderInfo[webclient.ProviderInfoBrokerPrivateKey] != "secret-key" {
		t.Errorf("the private key must be sent on create and update")
	}
	if create.ProviderInfo[webclient.ProviderInfoProviderType] != "apache-kafka" || create.ProviderInfo[webclient.ProviderInfoBrokerCAValidation] != "true" {
		t.Errorf("providerInfo = %v", create.ProviderInfo)
	}
	if create.SharedCluster == nil || *create.SharedCluster || update.SharedCluster != nil {
		t.Errorf("sharedCluster must be false on create and absent on update")
	}
	body, _ := json.Marshal(update)
	if !strings.Contains(string(body), `"description":null`) {
		t.Errorf("a removed description must be sent as null: %s", body)
	}
}

// Platform Manager and the Self-Service UI keep certificates, keys and truststores base64-encoded,
// and parts of Platform Manager decode them, so PEM from the configuration is sent encoded and read
// back as PEM.
func TestPEMValuesAreSentBase64AndReadBackAsPEM(t *testing.T) {
	const pem = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(pem))
	ctx := context.Background()

	cluster := testClusterData(t)
	broker, diags := types.ObjectValueFrom(ctx, clusterBrokerAttrTypes(), clusterBrokerData{
		BootstrapServers: types.StringValue("b:9092"), Protocol: types.StringValue("MTLS"), SecurityProtocol: types.StringNull(),
		Certificate: types.StringValue(pem), PrivateKey: types.StringValue(pem), Username: types.StringNull(),
		Password: types.StringNull(), Truststore: types.StringValue(pem), CaValidation: types.BoolValue(true),
	})
	cluster.Broker = broker
	request, d := createClusterRequestFromData(ctx, cluster, true)
	diags.Append(d...)
	for _, key := range []string{webclient.ProviderInfoBrokerCertificate, webclient.ProviderInfoBrokerPrivateKey, webclient.ProviderInfoBrokerTruststore} {
		if request.ProviderInfo[key] != encoded {
			t.Errorf("%s = %q, expected base64", key, request.ProviderInfo[key])
		}
	}

	data := testInstanceClusterData(t)
	var registry schemaRegistryData
	diags.Append(testSchemaRegistry(t, "CONFLUENT", "secret").As(ctx, &registry, basetypes.ObjectAsOptions{})...)
	registry.Authentication = types.StringValue("MTLS")
	registry.Certificate, registry.PrivateKey, registry.Truststore = types.StringValue(pem), types.StringValue(pem), types.StringValue(pem)
	data.SchemaRegistry, d = types.ObjectValueFrom(ctx, schemaRegistryAttrTypes(), registry)
	diags.Append(d...)
	srRequest, d := createSchemaRegistryRequestFromData(ctx, data, data)
	diags.Append(d...)
	if srRequest.Credentials.Certificate != encoded || srRequest.Credentials.PrivateKey == nil || *srRequest.Credentials.PrivateKey != encoded || srRequest.Credentials.Truststore != encoded {
		t.Errorf("schema registry credentials = %+v, expected base64", srRequest.Credentials)
	}

	answer := webclient.InstanceClusterSchemaRegistry{Url: "http://r", SchemaRegistryType: "confluent",
		Credentials: &webclient.SchemaRegistryCredentials{AuthenticationType: "MTLS", Certificate: encoded, PrivateKey: new(string), Truststore: encoded}}
	diags.Append(mapSchemaRegistryResponseToData(ctx, data, &answer)...)
	diags.Append(data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if registry.Certificate.ValueString() != pem || registry.Truststore.ValueString() != pem || registry.PrivateKey.ValueString() != pem {
		t.Errorf("read back = %+v, expected the PEM of the configuration", registry)
	}
	if got := pemFromAPI(encoded, types.StringValue(encoded)); got != encoded {
		t.Errorf("a base64 value in state must stay base64, got %q", got)
	}
}

func TestInstanceRequestAndResponseMapping(t *testing.T) {
	data := testInstanceData()
	data.AuthenticationMethods = stringSetToData([]string{"SCRAM_SHA_512", "MUTUAL_TLS"})
	data.ConnectUrls = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("http://a:1"), types.StringValue("http://b:2")})
	data.CertificateReusePrevention = types.StringValue("TENANT")

	request := createInstanceRequestFromData(data, "https://pm/api")

	if len(request.InstanceClusters) != 1 || request.InstanceClusters[0].Cluster != "https://pm/api/clusters/c1" {
		t.Errorf("instanceClusters = %+v", request.InstanceClusters)
	}
	methods := map[string]string{}
	for _, m := range request.EnabledAuthenticationMethods {
		methods[m.Mechanism] = m.Protocol
	}
	if methods["MUTUAL_TLS"] != "SSL" || methods["SCRAM_SHA_512"] != "SASL" {
		t.Errorf("authentication methods = %v", methods)
	}
	if request.ConnectUrls == nil || *request.ConnectUrls != "http://a:1,http://b:2" {
		t.Errorf("connectUrls = %v", request.ConnectUrls)
	}
	if request.CaCerts == nil {
		t.Errorf("caCerts must be sent, also when empty: Platform Manager requires it on create")
	}

	var answer webclient.InstanceResponse
	_ = json.Unmarshal([]byte(`{"uid":"i1","name":"Instance","shortName":"inst","certificateReusePreventionLevel":"None",
		"connectUrls":"http://a:1, http://b:2","instanceClusters":[{"cluster":{"uid":"c1"}}],
		"enabledAuthenticationMethods":[{"rank":0,"protocol":"SSL","mechanism":"MUTUAL_TLS"}]}`), &answer)
	mapInstanceResponseToData(data, &answer)
	if data.CertificateReusePrevention.ValueString() != "NONE" {
		t.Errorf("certificate_reuse_prevention = %q, expected NONE", data.CertificateReusePrevention.ValueString())
	}
	if len(data.ConnectUrls.Elements()) != 2 || len(data.Clusters.Elements()) != 1 || len(data.AuthenticationMethods.Elements()) != 1 {
		t.Errorf("mapped = %+v", data)
	}
}

// The authentications block is Apicurio only; basic authentication is always on.
func TestSchemaRegistryRequest(t *testing.T) {
	for registryType, expectAuthentications := range map[string]bool{"APICURIO": true, "CONFLUENT": false} {
		data := testInstanceClusterData(t)
		data.SchemaRegistry = testSchemaRegistry(t, registryType, "secret")
		request, diags := createSchemaRegistryRequestFromData(context.Background(), data, data)
		if diags.HasError() {
			t.Fatalf("request: %v", diags)
		}
		if (request.Authentications != nil) != expectAuthentications {
			t.Errorf("%s: authentications = %v", registryType, request.Authentications)
		}
		if expectAuthentications && !request.Authentications.Basic {
			t.Errorf("basic authentication must stay on")
		}
		if request.Credentials.Password == nil || *request.Credentials.Password != "secret" {
			t.Errorf("%s: password not sent", registryType)
		}
	}
}

// After the POST the registry is read back; the password the answer blanks stays as configured.
func TestInstanceClusterCreateKeepsThePassword(t *testing.T) {
	client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"password":"secret"`) {
				t.Errorf("POST body has no password: %s", body)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"url":"http://registry/apis/registry/v3","schemaRegistryType":"APICURIO",
			"credentials":{"authenticationType":"Basic-Auth","username":"user","password":""},
			"authentications":{"basic":true,"oAuth":false},"listeners":{},"properties":{}}`))
	})
	r := &instanceClusterResource{provider: AxualProvider{client: client}}
	data := testInstanceClusterData(t)
	var diags = &resource.CreateResponse{}
	r.write(context.Background(), data, data, &diags.Diagnostics, "create")
	var registry schemaRegistryData
	diags.Diagnostics.Append(data.SchemaRegistry.As(context.Background(), &registry, basetypes.ObjectAsOptions{})...)
	if diags.Diagnostics.HasError() {
		t.Fatalf("write: %v", diags.Diagnostics)
	}
	if registry.Password.ValueString() != "secret" || data.Id.ValueString() != "i1/c1" {
		t.Errorf("password = %q, id = %q", registry.Password.ValueString(), data.Id.ValueString())
	}
}

// The Self-Service UI writes and compares MTLS as "mTLS"; Platform Manager ignores the case. The
// provider sends "mTLS" and reads any case back as the "MTLS" of the configuration, also the "MTLS"
// that earlier builds stored.
func TestMTLSIsSentAsTheUISpellsItAndReadInAnyCase(t *testing.T) {
	ctx := context.Background()
	for value, want := range map[string]string{"MTLS": "mTLS", "mtls": "mTLS", "SASL/PLAIN": "SASL/PLAIN", "Basic-Auth": "Basic-Auth", "No-Auth": "No-Auth"} {
		if got := mtlsToAPI(value); got != want {
			t.Errorf("mtlsToAPI(%q) = %q, expected %q", value, got, want)
		}
	}

	request, diags := createClusterRequestFromData(ctx, testClusterData(t), true)
	if diags.HasError() || request.ProviderInfo[webclient.ProviderInfoBrokerProtocol] != "mTLS" {
		t.Errorf("broker protocol sent = %q, expected mTLS (%v)", request.ProviderInfo[webclient.ProviderInfoBrokerProtocol], diags)
	}
	for stored, want := range map[string]string{"mTLS": "MTLS", "MTLS": "MTLS", "sasl/scram-sha-512": "SASL/SCRAM-SHA-512"} {
		data := testClusterData(t)
		answer := webclient.ClusterResponse{Uid: "c1", Name: "c", AuthenticationInfo: map[string]string{webclient.ProviderInfoBrokerProtocol: stored}}
		diags := mapClusterResponseToData(ctx, data, &answer)
		var broker clusterBrokerData
		diags.Append(data.Broker.As(ctx, &broker, basetypes.ObjectAsOptions{})...)
		if diags.HasError() || broker.Protocol.ValueString() != want {
			t.Errorf("stored protocol %q read as %q, expected %q (%v)", stored, broker.Protocol.ValueString(), want, diags)
		}
	}

	registry := testInstanceClusterData(t)
	var sr schemaRegistryData
	diags = registry.SchemaRegistry.As(ctx, &sr, basetypes.ObjectAsOptions{})
	sr.Authentication = types.StringValue("MTLS")
	registry.SchemaRegistry, _ = types.ObjectValueFrom(ctx, schemaRegistryAttrTypes(), sr)
	srRequest, d := createSchemaRegistryRequestFromData(ctx, registry, registry)
	diags.Append(d...)
	if diags.HasError() || srRequest.Credentials.AuthenticationType != "mTLS" {
		t.Errorf("schema registry authentication sent = %q, expected mTLS (%v)", srRequest.Credentials.AuthenticationType, diags)
	}
	for stored, want := range map[string]string{"mTLS": "MTLS", "MTLS": "MTLS", "Basic-Auth": "Basic-Auth", "basic-auth": "Basic-Auth", "No-Auth": "No-Auth"} {
		answer := webclient.InstanceClusterSchemaRegistry{Url: "http://r", SchemaRegistryType: "APICURIO",
			Credentials: &webclient.SchemaRegistryCredentials{AuthenticationType: stored}}
		diags := mapSchemaRegistryResponseToData(ctx, registry, &answer)
		diags.Append(registry.SchemaRegistry.As(ctx, &sr, basetypes.ObjectAsOptions{})...)
		if diags.HasError() || sr.Authentication.ValueString() != want {
			t.Errorf("stored authentication %q read as %q, expected %q (%v)", stored, sr.Authentication.ValueString(), want, diags)
		}
	}
}

// connectEnabled and flinkEnabled follow the answer when Platform Manager returns them. When it does not,
// the value of the plan or state is kept, and false when that is null or unknown. Only a kept true warns.
func TestInstanceRuntimeSwitchFromTheAnswer(t *testing.T) {
	yes, no := true, false
	for name, tc := range map[string]struct {
		answer      *bool
		current     types.Bool
		want        bool
		wantWarning bool
	}{
		"answer true":         {&yes, types.BoolValue(false), true, false},
		"answer false":        {&no, types.BoolValue(true), false, false},
		"missing, plan true":  {nil, types.BoolValue(true), true, true},
		"missing, plan false": {nil, types.BoolValue(false), false, false},
		"missing, plan null":  {nil, types.BoolNull(), false, false},
		"missing, not known":  {nil, types.BoolUnknown(), false, false},
	} {
		t.Run(name, func(t *testing.T) {
			data := testInstanceData()
			data.ConnectEnabled, data.FlinkEnabled = tc.current, tc.current
			answer := &webclient.InstanceResponse{Uid: "i1", ConnectEnabled: tc.answer, FlinkEnabled: tc.answer}
			var diags diag.Diagnostics
			warnRuntimeSwitchesWithoutEffect(&diags, data, answer)
			mapInstanceResponseToData(data, answer)
			for attribute, value := range map[string]types.Bool{"connect_enabled": data.ConnectEnabled, "flink_enabled": data.FlinkEnabled} {
				if value.IsNull() || value.IsUnknown() || value.ValueBool() != tc.want {
					t.Errorf("%s = %v, expected %v", attribute, value, tc.want)
				}
			}
			wantWarnings := 0
			if tc.wantWarning {
				wantWarnings = 2
			}
			if diags.WarningsCount() != wantWarnings || diags.HasError() {
				t.Errorf("diagnostics = %v, expected %d warnings", diags, wantWarnings)
			}
		})
	}
}

// The answers of each Platform Manager line, through Create, Update and Read (an import reads with
// both values null). 15.0.x returns connectEnabled only; versions with Connect and Flink per tenant
// return neither, and then only a configured true warns.
func TestInstanceRuntimeSwitchesOnEachPlatformManager(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		answer                             string
		connect, flink                     bool // the plan
		wantConnect, wantFlink             bool // the state after Create and Update
		wantImportConnect, wantImportFlink bool // the state after an import
		wantWarnings                       int  // each of Create and Update
	}{
		"16.0.x":                    {`"connectEnabled":true,"flinkEnabled":true`, true, true, true, true, true, true, 0},
		"15.0.x":                    {`"connectEnabled":true`, true, false, true, false, true, false, 0},
		"per tenant, both true":     {``, true, true, true, true, false, false, 2},
		"per tenant, flink true":    {``, false, true, false, true, false, false, 1},
		"per tenant, left at false": {``, false, false, false, false, false, false, 0},
	} {
		t.Run(name, func(t *testing.T) {
			body := `{"uid":"i1","name":"Instance","shortName":"inst","instanceClusters":[{"cluster":{"uid":"c1"}}]}`
			if tc.answer != "" {
				body = strings.Replace(body, `{`, `{`+tc.answer+`,`, 1)
			}
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					w.WriteHeader(http.StatusCreated)
				}
				_, _ = w.Write([]byte(body))
			})
			r := &instanceResource{provider: AxualProvider{client: client}}
			plan := testInstanceData()
			plan.ConnectEnabled, plan.FlinkEnabled = types.BoolValue(tc.connect), types.BoolValue(tc.flink)
			planned := stateFor(t, r, plan)
			check := func(step string, state tfsdk.State, diags diag.Diagnostics, wantConnect, wantFlink bool, wantWarnings int) {
				t.Helper()
				var got instanceResourceData
				diags.Append(state.Get(ctx, &got)...)
				if diags.HasError() || diags.WarningsCount() != wantWarnings ||
					got.ConnectEnabled.ValueBool() != wantConnect || got.FlinkEnabled.ValueBool() != wantFlink {
					t.Errorf("%s: connect_enabled = %v, flink_enabled = %v, %d warnings %v; expected %v, %v, %d warnings",
						step, got.ConnectEnabled, got.FlinkEnabled, diags.WarningsCount(), diags, wantConnect, wantFlink, wantWarnings)
				}
			}

			created := resource.CreateResponse{State: emptyState(t, r)}
			r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: planned.Schema, Raw: planned.Raw}}, &created)
			check("create", created.State, created.Diagnostics, tc.wantConnect, tc.wantFlink, tc.wantWarnings)

			updated := resource.UpdateResponse{State: planned}
			r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Schema: planned.Schema, Raw: planned.Raw}, State: planned}, &updated)
			check("update", updated.State, updated.Diagnostics, tc.wantConnect, tc.wantFlink, tc.wantWarnings)

			imported := stateFor(t, r, testInstanceData())
			read := resource.ReadResponse{State: imported}
			r.Read(ctx, resource.ReadRequest{State: imported}, &read)
			check("import", read.State, read.Diagnostics, tc.wantImportConnect, tc.wantImportFlink, 0)
		})
	}
}

// A refresh warns too, so a configured true that does nothing shows on every plan after an upgrade of
// Platform Manager, not only on the next create or update.
func TestInstanceRefreshWarnsAboutRuntimeSwitchesWithoutEffect(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		answer       string
		state        bool
		wantWarnings int
	}{
		"per tenant, true in the state":  {``, true, 2},
		"per tenant, false in the state": {``, false, 0},
		"16.0.x, true in the state":      {`"connectEnabled":true,"flinkEnabled":true,`, true, 0},
	} {
		t.Run(name, func(t *testing.T) {
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{` + tc.answer + `"uid":"i1","name":"Instance","shortName":"inst","instanceClusters":[{"cluster":{"uid":"c1"}}]}`))
			})
			r := &instanceResource{provider: AxualProvider{client: client}}
			data := testInstanceData()
			data.ConnectEnabled, data.FlinkEnabled = types.BoolValue(tc.state), types.BoolValue(tc.state)
			state := stateFor(t, r, data)
			resp := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != tc.wantWarnings {
				t.Errorf("diagnostics = %v, expected %d warnings", resp.Diagnostics, tc.wantWarnings)
			}
		})
	}
}
