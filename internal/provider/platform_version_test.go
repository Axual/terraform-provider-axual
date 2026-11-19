package provider

import (
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPlatformVersionAtLeast(t *testing.T) {
	for version, want := range map[string]bool{
		"15.0.9": false, "16.0.0": true, "16.0.3": true, "16.1.0-SNAPSHOT": true, "17.0.0": true, "": false, "dev": false,
	} {
		if got := platformVersionAtLeast(version, 16, 0); got != want {
			t.Errorf("platformVersionAtLeast(%q, 16, 0) = %v, expected %v", version, got, want)
		}
		if got := SupportsInstanceFlink(version); got != want {
			t.Errorf("SupportsInstanceFlink(%q) = %v, expected %v", version, got, want)
		}
	}
}

// infoPM answers GET /info with info ("" answers 500) and counts the calls.
func infoPM(t *testing.T, info string, calls *int) AxualProvider {
	return AxualProvider{client: fakePM(t, func(w http.ResponseWriter, r *http.Request) {
		*calls++
		if r.URL.Path != "/info" || info == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(info))
	})}
}

// modifyPlanOf runs ModifyPlan of r for a plan of planned and a state of current (nil: a create).
func modifyPlanOf(t *testing.T, r resource.Resource, planned, current any) resource.ModifyPlanResponse {
	t.Helper()
	plan := stateFor(t, r, planned)
	req := resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}, State: emptyState(t, r)}
	if current != nil {
		req.State = stateFor(t, r, current)
	}
	resp := resource.ModifyPlanResponse{Plan: req.Plan}
	r.(resource.ResourceWithModifyPlan).ModifyPlan(context.Background(), req, &resp)
	return resp
}

func clusterWithStrimzi(t *testing.T, providerType string, strimzi bool) *clusterResourceData {
	data := testClusterData(t)
	data.ProviderType = types.StringValue(providerType)
	if strimzi {
		object, diags := types.ObjectValueFrom(context.Background(), clusterStrimziAttrTypes(), clusterStrimziData{
			ClusterName: types.StringValue("kafka"), Namespace: types.StringValue("kafka"), ProvisionerUrl: types.StringNull(),
			ProvisionerUsername: types.StringNull(), ProvisionerPassword: types.StringNull(),
		})
		if diags.HasError() {
			t.Fatalf("strimzi: %v", diags)
		}
		data.Strimzi = object
	}
	return data
}

// Platform Manager knows Strimzi from 16.1.0 on. The plan reads GET /info only when it starts to use Strimzi.
func TestClusterPlanRefusesStrimziOnOlderPlatformManager(t *testing.T) {
	for _, tc := range []struct {
		name         string
		info         string
		planned      *clusterResourceData
		current      *clusterResourceData
		wantError    bool
		wantCalls    int
		wantInDetail string
	}{
		{"strimzi on 16.0.3", `{"version":"16.0.3"}`, clusterWithStrimzi(t, "strimzi", true), nil, true, 1, "16.0.3"},
		{"strimzi block on 15.0.9", `{"version":"15.0.9"}`, clusterWithStrimzi(t, "apache-kafka", true), nil, true, 1, "15.0.9"},
		{"strimzi on 16.1.0", `{"version":"16.1.0-SNAPSHOT"}`, clusterWithStrimzi(t, "strimzi", true), nil, false, 1, ""},
		{"version unreadable", "", clusterWithStrimzi(t, "strimzi", true), nil, true, 1, "version failed"},
		{"no strimzi", `{"version":"15.0.9"}`, clusterWithStrimzi(t, "apache-kafka", false), nil, false, 0, ""},
		{"strimzi already in the state", `{"version":"16.0.3"}`, clusterWithStrimzi(t, "strimzi", true), clusterWithStrimzi(t, "strimzi", true), false, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r := &clusterResource{provider: infoPM(t, tc.info, &calls)}
			var current any
			if tc.current != nil {
				current = tc.current
			}
			resp := modifyPlanOf(t, r, tc.planned, current)
			if resp.Diagnostics.HasError() != tc.wantError || calls != tc.wantCalls {
				t.Fatalf("error = %v, GET /info calls = %d; expected error %v, %d calls. Diagnostics: %v",
					resp.Diagnostics.HasError(), calls, tc.wantError, tc.wantCalls, resp.Diagnostics)
			}
			if tc.wantError && !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), tc.wantInDetail) {
				t.Errorf("detail = %q, expected it to name %q", resp.Diagnostics.Errors()[0].Detail(), tc.wantInDetail)
			}
		})
	}
}

func instanceWith(flink, connect bool, connectUrls ...string) *instanceResourceData {
	data := testInstanceData()
	data.FlinkEnabled = types.BoolValue(flink)
	data.ConnectEnabled = types.BoolValue(connect)
	if len(connectUrls) > 0 {
		urls := []attr.Value{}
		for _, u := range connectUrls {
			urls = append(urls, types.StringValue(u))
		}
		data.ConnectUrls = types.ListValueMust(types.StringType, urls)
	}
	return data
}

// Platform Manager 15.0.x ignores flinkEnabled and refuses connectEnabled without connectUrls.
func TestInstancePlanRefusesFlinkAndConnectOnPlatformManager15(t *testing.T) {
	for _, tc := range []struct {
		name       string
		info       string
		planned    *instanceResourceData
		current    *instanceResourceData
		wantErrors int
		wantCalls  int
	}{
		{"flink on 15.0.9", `{"version":"15.0.9"}`, instanceWith(true, false), nil, 1, 1},
		{"connect without urls on 15.0.9", `{"version":"15.0.9"}`, instanceWith(false, true), nil, 1, 1},
		{"both on 15.0.9", `{"version":"15.0.9"}`, instanceWith(true, true), nil, 2, 1},
		{"both on 16.0.3", `{"version":"16.0.3"}`, instanceWith(true, true), nil, 0, 1},
		{"version unreadable", "", instanceWith(true, false), nil, 1, 1},
		{"connect with urls on 15.0.9", `{"version":"15.0.9"}`, instanceWith(false, true, "http://connect:8083"), nil, 0, 0},
		{"flink already in the state", `{"version":"15.0.9"}`, instanceWith(true, false), instanceWith(true, false), 0, 0},
		{"neither", `{"version":"15.0.9"}`, instanceWith(false, false), nil, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r := &instanceResource{provider: infoPM(t, tc.info, &calls)}
			var current any
			if tc.current != nil {
				current = tc.current
			}
			resp := modifyPlanOf(t, r, tc.planned, current)
			if resp.Diagnostics.ErrorsCount() != tc.wantErrors || calls != tc.wantCalls {
				t.Errorf("errors = %d, GET /info calls = %d; expected %d errors, %d calls. Diagnostics: %v",
					resp.Diagnostics.ErrorsCount(), calls, tc.wantErrors, tc.wantCalls, resp.Diagnostics)
			}
		})
	}
}

// The errors name the setting, the version it needs and the version of this Platform Manager.
func TestInstancePlanErrorTextOnPlatformManager15(t *testing.T) {
	calls := 0
	r := &instanceResource{provider: infoPM(t, `{"version":"15.0.9"}`, &calls)}
	resp := modifyPlanOf(t, r, instanceWith(true, true), nil)
	want := []struct{ attribute, summary, detail string }{
		{"flink_enabled", "flink_enabled = true needs Platform Manager 16.0.0 or later",
			"This Platform Manager is 15.0.9. It ignores flink_enabled, so the apply would fail."},
		{"connect_enabled", "connect_enabled = true without connect_urls needs Platform Manager 16.0.0 or later",
			"This Platform Manager is 15.0.9. It needs the connect_urls of Axual Connect when connect_enabled is true."},
	}
	errs := resp.Diagnostics.Errors()
	if len(errs) != len(want) {
		t.Fatalf("errors = %v, expected %d", errs, len(want))
	}
	for i, w := range want {
		d, ok := errs[i].(diag.DiagnosticWithPath)
		if !ok || d.Path().String() != w.attribute || d.Summary() != w.summary || d.Detail() != w.detail {
			t.Errorf("error %d = %v, expected on %s: %q, %q", i, errs[i], w.attribute, w.summary, w.detail)
		}
	}
}

// Platform Manager 15.0.x makes every Instance-Cluster new when the PATCH has instanceClusters, so an
// update sends it only when the Clusters change.
func TestInstanceUpdateSendsClustersOnlyWhenTheyChange(t *testing.T) {
	for name, tc := range map[string]struct {
		planned      []string
		wantClusters bool
	}{
		"same clusters":   {[]string{"c1"}, false},
		"cluster added":   {[]string{"c1", "c2"}, true},
		"cluster swapped": {[]string{"c2"}, true},
	} {
		t.Run(name, func(t *testing.T) {
			var body string
			r := &instanceResource{provider: AxualProvider{client: fakePM(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodPatch {
					b, _ := io.ReadAll(req.Body)
					body = string(b)
				}
				_, _ = w.Write([]byte(`{"uid":"i1","name":"Instance","shortName":"inst","instanceClusters":[{"cluster":{"uid":"c1"}}]}`))
			})}}
			plan := testInstanceData()
			plan.Clusters = stringSetToData(tc.planned)
			plan.Description = types.StringValue("changed")
			planned := stateFor(t, r, plan)
			state := stateFor(t, r, testInstanceData())
			resp := resource.UpdateResponse{State: state}
			r.Update(context.Background(), resource.UpdateRequest{Plan: tfsdk.Plan{Schema: planned.Schema, Raw: planned.Raw}, State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("update: %v", resp.Diagnostics)
			}
			if got := strings.Contains(body, `"instanceClusters"`); got != tc.wantClusters {
				t.Errorf("PATCH has instanceClusters = %v, expected %v: %s", got, tc.wantClusters, body)
			}
		})
	}
}

// Removing MUTUAL_TLS and leaving out ca_certificates plans no certificates: Platform Manager removes
// them, and a plan that keeps the ones of the state fails the apply with an inconsistent result.
func TestInstancePlanOfCaCertificatesWithoutMutualTLS(t *testing.T) {
	pem := stringSetToData([]string{"pem"})
	for name, tc := range map[string]struct {
		configured, methods types.Set
		want                []string
	}{
		"MUTUAL_TLS removed, certificates left out": {types.SetNull(types.StringType), stringSetToData([]string{"SCRAM_SHA_512"}), []string{}},
		"MUTUAL_TLS kept":       {types.SetNull(types.StringType), stringSetToData([]string{"MUTUAL_TLS"}), []string{"pem"}},
		"certificates set":      {pem, stringSetToData([]string{"SCRAM_SHA_512"}), []string{"pem"}},
		"methods not known yet": {types.SetNull(types.StringType), types.SetUnknown(types.StringType), []string{"pem"}},
		"methods left out":      {types.SetNull(types.StringType), types.SetNull(types.StringType), []string{"pem"}},
	} {
		t.Run(name, func(t *testing.T) {
			r := &instanceResource{}
			planned := testInstanceData()
			planned.AuthenticationMethods, planned.CaCertificates = tc.methods, pem
			configured := testInstanceData()
			configured.AuthenticationMethods, configured.CaCertificates = tc.methods, tc.configured
			plan, config := stateFor(t, r, planned), stateFor(t, r, configured)
			req := resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw},
				Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw}, State: emptyState(t, r)}
			resp := resource.ModifyPlanResponse{Plan: req.Plan}
			r.ModifyPlan(context.Background(), req, &resp)
			var got types.Set
			resp.Diagnostics.Append(resp.Plan.GetAttribute(context.Background(), path.Root("ca_certificates"), &got)...)
			if resp.Diagnostics.HasError() || got.IsNull() || !slices.Equal(stringSetFromData(got), tc.want) {
				t.Errorf("ca_certificates = %v, expected %v: %v", got, tc.want, resp.Diagnostics)
			}
		})
	}
}
