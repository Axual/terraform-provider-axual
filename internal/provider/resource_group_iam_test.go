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

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// groupPM is a fake Platform Manager that records the group calls it gets.
type groupPM struct {
	mu     sync.Mutex
	calls  []string
	bodies map[string]map[string]any
	// refuse makes the call with this "METHOD path" answer 400.
	refuse string
}

func (p *groupPM) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		body := map[string]any{}
		_ = json.Unmarshal(raw, &body)
		p.mu.Lock()
		p.calls = append(p.calls, key)
		if p.bodies == nil {
			p.bodies = map[string]map[string]any{}
		}
		p.bodies[key] = body
		p.mu.Unlock()

		if key == p.refuse {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"detail":"IAM Group Management is disabled for this tenant"}`))
			return
		}
		answer := map[string]any{"uid": "g1", "name": "team"}
		if ref, ok := body["iamReference"].(string); ok && ref != "" {
			answer["iamReference"] = ref
		}
		if r.Method == http.MethodGet {
			answer["iamReference"] = "stored-ref"
			answer["_embedded"] = map[string]any{"members": []any{map[string]any{"uid": "m1"}}, "managers": []any{map[string]any{"uid": "m1"}}}
		}
		if strings.HasSuffix(r.URL.Path, "/externalize") && r.Method == http.MethodDelete {
			delete(answer, "iamReference")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/groups" {
			w.WriteHeader(http.StatusCreated)
		}
		_ = json.NewEncoder(w).Encode(answer)
	}
}

func groupData(iamReference string, members ...string) groupResourceData {
	data := groupResourceData{
		Id:             types.StringValue("g1"),
		Name:           types.StringValue("team"),
		EmailAddress:   types.StringNull(),
		PhoneNumber:    types.StringNull(),
		Members:        types.SetNull(types.StringType),
		Managers:       types.SetNull(types.StringType),
		IamReference:   types.StringNull(),
		StoredMembers:  types.SetNull(types.StringType),
		StoredManagers: types.SetNull(types.StringType),
	}
	if iamReference != "" {
		data.IamReference = types.StringValue(iamReference)
	}
	if len(members) > 0 {
		data.Members = uidSet(members...)
	}
	return data
}

func updateGroup(t *testing.T, pm *groupPM, state, plan groupResourceData) (resource.UpdateResponse, groupResourceData) {
	t.Helper()
	r := &groupResource{provider: AxualProvider{client: fakePM(t, pm.handler(t))}}
	oldState := stateFor(t, r, state)
	newPlan := tfsdk.Plan{Schema: oldState.Schema, Raw: stateFor(t, r, plan).Raw}
	resp := resource.UpdateResponse{State: oldState}
	r.Update(context.Background(), resource.UpdateRequest{Plan: newPlan, State: oldState}, &resp)
	var saved groupResourceData
	if !resp.Diagnostics.HasError() {
		resp.State.Get(context.Background(), &saved)
	}
	return resp, saved
}

func hasKey(body map[string]any, key string) bool {
	_, ok := body[key]
	return ok
}

func TestGroupRequest_iamManagedGroupSendsTheReferenceAndNoMembers(t *testing.T) {
	data := groupData("engineering", "m1")

	request, err := createGroupRequestFromData(context.Background(), &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, _ := json.Marshal(request)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)

	if body["iamReference"] != "engineering" {
		t.Errorf("iamReference = %v, expected engineering", body["iamReference"])
	}
	if hasKey(body, "members") || hasKey(body, "managers") {
		t.Errorf("body %s has members or managers, expected neither for an IAM-managed group", raw)
	}
}

func TestGroupRequest_axualManagedGroupSendsMembersAndNoReference(t *testing.T) {
	data := groupData("")

	request, err := createGroupRequestFromData(context.Background(), &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, _ := json.Marshal(request)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)

	if hasKey(body, "iamReference") {
		t.Errorf("body %s has an iamReference, expected none", raw)
	}
	if members, ok := body["members"].([]any); !ok || len(members) != 0 {
		t.Errorf("members = %v, expected an empty list so that members can be cleared", body["members"])
	}
}

func TestGroupCreate_sendsTheIamReference(t *testing.T) {
	pm := &groupPM{}
	r := &groupResource{provider: AxualProvider{client: fakePM(t, pm.handler(t))}}
	plan := tfsdk.Plan{Schema: stateFor(t, r, groupData("")).Schema, Raw: stateFor(t, r, groupData("engineering", "m1")).Raw}
	config := tfsdk.Config{Schema: plan.Schema, Raw: plan.Raw}
	resp := resource.CreateResponse{State: emptyState(t, r)}

	r.Create(context.Background(), resource.CreateRequest{Config: config, Plan: plan}, &resp)

	var saved groupResourceData
	resp.State.Get(context.Background(), &saved)
	body := pm.bodies["POST /groups"]
	if resp.Diagnostics.HasError() || body["iamReference"] != "engineering" || hasKey(body, "members") {
		t.Errorf("create = %v, body %v; expected the reference sent and no members", resp.Diagnostics, body)
	}
	if saved.IamReference.ValueString() != "engineering" || saved.Members.IsNull() {
		t.Errorf("saved reference %q, members %s; expected the reference and the configured members kept", saved.IamReference.ValueString(), saved.Members)
	}
}

func TestGroupUpdate_addingAReferenceExternalizesFirst(t *testing.T) {
	pm := &groupPM{}

	resp, saved := updateGroup(t, pm, groupData(""), groupData("engineering", "m1"))

	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}
	if got := strings.Join(pm.calls, ", "); got != "POST /groups/g1/externalize, PATCH /groups/g1" {
		t.Errorf("calls = %s, expected externalize and then a PATCH", got)
	}
	if pm.bodies["POST /groups/g1/externalize"]["iamReference"] != "engineering" {
		t.Errorf("externalize body %v, expected the reference", pm.bodies["POST /groups/g1/externalize"])
	}
	if hasKey(pm.bodies["PATCH /groups/g1"], "members") {
		t.Errorf("PATCH body %v has members, expected none for an IAM-managed group", pm.bodies["PATCH /groups/g1"])
	}
	if saved.IamReference.ValueString() != "engineering" {
		t.Errorf("saved reference = %q, expected engineering", saved.IamReference.ValueString())
	}
}

func TestGroupUpdate_removingTheReferenceInternalizesFirst(t *testing.T) {
	pm := &groupPM{}

	resp, saved := updateGroup(t, pm, groupData("engineering"), groupData("", "m1"))

	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}
	if got := strings.Join(pm.calls, ", "); got != "DELETE /groups/g1/externalize, PATCH /groups/g1" {
		t.Errorf("calls = %s, expected internalize and then a PATCH", got)
	}
	if members, ok := pm.bodies["PATCH /groups/g1"]["members"].([]any); !ok || len(members) != 1 {
		t.Errorf("PATCH body %v, expected the members back under Terraform", pm.bodies["PATCH /groups/g1"])
	}
	if !saved.IamReference.IsNull() {
		t.Errorf("saved reference = %s, expected null", saved.IamReference)
	}
}

// The state of an IAM-managed group has no members, so the plan cannot show the lists that Platform
// Manager stores. Unlinking must not clear them unseen: the next plan shows them.
func TestGroupUpdate_unlinkingWithoutMembersKeepsTheStoredLists(t *testing.T) {
	pm := &groupPM{}

	resp, saved := updateGroup(t, pm, groupData("engineering"), groupData(""))

	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}
	body := pm.bodies["PATCH /groups/g1"]
	if hasKey(body, "members") || hasKey(body, "managers") {
		t.Errorf("PATCH body %v has members or managers, expected neither so that Platform Manager keeps the stored lists", body)
	}
	if !saved.Members.IsNull() || !saved.Managers.IsNull() {
		t.Errorf("saved members = %s, managers = %s, expected null for both (they are not in the configuration)", saved.Members, saved.Managers)
	}
}

func TestGroupUpdate_unlinkingWithOnlyMembersKeepsTheStoredManagers(t *testing.T) {
	pm := &groupPM{}

	resp, saved := updateGroup(t, pm, groupData("engineering"), groupData("", "m1"))

	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}
	body := pm.bodies["PATCH /groups/g1"]
	if !hasKey(body, "members") || hasKey(body, "managers") {
		t.Errorf("PATCH body %v, expected the members and no managers", body)
	}
	if !saved.Managers.IsNull() {
		t.Errorf("saved managers = %s, expected null (they are not in the configuration)", saved.Managers)
	}
}

func TestGroupUpdate_changingTheReferenceIsOnePatch(t *testing.T) {
	pm := &groupPM{}

	resp, saved := updateGroup(t, pm, groupData("old-ref"), groupData("new-ref"))

	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}
	if got := strings.Join(pm.calls, ", "); got != "PATCH /groups/g1" {
		t.Errorf("calls = %s, expected only a PATCH", got)
	}
	if saved.IamReference.ValueString() != "new-ref" {
		t.Errorf("saved reference = %q, expected new-ref", saved.IamReference.ValueString())
	}
}

func TestGroupUpdate_aRefusedLinkStopsBeforeAnyOtherChange(t *testing.T) {
	pm := &groupPM{refuse: "POST /groups/g1/externalize"}

	resp, _ := updateGroup(t, pm, groupData(""), groupData("engineering"))

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when Platform Manager refuses to link the group")
	}
	if got := strings.Join(pm.calls, ", "); got != "POST /groups/g1/externalize" {
		t.Errorf("calls = %s, expected nothing after the refused externalize", got)
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "IAM Group Management is disabled") {
		t.Errorf("error %q does not carry the platform message", resp.Diagnostics.Errors()[0].Detail())
	}
}

func TestGroupRead_iamManagedGroupKeepsItsMembersOutOfState(t *testing.T) {
	pm := &groupPM{}
	r := &groupResource{provider: AxualProvider{client: fakePM(t, pm.handler(t))}}
	state := stateFor(t, r, groupData("stored-ref"))
	resp := resource.ReadResponse{State: state}

	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

	var saved groupResourceData
	resp.State.Get(context.Background(), &saved)
	if saved.IamReference.ValueString() != "stored-ref" || !saved.Members.IsNull() || !saved.Managers.IsNull() {
		t.Errorf("saved reference %q, members %s, managers %s; expected the reference, and no members or managers",
			saved.IamReference.ValueString(), saved.Members, saved.Managers)
	}
	if !saved.StoredMembers.Equal(uidSet("m1")) || !saved.StoredManagers.Equal(uidSet("m1")) {
		t.Errorf("stored members %s, stored managers %s; expected the stored lists of the group", saved.StoredMembers, saved.StoredManagers)
	}
}

func TestGroupResponseMapping_anAxualManagedGroupHasNoStoredLists(t *testing.T) {
	group := groupDataFromResponse(`{"uid":"g1","name":"team","_embedded":{"members":[{"uid":"m1"}]}}`)

	if !group.StoredMembers.IsNull() || !group.StoredManagers.IsNull() {
		t.Errorf("stored members %s, stored managers %s; expected none for an Axual-managed group", group.StoredMembers, group.StoredManagers)
	}
}

func TestGroupModifyPlan_keepsTheStoredListsWhileTheLinkStays(t *testing.T) {
	r := &groupResource{}
	stateData := groupData("engineering")
	stateData.StoredMembers = uidSet("m1")
	state := stateFor(t, r, stateData)
	planData := groupData("engineering")
	planData.StoredMembers = types.SetUnknown(types.StringType)
	plan := stateFor(t, r, planData)
	resp := resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}

	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: resp.Plan, State: state}, &resp)

	var planned groupResourceData
	resp.Plan.Get(context.Background(), &planned)
	if !planned.StoredMembers.Equal(uidSet("m1")) {
		t.Errorf("planned stored members %s, expected the ones in the state", planned.StoredMembers)
	}
}

func TestGroupResponseMapping_aBlankReferenceMeansAxualManaged(t *testing.T) {
	var group = groupDataFromResponse(`{"uid":"g1","name":"team","iamReference":"   ","_embedded":{"members":[{"uid":"m1"}]}}`)

	if !group.IamReference.IsNull() || group.Members.IsNull() {
		t.Errorf("reference %s, members %s; expected no reference and the members of an Axual-managed group", group.IamReference, group.Members)
	}
}

func groupDataFromResponse(body string) groupResourceData {
	var response webclient.GroupResponse
	_ = json.Unmarshal([]byte(body), &response)
	data := groupData("")
	mapGroupResponseToData(context.Background(), &data, &response)
	return data
}

func TestGroupModifyPlan_refusesMembersOfAnIamManagedGroup(t *testing.T) {
	r := &groupResource{}
	plan := stateFor(t, r, groupData("engineering", "m1"))
	resp := resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}

	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}, State: emptyState(t, r)}, &resp)

	if resp.Diagnostics.ErrorsCount() != 1 || resp.Diagnostics.Errors()[0].Summary() != "Members and managers not supported" {
		t.Errorf("diagnostics = %v, expected one error about members of an IAM-managed group", resp.Diagnostics)
	}
}

func TestGroupModifyPlan_warnsWhenRemovingTheReferenceInternalizes(t *testing.T) {
	r := &groupResource{}
	state := stateFor(t, r, groupData("engineering"))
	plan := stateFor(t, r, groupData(""))
	resp := resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}

	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}, State: state}, &resp)

	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 || resp.Diagnostics.Warnings()[0].Summary() != "The group becomes Axual-managed again" {
		t.Errorf("diagnostics = %v, expected one warning about becoming Axual-managed", resp.Diagnostics)
	}
}

func TestGroupModifyPlan_warnsWhenLinkingAGroupThatHasStoredMembers(t *testing.T) {
	r := &groupResource{}
	state := stateFor(t, r, groupData("", "m1"))
	plan := stateFor(t, r, groupData("engineering"))
	resp := resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}

	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}, State: state}, &resp)

	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 || resp.Diagnostics.Warnings()[0].Summary() != "Stored members and managers stay in Platform Manager" {
		t.Errorf("diagnostics = %v, expected one warning that the stored members stay", resp.Diagnostics)
	}
}

func TestGroupModifyPlan_isQuietWhenLinkingAGroupWithoutMembers(t *testing.T) {
	r := &groupResource{}
	state := stateFor(t, r, groupData(""))
	plan := stateFor(t, r, groupData("engineering"))
	resp := resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}

	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}, State: state}, &resp)

	if len(resp.Diagnostics) != 0 {
		t.Errorf("diagnostics = %v, expected none", resp.Diagnostics)
	}
}

func TestGroupModifyPlan_isQuietForAnOrdinaryGroup(t *testing.T) {
	r := &groupResource{}
	state := stateFor(t, r, groupData("", "m1"))
	plan := stateFor(t, r, groupData("", "m1", "m2"))
	resp := resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}

	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}, State: state}, &resp)

	if len(resp.Diagnostics) != 0 {
		t.Errorf("diagnostics = %v, expected none", resp.Diagnostics)
	}
}

func TestIamReferenceValidation(t *testing.T) {
	var resp resource.SchemaResponse
	(&groupResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	attribute := resp.Schema.Attributes["iam_reference"].(schema.StringAttribute)

	cases := map[string]bool{
		"b0a18de3-fb51-4a8d-8196-788543828ce1": true,
		"data-team":                            true,
		"Data Team":                            true,
		"\u00fcn\u00ef-team":                   true,
		"a":                                    true,
		strings.Repeat("a", 255):               true,
		"":                                     false,
		"   ":                                  false,
		" data-team":                           false,
		"data-team ":                           false,
		"ops,team":                             false,
		",":                                    false,
		"a\nb":                                 false,
		"a\tb":                                 false,
		"\u00a0a":                              false,
		"a\u00a0":                              false,
		"a\u00a0b":                             true,
		strings.Repeat("a", 256):               false,
	}
	for value, valid := range cases {
		var vr validator.StringResponse
		for _, v := range attribute.Validators {
			v.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("iam_reference"), ConfigValue: types.StringValue(value)}, &vr)
		}
		if vr.Diagnostics.HasError() == valid {
			t.Errorf("iam_reference %q: valid=%v but the validators answered error=%v", value, valid, vr.Diagnostics.HasError())
		}
	}
}
