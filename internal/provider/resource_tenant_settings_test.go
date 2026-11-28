package provider

import (
	webclient "axual-webclient"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func strPtr(s string) *string { return &s }

func TestTenantSettingsFromAnswer(t *testing.T) {
	tenant := &webclient.TenantResponse{
		Settings: map[string]*string{
			"notificationsEnabled":        strPtr("TRUE"),
			"dataMaskingEnabled":          nil,
			"schemaRolesEnforced":         strPtr("maybe"),
			"iamGroupManagementEnabled":   strPtr("false"),
			"autoIAMGroupCreationEnabled": strPtr("true"),
			"connectRuntimeEnabled":       strPtr("true"),
			"historyRetentionDays":        strPtr("30"),
		},
		SupportedAuthenticationMethods: []webclient.TenantAuthenticationMethod{
			{Rank: 2, Protocol: "SASL", Mechanism: "SCRAM_SHA_512"},
			{Rank: 0, Protocol: "SSL", Mechanism: "MUTUAL_TLS"},
			{Rank: 2, Protocol: "SASL", Mechanism: "OAUTH_BEARER"},
		},
	}
	values, diags := tenantSettingsFromAnswer(tenant, true)
	if diags.HasError() {
		t.Fatal(diags)
	}
	for attrName, want := range map[string]bool{
		"notifications_enabled":        true,  // case does not matter
		"data_masking_enabled":         false, // a null value is the default
		"schema_roles_enforced":        false, // only "true" is true
		"axual_group_management":       true,  // a missing key is its default
		"iam_group_management":         false,
		"automatic_iam_group_creation": false, // no effect while IAM is off
		"connect_runtime_enabled":      true,
		"ksml_runtime_enabled":         false,
	} {
		if got := values.Bools[attrName]; got.IsNull() || got.ValueBool() != want {
			t.Errorf("%s = %v, expected %v", attrName, got, want)
		}
	}
	if values.HistoryRetentionDays.ValueInt64() != 30 {
		t.Errorf("history_retention_days = %v", values.HistoryRetentionDays)
	}
	var methods []string
	values.AuthenticationMethods.ElementsAs(context.Background(), &methods, false)
	if strings.Join(methods, ",") != "MUTUAL_TLS,OAUTH_BEARER,SCRAM_SHA_512" {
		t.Errorf("authentication_methods = %v, expected them by rank and then by name", methods)
	}

	old, _ := tenantSettingsFromAnswer(tenant, false)
	for _, a := range []string{"connect_runtime_enabled", "ksml_runtime_enabled", "flink_runtime_enabled"} {
		if !old.Bools[a].IsNull() {
			t.Errorf("%s = %v on an older Platform Manager, expected null", a, old.Bools[a])
		}
	}
	empty, _ := tenantSettingsFromAnswer(&webclient.TenantResponse{}, true)
	if !empty.HistoryRetentionDays.IsNull() {
		t.Errorf("history_retention_days = %v without the key, expected null", empty.HistoryRetentionDays)
	}
}

func testTenantValues(t *testing.T, bools map[string]types.Bool, days types.Int64, methods ...string) tenantSettingsValues {
	t.Helper()
	values := tenantSettingsValues{Bools: map[string]types.Bool{}, HistoryRetentionDays: days, AuthenticationMethods: types.ListNull(types.StringType)}
	for _, s := range tenantBoolSettings {
		values.Bools[s.attr] = types.BoolNull()
	}
	for k, v := range bools {
		values.Bools[k] = v
	}
	if methods != nil {
		values.AuthenticationMethods, _ = types.ListValueFrom(context.Background(), types.StringType, methods)
	}
	return values
}

func TestComputeTenantChanges(t *testing.T) {
	ctx := context.Background()
	current := testTenantValues(t, map[string]types.Bool{
		"notifications_enabled": types.BoolValue(false), "data_masking_enabled": types.BoolValue(true),
		"iam_group_management": types.BoolValue(true), "axual_group_management": types.BoolValue(true),
		"automatic_iam_group_creation": types.BoolValue(true), "connect_runtime_enabled": types.BoolValue(false),
	}, types.Int64Null(), "MUTUAL_TLS", "SCRAM_SHA_512")

	t.Run("nothing configured", func(t *testing.T) {
		changes, _ := computeTenantChanges(ctx, testTenantValues(t, nil, types.Int64Null()), current)
		if !changes.empty() {
			t.Errorf("expected no change, got %+v", changes)
		}
	})
	t.Run("only what differs", func(t *testing.T) {
		configured := testTenantValues(t, map[string]types.Bool{
			"notifications_enabled": types.BoolValue(true), "data_masking_enabled": types.BoolValue(true),
			"connect_runtime_enabled": types.BoolValue(true),
		}, types.Int64Value(0), "MUTUAL_TLS", "SCRAM_SHA_512")
		changes, _ := computeTenantChanges(ctx, configured, current)
		want := map[string]string{"notificationsEnabled": "true", "connectRuntimeEnabled": "true", "historyRetentionDays": "0"}
		if len(changes.patch.Settings) != len(want) {
			t.Fatalf("settings = %v, expected %v", changes.patch.Settings, want)
		}
		for k, v := range want {
			if changes.patch.Settings[k] != v {
				t.Errorf("settings[%s] = %q, expected %q", k, changes.patch.Settings[k], v)
			}
		}
		if changes.patch.SupportedAuthenticationMethods != nil || changes.group != nil {
			t.Errorf("expected no login method and no group change, got %+v", changes)
		}
	})
	t.Run("login methods in another order get new ranks", func(t *testing.T) {
		changes, _ := computeTenantChanges(ctx, testTenantValues(t, nil, types.Int64Null(), "SCRAM_SHA_512", "MUTUAL_TLS"), current)
		got := changes.patch.SupportedAuthenticationMethods
		if len(got) != 2 || got[0] != (webclient.TenantAuthenticationMethod{Rank: 0, Protocol: "SASL", Mechanism: "SCRAM_SHA_512"}) ||
			got[1] != (webclient.TenantAuthenticationMethod{Rank: 1, Protocol: "SSL", Mechanism: "MUTUAL_TLS"}) {
			t.Errorf("methods = %+v", got)
		}
	})
	t.Run("IAM off sends automatic creation off and leaves the Axual block out", func(t *testing.T) {
		changes, _ := computeTenantChanges(ctx, testTenantValues(t, map[string]types.Bool{"iam_group_management": types.BoolValue(false)}, types.Int64Null()), current)
		g := changes.group
		if g == nil || g.IamGroupManagement == nil || g.IamGroupManagement.Enabled || g.IamGroupManagement.AutomaticGroupCreationEnabled == nil ||
			*g.IamGroupManagement.AutomaticGroupCreationEnabled || g.AxualGroupManagement != nil {
			t.Fatalf("group = %+v, expected only the IAM block, with automatic creation off", g)
		}
		if len(changes.patch.Settings) != 0 {
			t.Errorf("group settings must not go through the PATCH: %v", changes.patch.Settings)
		}
	})
	t.Run("only the Axual setting sends only the Axual block", func(t *testing.T) {
		changes, _ := computeTenantChanges(ctx, testTenantValues(t, map[string]types.Bool{"axual_group_management": types.BoolValue(false)}, types.Int64Null()), current)
		g := changes.group
		if g == nil || g.IamGroupManagement != nil || g.AxualGroupManagement == nil || g.AxualGroupManagement.Enabled {
			t.Fatalf("group = %+v, expected only the Axual block", g)
		}
	})
	t.Run("a group setting equal to the current value sends nothing", func(t *testing.T) {
		changes, _ := computeTenantChanges(ctx, testTenantValues(t, map[string]types.Bool{"axual_group_management": types.BoolValue(true)}, types.Int64Null()), current)
		if changes.group != nil {
			t.Errorf("group = %+v, expected nil", changes.group)
		}
	})
}

func TestPlatformManagerMessage(t *testing.T) {
	for body, want := range map[string]string{
		// The answers of Platform Manager, as seen on master 3d126ddaa and 860eb3f19.
		`{"detail":"History retention days (999) cannot exceed the platform default (365)","status":400,"message":"Bad request","errors":["History retention days (999) cannot exceed the platform default (365)"],"subErrors":[{"field":"settings","errors":["x"]}]}`: "History retention days (999) cannot exceed the platform default (365) (status 400)",
		`{"detail":"At least one group management type must be enabled","fieldErrors":[],"errors":["At least one group management type must be enabled"]}`:                                                                                                             "At least one group management type must be enabled (status 400)",
		`{"message":"Bad request","errors":["first","second"]}`: "first second (status 400)",
		`{"message":"Bad request"}`:                             "Bad request (status 400)",
		`not json`:                                              "status: 400, body: not json",
	} {
		if got := platformManagerMessage(&webclient.HTTPError{StatusCode: 400, Body: body}); got != want {
			t.Errorf("message of %s = %q, expected %q", body, got, want)
		}
	}
}

// fakeTenantPM is a Platform Manager with one tenant. It stores what the PATCH and the group-management call send,
// like Platform Manager does: settings are merged, the login methods are replaced.
type fakeTenantPM struct {
	mu       sync.Mutex
	version  string
	uid      string
	settings map[string]*string
	methods  []webclient.TenantAuthenticationMethod
	patches  []string
	groups   []string
	// conflicts and lostLocks are how many PATCH calls answer 409, and then 403 with a lock error, before one succeeds.
	conflicts int
	lostLocks int
}

func newFakeTenantPM(version string) *fakeTenantPM {
	return &fakeTenantPM{version: version, uid: "t1", settings: map[string]*string{
		"iamGroupManagementEnabled": strPtr("false"), "axualGroupManagementEnabled": strPtr("true"),
	}, methods: []webclient.TenantAuthenticationMethod{{Rank: 0, Protocol: "SSL", Mechanism: "MUTUAL_TLS"}}}
}

func (f *fakeTenantPM) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/info":
		_ = json.NewEncoder(w).Encode(map[string]string{"version": f.version})
	case r.Method == http.MethodGet && r.URL.Path == "/tenant":
		_ = json.NewEncoder(w).Encode(webclient.TenantResponse{Uid: f.uid, Name: "Tenant", ShortName: "tenant", Settings: f.settings, SupportedAuthenticationMethods: f.methods})
	case r.Method == http.MethodPatch && r.URL.Path == "/tenants/"+f.uid:
		if f.conflicts > 0 {
			f.conflicts--
			w.WriteHeader(http.StatusConflict)
			return
		}
		if f.lostLocks > 0 {
			// A lost lock of the tenant row: Platform Manager answers 403, not 409.
			f.lostLocks--
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`Request processing failed: org.springframework.orm.ObjectOptimisticLockingFailureException: Unexpected row count`))
			return
		}
		f.patches = append(f.patches, string(body))
		var patch webclient.TenantSettingsPatch
		_ = json.Unmarshal(body, &patch)
		for k, v := range patch.Settings {
			f.settings[k] = strPtr(v)
		}
		if patch.SupportedAuthenticationMethods != nil {
			f.methods = patch.SupportedAuthenticationMethods
		}
		_, _ = w.Write([]byte("{}"))
	case r.Method == http.MethodPost && r.URL.Path == "/tenant/settings/group-management":
		f.groups = append(f.groups, string(body))
		var g webclient.GroupManagementSettings
		_ = json.Unmarshal(body, &g)
		if g.IamGroupManagement != nil {
			f.settings["iamGroupManagementEnabled"] = strPtr(boolText(g.IamGroupManagement.Enabled))
			f.settings["autoIAMGroupCreationEnabled"] = strPtr(boolText(*g.IamGroupManagement.AutomaticGroupCreationEnabled))
		}
		if g.AxualGroupManagement != nil {
			f.settings["axualGroupManagementEnabled"] = strPtr(boolText(g.AxualGroupManagement.Enabled))
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func boolText(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func newTenantSettingsTestResource(t *testing.T, pm *fakeTenantPM) *tenantSettingsResource {
	t.Helper()
	return &tenantSettingsResource{provider: AxualProvider{client: fakePM(t, pm.handler)}}
}

// tenantSettingsRaw builds a configuration, plan or state of axual_tenant_settings. Attributes that are not in
// values are null.
func tenantSettingsRaw(t *testing.T, r *tenantSettingsResource, values map[string]attr.Value) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	if values == nil {
		return state
	}
	for _, a := range []string{"id", "authentication_methods", "history_retention_days"} {
		if _, ok := values[a]; !ok {
			switch a {
			case "id":
				values[a] = types.StringNull()
			case "authentication_methods":
				values[a] = types.ListNull(types.StringType)
			default:
				values[a] = types.Int64Null()
			}
		}
	}
	for _, s := range tenantBoolSettings {
		if _, ok := values[s.attr]; !ok {
			values[s.attr] = types.BoolNull()
		}
	}
	for name, v := range values {
		if diags := state.SetAttribute(ctx, path.Root(name), v); diags.HasError() {
			t.Fatalf("set %s: %v", name, diags)
		}
	}
	return state
}

func TestTenantSettingsCreateSendsOnlyTheConfiguredChanges(t *testing.T) {
	ctx := context.Background()
	pm := newFakeTenantPM("16.1.0-SNAPSHOT")
	pm.conflicts, pm.lostLocks = 2, 1 // the PATCH is retried after both
	tenantWriteDelay = 0
	r := newTenantSettingsTestResource(t, pm)
	methods, _ := types.ListValueFrom(ctx, types.StringType, []string{"MUTUAL_TLS", "SCRAM_SHA_512"})
	config := tenantSettingsRaw(t, r, map[string]attr.Value{
		"authentication_methods":  methods,
		"connect_runtime_enabled": types.BoolValue(true),
		"iam_group_management":    types.BoolValue(true),
		"axual_group_management":  types.BoolValue(true), // already true: no change on its own
	})
	resp := resource.CreateResponse{State: tenantSettingsRaw(t, r, nil)}
	r.Create(ctx, resource.CreateRequest{Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw}}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if len(pm.patches) != 1 || pm.patches[0] != `{"settings":{"connectRuntimeEnabled":"true"},"supportedAuthenticationMethods":[{"rank":0,"protocol":"SSL","mechanism":"MUTUAL_TLS"},{"rank":1,"protocol":"SASL","mechanism":"SCRAM_SHA_512"}]}` {
		t.Errorf("patches = %v", pm.patches)
	}
	if len(pm.groups) != 1 || pm.groups[0] != `{"iamGroupManagement":{"enabled":true,"automaticGroupCreationEnabled":false}}` {
		t.Errorf("group calls = %v", pm.groups)
	}
	var id types.String
	var notifications, connect types.Bool
	resp.State.GetAttribute(ctx, path.Root("id"), &id)
	resp.State.GetAttribute(ctx, path.Root("notifications_enabled"), &notifications)
	resp.State.GetAttribute(ctx, path.Root("connect_runtime_enabled"), &connect)
	if id.ValueString() != "t1" || notifications.IsNull() || notifications.ValueBool() || !connect.ValueBool() {
		t.Errorf("state: id %v, notifications_enabled %v, connect_runtime_enabled %v", id, notifications, connect)
	}

	// A second apply of the same configuration sends nothing.
	pm.patches, pm.groups = nil, nil
	r.Update(ctx, resource.UpdateRequest{Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw}}, &resource.UpdateResponse{State: resp.State})
	if len(pm.patches)+len(pm.groups) != 0 {
		t.Errorf("second apply sent %v %v", pm.patches, pm.groups)
	}
}

func modifyTenantPlan(t *testing.T, r *tenantSettingsResource, config, plan, state tfsdk.State) resource.ModifyPlanResponse {
	t.Helper()
	resp := resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}
	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: config.Schema, Raw: config.Raw},
		Plan:   tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw},
		State:  state,
	}, &resp)
	return resp
}

func TestTenantSettingsModifyPlan(t *testing.T) {
	ctx := context.Background()
	t.Run("destroy warns that nothing changes", func(t *testing.T) {
		r := newTenantSettingsTestResource(t, newFakeTenantPM("16.0.5"))
		null := tenantSettingsRaw(t, r, nil)
		resp := modifyTenantPlan(t, r, null, null, tenantSettingsRaw(t, r, map[string]attr.Value{"id": types.StringValue("t1")}))
		if resp.Diagnostics.WarningsCount() != 1 || resp.Diagnostics.HasError() {
			t.Errorf("diagnostics = %v", resp.Diagnostics)
		}
	})
	for version, wantError := range map[string]bool{"16.0.5": true, "15.0.11": true, "16.1.0-SNAPSHOT": false, "17.0.0": false} {
		t.Run("runtime switch on "+version, func(t *testing.T) {
			r := newTenantSettingsTestResource(t, newFakeTenantPM(version))
			values := map[string]attr.Value{"ksml_runtime_enabled": types.BoolValue(true)}
			config := tenantSettingsRaw(t, r, values)
			resp := modifyTenantPlan(t, r, config, config, tenantSettingsRaw(t, r, nil))
			if resp.Diagnostics.HasError() != wantError {
				t.Errorf("error = %v, expected %v: %v", resp.Diagnostics.HasError(), wantError, resp.Diagnostics)
			}
		})
	}
	t.Run("other settings work on an older Platform Manager", func(t *testing.T) {
		r := newTenantSettingsTestResource(t, newFakeTenantPM("15.0.11"))
		config := tenantSettingsRaw(t, r, map[string]attr.Value{"notifications_enabled": types.BoolValue(true)})
		if resp := modifyTenantPlan(t, r, config, config, tenantSettingsRaw(t, r, nil)); resp.Diagnostics.HasError() {
			t.Error(resp.Diagnostics)
		}
	})
	t.Run("both group modes off is refused", func(t *testing.T) {
		r := newTenantSettingsTestResource(t, newFakeTenantPM("16.0.5"))
		config := tenantSettingsRaw(t, r, map[string]attr.Value{"iam_group_management": types.BoolValue(false)})
		plan := tenantSettingsRaw(t, r, map[string]attr.Value{"iam_group_management": types.BoolValue(false), "axual_group_management": types.BoolValue(false)})
		if resp := modifyTenantPlan(t, r, config, plan, plan); !resp.Diagnostics.HasError() {
			t.Error("expected an error")
		}
	})
	t.Run("automatic creation without IAM is refused", func(t *testing.T) {
		r := newTenantSettingsTestResource(t, newFakeTenantPM("16.0.5"))
		values := map[string]attr.Value{"iam_group_management": types.BoolValue(false), "automatic_iam_group_creation": types.BoolValue(true)}
		config := tenantSettingsRaw(t, r, values)
		if resp := modifyTenantPlan(t, r, config, config, tenantSettingsRaw(t, r, nil)); !resp.Diagnostics.HasError() {
			t.Error("expected an error")
		}
	})
	t.Run("an update plans the settings it does not manage as unknown", func(t *testing.T) {
		r := newTenantSettingsTestResource(t, newFakeTenantPM("16.0.5"))
		state := tenantSettingsRaw(t, r, map[string]attr.Value{"id": types.StringValue("t1"),
			"notifications_enabled": types.BoolValue(false), "data_masking_enabled": types.BoolValue(true), "history_retention_days": types.Int64Value(30)})
		config := tenantSettingsRaw(t, r, map[string]attr.Value{"notifications_enabled": types.BoolValue(true)})
		plan := tenantSettingsRaw(t, r, map[string]attr.Value{"id": types.StringValue("t1"),
			"notifications_enabled": types.BoolValue(true), "data_masking_enabled": types.BoolValue(true), "history_retention_days": types.Int64Value(30)})
		resp := modifyTenantPlan(t, r, config, plan, state)
		var notifications, masking types.Bool
		var days types.Int64
		resp.Plan.GetAttribute(ctx, path.Root("notifications_enabled"), &notifications)
		resp.Plan.GetAttribute(ctx, path.Root("data_masking_enabled"), &masking)
		resp.Plan.GetAttribute(ctx, path.Root("history_retention_days"), &days)
		if resp.Diagnostics.HasError() || !notifications.ValueBool() || !masking.IsUnknown() || !days.IsUnknown() {
			t.Errorf("notifications_enabled %v, data_masking_enabled %v, history_retention_days %v, diagnostics %v", notifications, masking, days, resp.Diagnostics)
		}
		// Without a change, nothing is unknown.
		same := modifyTenantPlan(t, r, tenantSettingsRaw(t, r, map[string]attr.Value{"notifications_enabled": types.BoolValue(false)}), state, state)
		same.Plan.GetAttribute(ctx, path.Root("data_masking_enabled"), &masking)
		if masking.IsUnknown() {
			t.Error("a plan without a change marks data_masking_enabled unknown")
		}
	})
	t.Run("IAM off plans automatic creation off", func(t *testing.T) {
		r := newTenantSettingsTestResource(t, newFakeTenantPM("16.0.5"))
		config := tenantSettingsRaw(t, r, map[string]attr.Value{"iam_group_management": types.BoolValue(false)})
		plan := tenantSettingsRaw(t, r, map[string]attr.Value{"id": types.StringValue("t1"), "iam_group_management": types.BoolValue(false),
			"axual_group_management": types.BoolValue(true), "automatic_iam_group_creation": types.BoolValue(true)})
		resp := modifyTenantPlan(t, r, config, plan, plan)
		var auto types.Bool
		resp.Plan.GetAttribute(ctx, path.Root("automatic_iam_group_creation"), &auto)
		if resp.Diagnostics.HasError() || auto.IsNull() || auto.ValueBool() {
			t.Errorf("automatic_iam_group_creation = %v, diagnostics %v", auto, resp.Diagnostics)
		}
	})
}

func TestTenantSettingsKeepStateWhenUnset(t *testing.T) {
	ctx := context.Background()
	r := newTenantSettingsTestResource(t, newFakeTenantPM("16.0.5"))
	state := tenantSettingsRaw(t, r, map[string]attr.Value{"id": types.StringValue("t1")})
	resp := planmodifierBool(ctx, state, types.BoolNull(), types.BoolUnknown())
	if !resp.IsNull() {
		t.Errorf("an unset attribute with a null state is planned as %v, expected null", resp)
	}
	if got := planmodifierBool(ctx, tenantSettingsRaw(t, r, nil), types.BoolNull(), types.BoolUnknown()); !got.IsUnknown() {
		t.Errorf("on create an unset attribute is planned as %v, expected unknown", got)
	}
	if got := planmodifierBool(ctx, state, types.BoolValue(true), types.BoolValue(true)); !got.ValueBool() {
		t.Errorf("a set attribute is planned as %v", got)
	}
}

func planmodifierBool(ctx context.Context, state tfsdk.State, config, plan types.Bool) types.Bool {
	req := planmodifierBoolRequest(state, config, plan)
	resp := planmodifierBoolResponse(plan)
	keepStateWhenUnsetBool{}.PlanModifyBool(ctx, req, resp)
	return resp.PlanValue
}

func TestTenantSettingsImportAndRead(t *testing.T) {
	ctx := context.Background()
	pm := newFakeTenantPM("16.0.5")
	r := newTenantSettingsTestResource(t, pm)
	for id, wantError := range map[string]bool{"t1": false, "tenant": false, "TENANT": false, "other": true} {
		resp := resource.ImportStateResponse{State: tenantSettingsRaw(t, r, nil)}
		r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)
		if resp.Diagnostics.HasError() != wantError {
			t.Errorf("import %q: error = %v, expected %v: %v", id, resp.Diagnostics.HasError(), wantError, resp.Diagnostics)
		}
	}
	state := tenantSettingsRaw(t, r, map[string]attr.Value{"id": types.StringValue("t2")})
	resp := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Error("reading the settings of another tenant must fail")
	}
}

func TestTenantDataSource(t *testing.T) {
	ctx := context.Background()
	pm := newFakeTenantPM("16.0.5")
	d := &tenantDataSource{provider: AxualProvider{client: fakePM(t, pm.handler)}}
	schemaResp := datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}}
	d.Read(ctx, datasource.ReadRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var shortName types.String
	var connect, axual types.Bool
	resp.State.GetAttribute(ctx, path.Root("short_name"), &shortName)
	resp.State.GetAttribute(ctx, path.Root("connect_runtime_enabled"), &connect)
	resp.State.GetAttribute(ctx, path.Root("axual_group_management"), &axual)
	if shortName.ValueString() != "tenant" || !connect.IsNull() || !axual.ValueBool() {
		t.Errorf("short_name %v, connect_runtime_enabled %v (expected null on 16.0.5), axual_group_management %v", shortName, connect, axual)
	}
}

func planmodifierBoolRequest(state tfsdk.State, config, plan types.Bool) planmodifier.BoolRequest {
	return planmodifier.BoolRequest{State: state, ConfigValue: config, PlanValue: plan, StateValue: types.BoolNull()}
}

func planmodifierBoolResponse(plan types.Bool) *planmodifier.BoolResponse {
	return &planmodifier.BoolResponse{PlanValue: plan}
}

// A login without TENANT_ADMIN gets 403 "Access Denied": the message names the role.
func TestPlatformManagerMessageForbidden(t *testing.T) {
	got := platformManagerMessage(&webclient.HTTPError{StatusCode: 403, Body: `{"detail":"Access Denied","status":403}`})
	if got != "Access Denied. Changing tenant settings needs the TENANT_ADMIN role (status 403)" {
		t.Errorf("message = %q", got)
	}
}
