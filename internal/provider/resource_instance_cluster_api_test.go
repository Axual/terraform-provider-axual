package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// registryWith is an Instance-Cluster whose schema registry has these values. endpoints maps a name to
// "api url used_by"; nil is a null map.
func registryWith(t *testing.T, registryType, version, api string, listeners map[string]string, endpoints map[string]string) *instanceClusterResourceData {
	t.Helper()
	ctx := context.Background()
	data := testInstanceClusterData(t)
	var registry schemaRegistryData
	if diags := data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("registry: %v", diags)
	}
	registry.Type = types.StringValue(registryType)
	registry.Version = types.StringNull()
	if version != "" {
		registry.Version = types.StringValue(version)
	}
	registry.Api = types.StringNull()
	if api != "" {
		registry.Api = types.StringValue(api)
	}
	registry.Listeners = types.MapNull(types.StringType)
	if listeners != nil {
		registry.Listeners = stringMapToData(listeners)
	}
	registry.ClientEndpoints = types.MapNull(schemaRegistryClientEndpointType())
	if endpoints != nil {
		elements := map[string]attr.Value{}
		for name, value := range endpoints {
			parts := strings.Fields(value)
			elements[name] = types.ObjectValueMust(schemaRegistryClientEndpointType().AttrTypes, map[string]attr.Value{
				"api": types.StringValue(parts[0]), "url": types.StringValue(parts[1]), "used_by": types.StringValue(parts[2]),
			})
		}
		registry.ClientEndpoints = types.MapValueMust(schemaRegistryClientEndpointType(), elements)
	}
	object, diags := types.ObjectValueFrom(ctx, schemaRegistryAttrTypes(), registry)
	if diags.HasError() {
		t.Fatalf("registry: %v", diags)
	}
	data.SchemaRegistry = object
	return data
}

func requestJSON(t *testing.T, plan, config *instanceClusterResourceData) map[string]any {
	t.Helper()
	request, diags := createSchemaRegistryRequestFromData(context.Background(), plan, config)
	if diags.HasError() {
		t.Fatalf("request: %v", diags)
	}
	body, _ := json.Marshal(request)
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	return out
}

// Without api and client_endpoints, the body is the one Platform Manager 15.0.x and 16.0.x know.
func TestSchemaRegistryRequestSendsNothingNewWhenNotSet(t *testing.T) {
	data := registryWith(t, "APICURIO", "", "", map[string]string{"a": "http://a"}, nil)
	body := requestJSON(t, data, data)
	if _, ok := body["api"]; ok {
		t.Errorf("api sent: %v", body)
	}
	if _, ok := body["clientEndpoints"]; ok {
		t.Errorf("clientEndpoints sent: %v", body)
	}
	if listeners, _ := body["listeners"].(map[string]any); listeners["a"] != "http://a" {
		t.Errorf("listeners = %v, expected a", body["listeners"])
	}
}

// Configured endpoints replace the listeners, sorted by name; api is sent when the plan knows it, also
// when it comes from the state.
func TestSchemaRegistryRequestSendsEndpointsAndApi(t *testing.T) {
	endpoints := map[string]string{"b": "APICURIO_V3 http://b ALL_APPS", "a": "APICURIO_V2 http://a MANAGED_APPS"}
	data := registryWith(t, "APICURIO", "", "APICURIO_V3", nil, endpoints)
	body := requestJSON(t, data, data)
	if body["api"] != "APICURIO_V3" || body["listeners"] != nil {
		t.Errorf("api = %v, listeners = %v; expected APICURIO_V3 and null", body["api"], body["listeners"])
	}
	got, _ := json.Marshal(body["clientEndpoints"])
	want := `[{"api":"APICURIO_V2","name":"a","url":"http://a","usedBy":"MANAGED_APPS"},{"api":"APICURIO_V3","name":"b","url":"http://b","usedBy":"ALL_APPS"}]`
	if string(got) != want {
		t.Errorf("clientEndpoints = %s, expected %s", got, want)
	}

	// api not in the configuration, but in the plan from the state: it is sent, so Platform Manager keeps it.
	config := registryWith(t, "APICURIO", "", "", nil, nil)
	plan := registryWith(t, "APICURIO", "", "APICURIO_V2", map[string]string{}, nil)
	if body := requestJSON(t, plan, config); body["api"] != "APICURIO_V2" || body["clientEndpoints"] != nil {
		t.Errorf("api = %v, clientEndpoints = %v; expected APICURIO_V2 and none", body["api"], body["clientEndpoints"])
	}
}

// An endpoint with the name of an existing one gets its id, so Platform Manager updates it in place.
func TestInstanceClusterWriteKeepsEndpointIds(t *testing.T) {
	var posted string
	client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			posted = string(body)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"url":"http://registry/apis/registry/v3","schemaRegistryType":"APICURIO","api":"APICURIO_V3",
			"credentials":{"authenticationType":"Basic-Auth","username":"user","password":""},"listeners":{"a":"http://a"},
			"clientEndpoints":[{"id":"id-a","name":"a","api":"APICURIO_V2","url":"http://a","usedBy":"MANAGED_APPS"}],"properties":{}}`))
	})
	r := &instanceClusterResource{provider: AxualProvider{client: client}}
	data := registryWith(t, "APICURIO", "", "", nil, map[string]string{"a": "APICURIO_V2 http://a MANAGED_APPS", "c": "APICURIO_V3 http://c ALL_APPS"})
	resp := resource.CreateResponse{}
	r.write(context.Background(), data, data, &resp.Diagnostics, "create")
	if resp.Diagnostics.HasError() {
		t.Fatalf("write: %v", resp.Diagnostics)
	}
	if !strings.Contains(posted, `{"id":"id-a","name":"a"`) || !strings.Contains(posted, `{"name":"c"`) {
		t.Errorf("POST body = %s; expected a with its id and c without one", posted)
	}
}

// Platform Manager before 16.1.0 returns no api and no client endpoints: both read as null. 16.1.0
// returns them, also an empty list.
func TestSchemaRegistryReadOfApiAndEndpoints(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		answer        string
		wantApi       string
		wantEndpoints int // -1: null
	}{
		"16.0.3":            {`{"url":"http://r","schemaRegistryType":"APICURIO","listeners":{"a":"http://a"}}`, "", -1},
		"16.1.0, none":      {`{"url":"http://r","schemaRegistryType":"APICURIO","api":"APICURIO_V3","clientEndpoints":[]}`, "APICURIO_V3", 0},
		"16.1.0, endpoints": {`{"url":"http://r","schemaRegistryType":"APICURIO","api":"APICURIO_V2","clientEndpoints":[{"id":"1","name":"a","api":"APICURIO_V2","url":"http://a","usedBy":"ALL_APPS"}]}`, "APICURIO_V2", 1},
	} {
		t.Run(name, func(t *testing.T) {
			var answer webclient.InstanceClusterSchemaRegistry
			if err := json.Unmarshal([]byte(tc.answer), &answer); err != nil {
				t.Fatal(err)
			}
			data := testInstanceClusterData(t)
			var registry schemaRegistryData
			diags := mapSchemaRegistryResponseToData(ctx, data, &answer)
			diags.Append(data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				t.Fatalf("map: %v", diags)
			}
			if registry.Api.ValueString() != tc.wantApi || registry.Api.IsNull() != (tc.wantApi == "") {
				t.Errorf("api = %v, expected %q", registry.Api, tc.wantApi)
			}
			if tc.wantEndpoints < 0 && !registry.ClientEndpoints.IsNull() || tc.wantEndpoints >= 0 && len(registry.ClientEndpoints.Elements()) != tc.wantEndpoints {
				t.Errorf("client_endpoints = %v, expected %d", registry.ClientEndpoints, tc.wantEndpoints)
			}
		})
	}
}

// modifyInstanceClusterPlan runs ModifyPlan with config, a plan and a state (nil: a create).
func modifyInstanceClusterPlan(t *testing.T, r *instanceClusterResource, config, planned, current *instanceClusterResourceData) (resource.ModifyPlanResponse, schemaRegistryData) {
	t.Helper()
	plan := stateFor(t, r, planned)
	cfg := stateFor(t, r, config)
	req := resource.ModifyPlanRequest{Config: tfsdk.Config{Schema: cfg.Schema, Raw: cfg.Raw}, Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}, State: emptyState(t, r)}
	if current != nil {
		req.State = stateFor(t, r, current)
	}
	resp := resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, &resp)
	var data instanceClusterResourceData
	var registry schemaRegistryData
	resp.Diagnostics.Append(resp.Plan.Get(context.Background(), &data)...)
	resp.Diagnostics.Append(data.SchemaRegistry.As(context.Background(), &registry, basetypes.ObjectAsOptions{})...)
	return resp, registry
}

// api and client_endpoints need Platform Manager 16.1.0. The plan reads GET /info only when the
// configuration sets one of them to a new value.
func TestInstanceClusterPlanRefusesApiAndEndpointsBefore161(t *testing.T) {
	endpoint := map[string]string{"a": "APICURIO_V3 http://a ALL_APPS"}
	for _, tc := range []struct {
		name       string
		info       string
		config     *instanceClusterResourceData
		current    *instanceClusterResourceData
		wantErrors int
		wantCalls  int
	}{
		{"api on 16.0.3", `{"version":"16.0.3"}`, registryWith(t, "APICURIO", "", "APICURIO_V3", nil, nil), nil, 1, 1},
		{"endpoints on 15.0.9", `{"version":"15.0.9"}`, registryWith(t, "APICURIO", "", "", nil, endpoint), nil, 1, 1},
		{"both on 16.0.3", `{"version":"16.0.3"}`, registryWith(t, "APICURIO", "", "APICURIO_V3", nil, endpoint), nil, 2, 1},
		{"both on 16.1.0", `{"version":"16.1.0-SNAPSHOT"}`, registryWith(t, "APICURIO", "", "APICURIO_V3", nil, endpoint), nil, 0, 1},
		{"version unreadable", "", registryWith(t, "APICURIO", "", "APICURIO_V3", nil, nil), nil, 1, 1},
		{"neither set", `{"version":"15.0.9"}`, registryWith(t, "APICURIO", "", "", map[string]string{"a": "http://a"}, nil), nil, 0, 0},
		{"unchanged", `{"version":"16.0.3"}`, registryWith(t, "APICURIO", "", "APICURIO_V3", nil, endpoint), registryWith(t, "APICURIO", "", "APICURIO_V3", map[string]string{"a": "http://a"}, endpoint), 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r := &instanceClusterResource{provider: infoPM(t, tc.info, &calls)}
			resp, _ := modifyInstanceClusterPlan(t, r, tc.config, tc.config, tc.current)
			if resp.Diagnostics.ErrorsCount() != tc.wantErrors || calls != tc.wantCalls {
				t.Errorf("errors = %d, GET /info calls = %d; expected %d errors, %d calls. Diagnostics: %v",
					resp.Diagnostics.ErrorsCount(), calls, tc.wantErrors, tc.wantCalls, resp.Diagnostics)
			}
		})
	}
}

// Each error names its own attribute.
func TestInstanceClusterPlanErrorTextBefore161(t *testing.T) {
	calls := 0
	r := &instanceClusterResource{provider: infoPM(t, `{"version":"16.0.3"}`, &calls)}
	config := registryWith(t, "APICURIO", "", "APICURIO_V3", nil, map[string]string{"a": "APICURIO_V3 http://a ALL_APPS"})
	resp, _ := modifyInstanceClusterPlan(t, r, config, config, nil)
	want := map[string]string{
		"api needs Platform Manager 16.1.0 or later":              "This Platform Manager is 16.0.3. It ignores api, so the apply would fail.",
		"client_endpoints needs Platform Manager 16.1.0 or later": "This Platform Manager is 16.0.3. It ignores client_endpoints, so the apply would fail.",
	}
	if errs := resp.Diagnostics.Errors(); len(errs) != len(want) {
		t.Fatalf("errors = %v, expected %d", errs, len(want))
	}
	for _, d := range resp.Diagnostics.Errors() {
		if want[d.Summary()] != d.Detail() {
			t.Errorf("error %q: %q, expected %q", d.Summary(), d.Detail(), want[d.Summary()])
		}
	}
}

// A value Platform Manager derives is planned as unknown when what it comes from changes, and kept
// otherwise.
func TestInstanceClusterPlanOfDerivedValues(t *testing.T) {
	endpointsA := map[string]string{"a": "APICURIO_V3 http://a ALL_APPS"}
	endpointsB := map[string]string{"b": "APICURIO_V3 http://b ALL_APPS"}
	state := registryWith(t, "APICURIO", "", "APICURIO_V3", map[string]string{"a": "http://a"}, endpointsA)
	for _, tc := range []struct {
		name                                           string
		config, plan                                   *instanceClusterResourceData
		wantApi, wantEndpointsUnknown, wantListUnknown bool
	}{
		// The plan holds the state for the values the configuration leaves out (UseStateForUnknown).
		{"nothing changes", registryWith(t, "APICURIO", "", "", nil, nil), state, false, false, false},
		{"version changes", registryWith(t, "APICURIO", "2.6", "", nil, nil),
			registryWith(t, "APICURIO", "2.6", "APICURIO_V3", map[string]string{"a": "http://a"}, endpointsA), true, false, false},
		{"api set, version changes", registryWith(t, "APICURIO", "2.6", "APICURIO_V3", nil, nil),
			registryWith(t, "APICURIO", "2.6", "APICURIO_V3", map[string]string{"a": "http://a"}, endpointsA), false, false, false},
		{"listeners change", registryWith(t, "APICURIO", "", "", map[string]string{"b": "http://b"}, nil),
			registryWith(t, "APICURIO", "", "APICURIO_V3", map[string]string{"b": "http://b"}, endpointsA), false, true, false},
		{"endpoints change", registryWith(t, "APICURIO", "", "", nil, endpointsB),
			registryWith(t, "APICURIO", "", "APICURIO_V3", map[string]string{"a": "http://a"}, endpointsB), false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &instanceClusterResource{provider: AxualProvider{}}
			resp, planned := modifyInstanceClusterPlan(t, r, tc.config, tc.plan, state)
			if resp.Diagnostics.HasError() {
				t.Fatalf("plan: %v", resp.Diagnostics)
			}
			if planned.Api.IsUnknown() != tc.wantApi || planned.ClientEndpoints.IsUnknown() != tc.wantEndpointsUnknown || planned.Listeners.IsUnknown() != tc.wantListUnknown {
				t.Errorf("unknown: api %v, client_endpoints %v, listeners %v; expected %v, %v, %v",
					planned.Api.IsUnknown(), planned.ClientEndpoints.IsUnknown(), planned.Listeners.IsUnknown(),
					tc.wantApi, tc.wantEndpointsUnknown, tc.wantListUnknown)
			}
		})
	}
}

// Platform Manager refuses an api of another type; the plan says so first.
func TestInstanceClusterConfigRefusesApiOfAnotherType(t *testing.T) {
	for name, tc := range map[string]struct {
		registryType, api string
		wantError         bool
	}{
		"CONFLUENT_V7 on APICURIO": {"APICURIO", "CONFLUENT_V7", true},
		"APICURIO_V2 on CONFLUENT": {"CONFLUENT", "APICURIO_V2", true},
		"APICURIO_V2 on APICURIO":  {"APICURIO", "APICURIO_V2", false},
		"no api":                   {"CONFLUENT", "", false},
	} {
		t.Run(name, func(t *testing.T) {
			r := &instanceClusterResource{}
			config := stateFor(t, r, registryWith(t, tc.registryType, "", tc.api, nil, nil))
			resp := resource.ValidateConfigResponse{}
			r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw}}, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Errorf("error = %v, expected %v: %v", resp.Diagnostics.HasError(), tc.wantError, resp.Diagnostics)
			}
		})
	}
}

// lockLost is the answer of Platform Manager 16.1.0 when a parallel schema registry change on the same
// Instance won the optimistic lock.
const lockLost = `Request processing failed: org.springframework.orm.ObjectOptimisticLockingFailureException: Row was already updated or deleted by another transaction for entity [io.axual.governance.systemmanagement.Instance with id 'i1']`

// lockLostJPA is the same error from the Instance update, which calls the JPA EntityManager directly.
const lockLostJPA = "Request processing failed: jakarta.persistence.OptimisticLockException: Unexpected row count (expected row count 1 but was 0) [update `instance` set `description`=?,`opt_lock`=? where `id`=? and `opt_lock`=?] for entity [io.axual.governance.systemmanagement.Instance with id 'i1']"

// A schema registry write that loses the optimistic lock is retried, and the endpoint ids are looked up
// again before every attempt.
func TestInstanceClusterWriteRetriesALostLock(t *testing.T) {
	t.Cleanup(shorten(&schemaRegistryWriteDelay))
	for name, tc := range map[string]struct {
		postAnswers         []int // status of each POST; the rest answer 204
		body                string
		wantPosts, wantGets int
		wantError           bool
	}{
		"lock lost twice":   {[]int{403, 403}, lockLost, 3, 4, false},
		"other 403":         {[]int{403}, `{"detail":"Access is denied"}`, 1, 1, true},
		"lock lost always":  {[]int{403, 403, 403, 403, 403, 403, 403, 403, 403, 403}, lockLost, 10, 10, true},
		"no lock (old PMs)": {nil, "", 1, 2, false},
	} {
		t.Run(name, func(t *testing.T) {
			posts, gets := 0, 0
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					posts++
					if posts <= len(tc.postAnswers) {
						w.WriteHeader(tc.postAnswers[posts-1])
						_, _ = w.Write([]byte(tc.body))
						return
					}
					w.WriteHeader(http.StatusNoContent)
					return
				}
				gets++
				_, _ = w.Write([]byte(`{"url":"http://registry/apis/registry/v3","schemaRegistryType":"APICURIO","api":"APICURIO_V3",
					"credentials":{"authenticationType":"No-Auth"},"listeners":{"a":"http://a"},
					"clientEndpoints":[{"id":"id-a","name":"a","api":"APICURIO_V3","url":"http://a","usedBy":"ALL_APPS"}],"properties":{}}`))
			})
			r := &instanceClusterResource{provider: AxualProvider{client: client}}
			data := registryWith(t, "APICURIO", "", "", nil, map[string]string{"a": "APICURIO_V3 http://a ALL_APPS"})
			resp := resource.CreateResponse{}
			r.write(context.Background(), data, data, &resp.Diagnostics, "create")
			if resp.Diagnostics.HasError() != tc.wantError || posts != tc.wantPosts || gets != tc.wantGets {
				t.Errorf("error = %v, POSTs = %d, GETs = %d; expected error %v, %d POSTs, %d GETs. Diagnostics: %v",
					resp.Diagnostics.HasError(), posts, gets, tc.wantError, tc.wantPosts, tc.wantGets, resp.Diagnostics)
			}
		})
	}
}

func TestInstanceClusterDeleteRetriesALostLock(t *testing.T) {
	t.Cleanup(shorten(&schemaRegistryWriteDelay))
	deletes := 0
	client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
		deletes++
		if deletes == 1 {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(lockLost))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if resp := deleteResource(t, &instanceClusterResource{provider: AxualProvider{client: client}}, testInstanceClusterData(t)); resp.Diagnostics.HasError() || deletes != 2 {
		t.Errorf("delete = %v after %d calls, expected success after 2", resp.Diagnostics, deletes)
	}
}

// A delay of 0 or less retries without waiting, instead of a panic in rand.N.
func TestRetryIfJitteredWithoutADelay(t *testing.T) {
	for _, delay := range []time.Duration{0, -time.Second} {
		calls := 0
		_ = RetryIfJittered(context.Background(), 3, delay, func(error) bool { return true }, func() error {
			calls++
			return io.EOF
		})
		if calls != 3 {
			t.Errorf("delay %v: %d calls, expected 3", delay, calls)
		}
	}
}

func TestRetryIfJitteredWaitsAroundTheDelay(t *testing.T) {
	start := time.Now()
	calls := 0
	_ = RetryIfJittered(context.Background(), 3, 20*time.Millisecond, func(error) bool { return true }, func() error {
		calls++
		return io.EOF
	})
	// Two waits, each between 10 and 30 ms.
	if elapsed := time.Since(start); calls != 3 || elapsed < 20*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Errorf("%d calls in %v, expected 3 calls after two waits of 10 to 30 ms", calls, elapsed)
	}
}

// The POST creates or replaces, so a create refuses an Instance-Cluster that already has a schema
// registry, and tells the user to import it.
func TestInstanceClusterCreateRefusesAnExistingRegistry(t *testing.T) {
	for name, tc := range map[string]struct {
		firstGet  func(w http.ResponseWriter)
		wantPost  bool
		wantError string
	}{
		"registry exists": {func(w http.ResponseWriter) {
			_, _ = w.Write([]byte(`{"url":"https://ui-made.example.com","schemaRegistryType":"APICURIO","credentials":{"authenticationType":"No-Auth"}}`))
		}, false, "already has a schema registry (https://ui-made.example.com)"},
		"no registry": {func(w http.ResponseWriter) {
			_, _ = w.Write([]byte(`{"url":null,"schemaRegistryType":null,"credentials":{"authenticationType":null},"clientEndpoints":[]}`))
		}, true, ""},
		"read fails": {func(w http.ResponseWriter) { w.WriteHeader(http.StatusInternalServerError) }, false, "Unable to read"},
	} {
		t.Run(name, func(t *testing.T) {
			gets, posted := 0, false
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					posted = true
					w.WriteHeader(http.StatusNoContent)
					return
				}
				if r.URL.Path == "/instances/i1" {
					_, _ = w.Write([]byte(`{"uid":"i1","instanceClusters":[{"cluster":{"uid":"c1"}}]}`))
					return
				}
				gets++
				if gets == 1 {
					tc.firstGet(w)
					return
				}
				_, _ = w.Write([]byte(`{"url":"http://registry/apis/registry/v3","schemaRegistryType":"APICURIO",
					"credentials":{"authenticationType":"Basic-Auth","username":"user","password":""},"listeners":{},"properties":{}}`))
			})
			r := &instanceClusterResource{provider: AxualProvider{client: client}}
			planned := stateFor(t, r, testInstanceClusterData(t))
			resp := resource.CreateResponse{State: emptyState(t, r)}
			r.Create(context.Background(), resource.CreateRequest{
				Plan:   tfsdk.Plan{Schema: planned.Schema, Raw: planned.Raw},
				Config: tfsdk.Config{Schema: planned.Schema, Raw: planned.Raw},
			}, &resp)
			gotError := ""
			for _, d := range resp.Diagnostics.Errors() {
				gotError += d.Summary() + ": " + d.Detail()
			}
			if posted != tc.wantPost || (tc.wantError == "") != (gotError == "") || !strings.Contains(gotError, tc.wantError) {
				t.Errorf("POST sent = %v, error = %q; expected POST %v and an error with %q", posted, gotError, tc.wantPost, tc.wantError)
			}
		})
	}
}

// A create whose answer is lost, or whose read after the POST fails, may have saved the registry. It
// keeps the resource in the state with the error, so Terraform taints it and the next apply replaces it,
// instead of refusing the registry as already configured. A refused POST keeps nothing.
func TestInstanceClusterCreateKeepsAMaybeSavedRegistryTainted(t *testing.T) {
	for name, tc := range map[string]struct {
		post, readBack int // statuses; 0 closes the connection without an answer
		wantState      bool
	}{
		"POST answer is a 502":       {http.StatusBadGateway, http.StatusOK, true},
		"POST gets no answer":        {0, http.StatusOK, true},
		"read after the POST is 503": {http.StatusNoContent, http.StatusServiceUnavailable, true},
		"POST refused with a 400":    {http.StatusBadRequest, http.StatusOK, false},
		"POST refused with a 403":    {http.StatusForbidden, http.StatusOK, false},
	} {
		t.Run(name, func(t *testing.T) {
			posted := false
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost && tc.post == 0:
					posted = true
					conn, _, _ := w.(http.Hijacker).Hijack()
					_ = conn.Close()
				case r.Method == http.MethodPost:
					posted = true
					w.WriteHeader(tc.post)
				case !posted:
					_, _ = w.Write([]byte(`{"url":null,"credentials":{"authenticationType":null}}`))
				default:
					w.WriteHeader(tc.readBack)
					_, _ = w.Write([]byte(`{"url":"http://registry/apis/registry/v3","schemaRegistryType":"APICURIO",
						"credentials":{"authenticationType":"Basic-Auth","username":"user","password":""}}`))
				}
			})
			r := &instanceClusterResource{provider: AxualProvider{client: client}}
			// The configuration leaves out the computed values, so the plan has them unknown.
			config := stateFor(t, r, testInstanceClusterData(t))
			data := testInstanceClusterData(t)
			var registry schemaRegistryData
			_ = data.SchemaRegistry.As(context.Background(), &registry, basetypes.ObjectAsOptions{})
			registry.Listeners, registry.Properties = types.MapUnknown(types.StringType), types.MapUnknown(types.StringType)
			registry.Api, registry.ClientEndpoints = types.StringUnknown(), types.MapUnknown(schemaRegistryClientEndpointType())
			data.SchemaRegistry, _ = types.ObjectValueFrom(context.Background(), schemaRegistryAttrTypes(), registry)
			planned := stateFor(t, r, data)
			resp := resource.CreateResponse{State: emptyState(t, r)}
			r.Create(context.Background(), resource.CreateRequest{
				Plan:   tfsdk.Plan{Schema: planned.Schema, Raw: planned.Raw},
				Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw},
			}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected the error to stay: %v", resp.Diagnostics)
			}
			if kept := !resp.State.Raw.IsNull(); kept != tc.wantState {
				t.Fatalf("state kept = %v, expected %v: %v", kept, tc.wantState, resp.Diagnostics)
			}
			if tc.wantState && !resp.State.Raw.IsFullyKnown() {
				t.Errorf("the kept state has unknown values: %v", resp.State.Raw)
			}
		})
	}
}

// Platform Manager answers a stored password or private key as "" and a lost one as null, for example
// after the Instance short_name or the Cluster name changed. A lost one is planned again, with a warning.
func TestInstanceClusterReadFindsALostSecret(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		statePassword, answerPassword string
		wantPassword                  types.String
		wantWarning                   bool
	}{
		"password stored":           {"secret", `""`, types.StringValue("secret"), false},
		"password lost":             {"secret", `null`, types.StringNull(), true},
		"password left out":         {"secret", ``, types.StringNull(), true},
		"imported, no password yet": {"", `null`, types.StringNull(), false},
	} {
		t.Run(name, func(t *testing.T) {
			password := ""
			if tc.answerPassword != "" {
				password = `,"password":` + tc.answerPassword
			}
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"url":"http://registry/apis/registry/v3","schemaRegistryType":"APICURIO",
					"credentials":{"authenticationType":"Basic-Auth","username":"user"` + password + `}}`))
			})
			r := &instanceClusterResource{provider: AxualProvider{client: client}}
			data := testInstanceClusterData(t)
			if tc.statePassword == "" {
				data.SchemaRegistry = testSchemaRegistry(t, "APICURIO", "")
				var registry schemaRegistryData
				_ = data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})
				registry.Password = types.StringNull()
				data.SchemaRegistry, _ = types.ObjectValueFrom(ctx, schemaRegistryAttrTypes(), registry)
			}
			state := stateFor(t, r, data)
			resp := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &resp)
			var got instanceClusterResourceData
			resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
			var registry schemaRegistryData
			resp.Diagnostics.Append(got.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})...)
			if resp.Diagnostics.HasError() || !registry.Password.Equal(tc.wantPassword) || (resp.Diagnostics.WarningsCount() == 1) != tc.wantWarning {
				t.Errorf("password = %v, diagnostics = %v; expected %v and a warning %v", registry.Password, resp.Diagnostics, tc.wantPassword, tc.wantWarning)
			}
		})
	}
}

// The same for the private key of an MTLS registry.
func TestInstanceClusterReadFindsALostPrivateKey(t *testing.T) {
	ctx := context.Background()
	for answerKey, wantKept := range map[string]bool{`""`: true, `null`: false} {
		client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"url":"http://registry/apis/registry/v3","schemaRegistryType":"APICURIO",
				"credentials":{"authenticationType":"mTLS","certificate":"Y2VydA==","privateKey":` + answerKey + `}}`))
		})
		r := &instanceClusterResource{provider: AxualProvider{client: client}}
		data := testInstanceClusterData(t)
		var registry schemaRegistryData
		_ = data.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})
		registry.Authentication, registry.Username, registry.Password = types.StringValue("MTLS"), types.StringNull(), types.StringNull()
		registry.Certificate, registry.PrivateKey = types.StringValue("cert"), types.StringValue("key")
		data.SchemaRegistry, _ = types.ObjectValueFrom(ctx, schemaRegistryAttrTypes(), registry)
		state := stateFor(t, r, data)
		resp := resource.ReadResponse{State: state}
		r.Read(ctx, resource.ReadRequest{State: state}, &resp)
		var got instanceClusterResourceData
		resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
		resp.Diagnostics.Append(got.SchemaRegistry.As(ctx, &registry, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() || (registry.PrivateKey.ValueString() == "key") != wantKept {
			t.Errorf("answer %s: private_key = %v, expected kept %v: %v", answerKey, registry.PrivateKey, wantKept, resp.Diagnostics)
		}
	}
}

// Platform Manager 16.1.0 answers the registry requests of a Cluster that was removed from the Instance
// with a 403. When the Instance shows that the Cluster left, the registry is gone; any other 403 stays an error.
func TestInstanceClusterWhenTheClusterLeftTheInstance(t *testing.T) {
	ctx := context.Background()
	const lazy = `Request processing failed: org.hibernate.LazyInitializationException: could not initialize proxy - no Session`
	for name, tc := range map[string]struct {
		instance             string // the answer of GET /instances/i1; "" answers 403
		wantGone, wantDelete bool
		wantCreateError      string
	}{
		"cluster left":         {`{"uid":"i1","instanceClusters":[{"cluster":{"uid":"c2"}}]}`, true, true, "is not in the clusters"},
		"cluster still there":  {`{"uid":"i1","instanceClusters":[{"cluster":{"uid":"c1"}}]}`, false, false, "Unable to read"},
		"instance not visible": {``, false, false, "Unable to read"},
	} {
		t.Run(name, func(t *testing.T) {
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/instances/i1" && tc.instance != "" {
					_, _ = w.Write([]byte(tc.instance))
					return
				}
				if r.Method == http.MethodPost {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(lazy))
			})
			r := &instanceClusterResource{provider: AxualProvider{client: client}}
			state := stateFor(t, r, testInstanceClusterData(t))
			read := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &read)
			if gone := !read.Diagnostics.HasError() && read.State.Raw.IsNull(); gone != tc.wantGone {
				t.Errorf("read: gone = %v, expected %v: %v", gone, tc.wantGone, read.Diagnostics)
			}
			if deleted := !deleteResource(t, r, testInstanceClusterData(t)).Diagnostics.HasError(); deleted != tc.wantDelete {
				t.Errorf("delete: done = %v, expected %v", deleted, tc.wantDelete)
			}
			created := resource.CreateResponse{State: emptyState(t, r)}
			r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw},
				Config: tfsdk.Config{Schema: state.Schema, Raw: state.Raw}}, &created)
			if !created.Diagnostics.HasError() || !strings.Contains(created.Diagnostics.Errors()[0].Detail(), tc.wantCreateError) {
				t.Errorf("create: %v, expected an error with %q", created.Diagnostics, tc.wantCreateError)
			}
		})
	}
}

// A schema registry for a Cluster that is not in the Instance says so, instead of "resource not found".
func TestInstanceClusterWriteNamesAClusterNotInTheInstance(t *testing.T) {
	client := fakePM(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	r := &instanceClusterResource{provider: AxualProvider{client: client}}
	data := testInstanceClusterData(t)
	var diags diag.Diagnostics
	r.write(context.Background(), data, data, &diags, "create")
	if !diags.HasError() || !strings.Contains(diags.Errors()[0].Detail(), "Cluster c1 is not in the clusters of Instance i1") {
		t.Errorf("diagnostics = %v, expected that the Cluster is not in the Instance", diags)
	}
}
