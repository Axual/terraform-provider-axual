package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	webclient "axual-webclient"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestDecisionValidateConfig: the configuration sets grant_id (single repository), all four lookup
// attributes (multi-repo), or both (as `terraform plan -generate-config-out` writes them), never a part.
func TestDecisionValidateConfig(t *testing.T) {
	lookups := map[string]any{"application": "Log Scraper", "topic": "t", "environment": "e", "access_type": "CONSUMER"}
	tests := []struct {
		name    string
		set     map[string]any
		wantErr bool
	}{
		{"four lookup attributes", lookups, false},
		{"grant_id", map[string]any{"grant_id": "g"}, false},
		{"grant_id known only at apply", map[string]any{"grant_id": tftypes.UnknownValue}, false},
		{"lookup attribute known only at apply", map[string]any{"application": tftypes.UnknownValue, "topic": "t", "environment": "e", "access_type": "CONSUMER"}, false},
		{"three lookup attributes", map[string]any{"application": "Log Scraper", "topic": "t", "environment": "e"}, true},
		{"neither", map[string]any{}, true},
		{"grant_id and lookup attributes", map[string]any{"grant_id": "g", "application": "Log Scraper", "topic": "t", "environment": "e", "access_type": "CONSUMER"}, false},
		{"grant_id and three lookup attributes", map[string]any{"grant_id": "g", "application": "Log Scraper", "topic": "t", "environment": "e"}, true},
		{"grant_id and one lookup attribute", map[string]any{"grant_id": "g", "topic": "t"}, true},
	}
	r := &grantDecisionResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	objectType := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]tftypes.Value{}
			for name, attrType := range objectType.AttributeTypes {
				values[name] = tftypes.NewValue(attrType, nil)
			}
			values["decision"] = tftypes.NewValue(tftypes.String, decisionApproved)
			for name, v := range tt.set {
				values[name] = tftypes.NewValue(tftypes.String, v)
			}
			req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objectType, values)}}
			var resp resource.ValidateConfigResponse
			r.ValidateConfig(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() != tt.wantErr {
				t.Errorf("error = %v, want %v: %v", resp.Diagnostics.HasError(), tt.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestGrantMatchesLookups: with grant_id and the lookup attributes, both must describe the same grant.
func TestGrantMatchesLookups(t *testing.T) {
	g := webclient.GrantView{Uid: "g1", TopicUid: "t", EnvironmentUid: "e", AccessType: "CONSUMER",
		ApplicationName: "Log Scraper", ApplicationUid: "0123456789abcdef0123456789abcdef"}
	d := func(app, topic, env, access string) grantDecisionData {
		return grantDecisionData{Application: types.StringValue(app), Topic: types.StringValue(topic),
			Environment: types.StringValue(env), AccessType: types.StringValue(access)}
	}
	tests := []struct {
		name string
		d    grantDecisionData
		want bool
	}{
		{"by name", d("Log Scraper", "t", "e", "CONSUMER"), true},
		{"by UID, other case", d("0123456789ABCDEF0123456789ABCDEF", "t", "e", "CONSUMER"), true},
		{"access type in lower case", d("Log Scraper", "t", "e", "consumer"), true},
		{"other application", d("POC Secret", "t", "e", "CONSUMER"), false},
		{"name in other case", d("log scraper", "t", "e", "CONSUMER"), true},
		{"other topic", d("Log Scraper", "t2", "e", "CONSUMER"), false},
		{"other environment", d("Log Scraper", "t", "e2", "CONSUMER"), false},
		{"other access type", d("Log Scraper", "t", "e", "PRODUCER"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := grantMatchesLookups(g, tt.d); got != tt.want {
				t.Errorf("grantMatchesLookups = %v, want %v", got, tt.want)
			}
		})
	}
}

// decisionPM is a fake Platform Manager with the grants of one topic, environment and access type.
// It records the approve and deny calls.
type decisionPM struct {
	mu     sync.Mutex
	grants map[string]map[string]any
	calls  []string
}

const (
	decisionTopic = "70000000000000000000000000000001"
	decisionEnv   = "e0000000000000000000000000000001"
	decisionAppA  = "a0000000000000000000000000000001"
	decisionAppB  = "b0000000000000000000000000000001"
)

func (f *decisionPM) add(uid, status, app string) {
	links := map[string]any{"deny": map[string]any{}}
	if status == "Pending" {
		links["approve"] = map[string]any{}
	}
	f.grants[uid] = map[string]any{"uid": uid, "status": status, "accessType": "CONSUMER", "_links": links,
		"_embedded": map[string]any{"application": map[string]any{"uid": app, "name": "app " + app[:1]},
			"stream": map[string]any{"uid": decisionTopic}, "environment": map[string]any{"uid": decisionEnv}}}
}

func (f *decisionPM) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/application_access_grants/"), "/")
	switch {
	case strings.HasPrefix(r.URL.Path, "/application_access_grants/search/"):
		var list []map[string]any
		for _, g := range f.grants {
			list = append(list, g)
		}
		_ = json.NewEncoder(w).Encode(list)
	case !strings.HasPrefix(r.URL.Path, "/application_access_grants/") || f.grants[parts[0]] == nil:
		w.WriteHeader(http.StatusNotFound)
	case r.Method == http.MethodPut:
		f.calls = append(f.calls, "approve "+parts[0])
		f.grants[parts[0]]["status"] = "Approved"
	case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "deny":
		f.calls = append(f.calls, "deny "+parts[0])
		f.grants[parts[0]]["status"] = map[string]string{"Approved": "Revoked", "Pending": "Rejected"}[f.grants[parts[0]]["status"].(string)]
	default:
		_ = json.NewEncoder(w).Encode(f.grants[parts[0]])
	}
}

// decisionState is an APPROVED decision by application UID that matched grant g1 of application A.
func decisionState() grantDecisionData {
	return grantDecisionData{
		Id:          types.StringValue(decisionTopic + "/" + decisionEnv + "/CONSUMER/" + decisionAppA),
		Application: types.StringValue(decisionAppA), Topic: types.StringValue(decisionTopic),
		Environment: types.StringValue(decisionEnv), AccessType: types.StringValue("CONSUMER"),
		Decision: types.StringValue(decisionApproved), Reason: types.StringNull(), RevokeOnDestroy: types.BoolValue(true),
		GrantId: types.StringValue("g1"), CurrentStatus: types.StringValue("Approved"), ClosedOutside: types.BoolValue(false),
		ApplicationUid: types.StringValue(decisionAppA),
	}
}

// planDecisionFrom runs ModifyPlan for an update of state to application app (unknown: types.StringUnknown()).
func planDecisionFrom(t *testing.T, r *grantDecisionResource, state grantDecisionData, app types.String) (resource.ModifyPlanResponse, grantDecisionData) {
	t.Helper()
	config := grantDecisionData{Id: types.StringNull(), Application: app, Topic: state.Topic, Environment: state.Environment,
		AccessType: state.AccessType, Decision: state.Decision, Reason: types.StringNull(), RevokeOnDestroy: types.BoolNull(),
		GrantId: types.StringNull(), CurrentStatus: types.StringNull(), ClosedOutside: types.BoolNull(), ApplicationUid: types.StringNull()}
	// The proposed plan: the configuration, with the computed values of the state, unknown when the
	// configuration changes (as Terraform proposes them).
	proposed := state
	proposed.Application = app
	if !app.Equal(state.Application) {
		proposed.GrantId, proposed.CurrentStatus, proposed.ClosedOutside, proposed.ApplicationUid =
			types.StringUnknown(), types.StringUnknown(), types.BoolUnknown(), types.StringUnknown()
	}
	plan, cfg, st := stateFor(t, r, proposed), stateFor(t, r, config), stateFor(t, r, state)
	req := resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw},
		Config: tfsdk.Config{Schema: cfg.Schema, Raw: cfg.Raw}, State: st}
	resp := resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, &resp)
	var planned grantDecisionData
	resp.Diagnostics.Append(resp.Plan.Get(context.Background(), &planned)...)
	return resp, planned
}

// TestDecisionPlanKeepsItsActionWhenTheApplicationIsReplaced pins the fix for "Provider produced
// inconsistent final plan". A decision whose application UID is unknown in the plan (the same apply
// replaces the application) is planned as an update with id unknown. The apply plans again with the
// new UID, after the old application and its grant are gone, and that plan must stay an update: it
// approves the new grant, and id follows it. A known UID of another application whose old grant still
// exists is replaced, as before. A UID that finds the same grant changes nothing.
func TestDecisionPlanKeepsItsActionWhenTheApplicationIsReplaced(t *testing.T) {
	t.Parallel() // an approve waits 10 seconds
	newID := decisionTopic + "/" + decisionEnv + "/CONSUMER/" + decisionAppB
	tests := []struct {
		name     string
		app      types.String
		noGrant  bool   // the state has no grant (it never found a request)
		oldGrant string // status of g1; "" when application A and its grant are gone
		replace  bool
		id       string // planned id; "" is unknown
		calls    []string
		status   string
		grantID  string
	}{
		{name: "unknown in the plan", app: types.StringUnknown(), oldGrant: "Approved"},
		{name: "unknown in the plan, no grant in the state", app: types.StringUnknown(), noGrant: true, replace: true},
		{name: "known, old grant gone (the plan made again at apply)", app: types.StringValue(decisionAppB),
			id: newID, calls: []string{"approve g2"}, status: "Approved", grantID: "g2"},
		{name: "known, old grant still there", app: types.StringValue(decisionAppB), oldGrant: "Approved", replace: true, id: newID},
		{name: "known, same grant", app: types.StringValue(decisionAppA), oldGrant: "Approved",
			id: decisionState().Id.ValueString(), status: "Approved", grantID: "g1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel() // an approve waits 10 seconds
			pm := &decisionPM{grants: map[string]map[string]any{}}
			pm.add("g2", "Pending", decisionAppB)
			if test.oldGrant != "" {
				pm.add("g1", test.oldGrant, decisionAppA)
			}
			client := fakePM(t, pm.handler)
			r := &grantDecisionResource{provider: AxualProvider{client: client, grants: newGrantCache(client)}}
			state := decisionState()
			if test.noGrant {
				state.GrantId, state.CurrentStatus = types.StringNull(), types.StringValue(statusNotFound)
			}

			resp, planned := planDecisionFrom(t, r, state, test.app)
			if resp.Diagnostics.HasError() {
				t.Fatalf("plan: %v", resp.Diagnostics)
			}
			if replaced := len(resp.RequiresReplace) > 0; replaced != test.replace {
				t.Fatalf("plan replaces = %t (%v), expected %t", replaced, resp.RequiresReplace, test.replace)
			}
			if got := planned.Id.ValueString(); planned.Id.IsUnknown() != (test.id == "") || got != test.id {
				t.Errorf("planned id = %s, expected %q (empty is unknown)", planned.Id, test.id)
			}
			if test.replace || test.app.IsUnknown() {
				return
			}
			out := emptyState(t, r)
			var diags diag.Diagnostics
			r.apply(context.Background(), planned, state, false, false, grantDecisionPrivate{}, &out, nil, nil, &diags)
			if diags.HasError() {
				t.Fatalf("apply: %v", diags)
			}
			if !slices.Equal(pm.calls, test.calls) {
				t.Errorf("calls = %v, expected %v", pm.calls, test.calls)
			}
			var stored grantDecisionData
			out.Get(context.Background(), &stored)
			if stored.CurrentStatus.ValueString() != test.status || stored.GrantId.ValueString() != test.grantID || stored.Id.ValueString() != test.id {
				t.Errorf("stored status %s on grant %s with id %s, expected %s on %s with id %s",
					stored.CurrentStatus, stored.GrantId, stored.Id, test.status, test.grantID, test.id)
			}
		})
	}
}

// TestDecisionUpdateToAnotherGrantRevokesTheOldOne: an update that the plan made for another grant
// (current_status unknown) revokes the old grant if it is still Approved, as a replace does, and then
// decides on the new one.
func TestDecisionUpdateToAnotherGrantRevokesTheOldOne(t *testing.T) {
	t.Parallel() // an approve waits 10 seconds
	pm := &decisionPM{grants: map[string]map[string]any{}}
	pm.add("g1", "Approved", decisionAppA)
	pm.add("g2", "Pending", decisionAppB)
	client := fakePM(t, pm.handler)
	r := &grantDecisionResource{provider: AxualProvider{client: client, grants: newGrantCache(client)}}
	planned := decisionState()
	planned.Application, planned.Id = types.StringValue(decisionAppB), types.StringUnknown()
	planned.GrantId, planned.CurrentStatus, planned.ClosedOutside, planned.ApplicationUid =
		types.StringUnknown(), types.StringUnknown(), types.BoolUnknown(), types.StringUnknown()
	out := emptyState(t, r)
	var diags diag.Diagnostics
	r.apply(context.Background(), planned, decisionState(), false, false, grantDecisionPrivate{}, &out, nil, nil, &diags)
	if diags.HasError() {
		t.Fatalf("apply: %v", diags)
	}
	if expected := []string{"deny g1", "approve g2"}; !slices.Equal(pm.calls, expected) {
		t.Errorf("calls = %v, expected %v", pm.calls, expected)
	}
}

// TestDecisionIdentityIsMutable: the identity follows id when an update points the block at another
// grant, which the framework allows only with MutableIdentity.
func TestDecisionIdentityIsMutable(t *testing.T) {
	var resp resource.MetadataResponse
	(&grantDecisionResource{}).Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "axual"}, &resp)
	if !resp.ResourceBehavior.MutableIdentity {
		t.Error("MutableIdentity = false, expected true")
	}
}

// TestDecisionSameGrantPlanStillStopsOnAnotherGrant: a plan for the same grant (name to UID of the same
// application) promises no action. When another grant matches at apply, the apply changes nothing.
func TestDecisionSameGrantPlanStillStopsOnAnotherGrant(t *testing.T) {
	pm := &decisionPM{grants: map[string]map[string]any{}}
	pm.add("g1", "Approved", decisionAppA)
	client := fakePM(t, pm.handler)
	r := &grantDecisionResource{provider: AxualProvider{client: client, grants: newGrantCache(client)}}
	state := decisionState()
	state.Application = types.StringValue("app a")
	planned := state
	planned.Application = types.StringValue(decisionAppA)

	// Between plan and apply the application is created again: its new grant g3 is the only one.
	delete(pm.grants, "g1")
	pm.add("g3", "Pending", decisionAppA)
	out := emptyState(t, r)
	var diags diag.Diagnostics
	r.apply(context.Background(), planned, state, false, false, grantDecisionPrivate{}, &out, nil, nil, &diags)
	if !diags.HasError() || !strings.Contains(diags.Errors()[0].Summary(), "The grant changed between plan and apply") {
		t.Errorf("apply diagnostics = %v, expected \"The grant changed between plan and apply\"", diags)
	}
	if len(pm.calls) != 0 {
		t.Errorf("calls = %v, expected none", pm.calls)
	}
}
