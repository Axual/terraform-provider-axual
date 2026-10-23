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
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
		Type:           types.StringValue(registryType),
		Url:            types.StringValue("http://registry/apis/registry/v3"),
		Version:        types.StringNull(),
		Authentication: types.StringValue("Basic-Auth"),
		Username:       types.StringValue("user"),
		Password:       types.StringValue(password),
		Certificate:    types.StringNull(),
		PrivateKey:     types.StringNull(),
		Truststore:     types.StringNull(),
		OauthEnabled:   types.BoolValue(false),
		Keycloak:       types.ObjectNull(schemaRegistryKeycloakAttrTypes()),
		Listeners:      types.MapNull(types.StringType),
		Properties:     types.MapNull(types.StringType),
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
	tests := map[string]struct {
		handler  http.HandlerFunc
		resource func(AxualProvider) resource.Resource
		data     any
	}{
		"cluster":                       {pmInstanceGone, func(p AxualProvider) resource.Resource { return &clusterResource{provider: p} }, testClusterData(t)},
		"instance":                      {pmInstanceGone, func(p AxualProvider) resource.Resource { return &instanceResource{provider: p} }, testInstanceData()},
		"instance cluster, link gone":   {pmInstanceGone, func(p AxualProvider) resource.Resource { return &instanceClusterResource{provider: p} }, testInstanceClusterData(t)},
		"instance cluster, no registry": {emptyRegistry, func(p AxualProvider) resource.Resource { return &instanceClusterResource{provider: p} }, testInstanceClusterData(t)},
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
	srRequest, d := createSchemaRegistryRequestFromData(ctx, data)
	diags.Append(d...)
	if srRequest.Credentials.Certificate != encoded || srRequest.Credentials.PrivateKey != encoded || srRequest.Credentials.Truststore != encoded {
		t.Errorf("schema registry credentials = %+v, expected base64", srRequest.Credentials)
	}

	answer := webclient.InstanceClusterSchemaRegistry{Url: "http://r", SchemaRegistryType: "confluent",
		Credentials: &webclient.SchemaRegistryCredentials{AuthenticationType: "MTLS", Certificate: encoded, Truststore: encoded}}
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
		request, diags := createSchemaRegistryRequestFromData(context.Background(), data)
		if diags.HasError() {
			t.Fatalf("request: %v", diags)
		}
		if (request.Authentications != nil) != expectAuthentications {
			t.Errorf("%s: authentications = %v", registryType, request.Authentications)
		}
		if expectAuthentications && !request.Authentications.Basic {
			t.Errorf("basic authentication must stay on")
		}
		if request.Credentials.Password != "secret" {
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
	r.write(context.Background(), data, &diags.Diagnostics, "create")
	var registry schemaRegistryData
	diags.Diagnostics.Append(data.SchemaRegistry.As(context.Background(), &registry, basetypes.ObjectAsOptions{})...)
	if diags.Diagnostics.HasError() {
		t.Fatalf("write: %v", diags.Diagnostics)
	}
	if registry.Password.ValueString() != "secret" || data.Id.ValueString() != "i1/c1" {
		t.Errorf("password = %q, id = %q", registry.Password.ValueString(), data.Id.ValueString())
	}
}
