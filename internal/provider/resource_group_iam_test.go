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
	// noReference makes a GET answer with a group that is not linked.
	noReference bool
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
		if r.Method == http.MethodGet || key == "DELETE /groups/g1/externalize" {
			if !p.noReference && r.Method == http.MethodGet {
				answer["iamReference"] = "stored-ref"
			}
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
		Id:           types.StringValue("g1"),
		Name:         types.StringValue("team"),
		EmailAddress: types.StringNull(),
		PhoneNumber:  types.StringNull(),
		Members:      types.SetNull(types.StringType),
		Managers:     types.SetNull(types.StringType),
		IamReference: types.StringNull(),
	}
	if iamReference != "" {
		data.IamReference = types.StringValue(iamReference)
	}
	if len(members) > 0 {
		data.Members = uidSet(members...)
	}
	return data
}

// emptyReference is a group whose configuration says iam_reference = "": the group is Axual-managed.
func emptyReference(members ...string) groupResourceData {
	data := groupData("", members...)
	data.IamReference = types.StringValue("")
	return data
}

// modifyPlan runs ModifyPlan with the configuration, the plan that Terraform proposes, and the state.
func modifyPlan(t *testing.T, config, plan groupResourceData, state *groupResourceData) resource.ModifyPlanResponse {
	t.Helper()
	r := &groupResource{}
	planState := stateFor(t, r, plan)
	configState := stateFor(t, r, config)
	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: configState.Schema, Raw: configState.Raw},
		Plan:   tfsdk.Plan{Schema: planState.Schema, Raw: planState.Raw},
		State:  emptyState(t, r),
	}
	if state != nil {
		req.State = stateFor(t, r, *state)
	}
	resp := resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(context.Background(), req, &resp)
	return resp
}

func plannedReference(t *testing.T, resp resource.ModifyPlanResponse) types.String {
	t.Helper()
	var reference types.String
	resp.Plan.GetAttribute(context.Background(), path.Root("iam_reference"), &reference)
	return reference
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

func TestGroupCreate_anEmptyReferenceMakesAnAxualManagedGroup(t *testing.T) {
	pm := &groupPM{noReference: true}
	r := &groupResource{provider: AxualProvider{client: fakePM(t, pm.handler(t))}}
	plan := tfsdk.Plan{Schema: stateFor(t, r, groupData("")).Schema, Raw: stateFor(t, r, emptyReference("m1")).Raw}
	config := tfsdk.Config{Schema: plan.Schema, Raw: plan.Raw}
	resp := resource.CreateResponse{State: emptyState(t, r)}

	r.Create(context.Background(), resource.CreateRequest{Config: config, Plan: plan}, &resp)

	var saved groupResourceData
	resp.State.Get(context.Background(), &saved)
	body := pm.bodies["POST /groups"]
	if resp.Diagnostics.HasError() || hasKey(body, "iamReference") || !hasKey(body, "members") {
		t.Errorf("create = %v, body %v; expected the members sent and no reference", resp.Diagnostics, body)
	}
	if saved.IamReference.IsNull() || saved.IamReference.ValueString() != "" {
		t.Errorf("saved reference = %s, expected the empty string of the plan", saved.IamReference)
	}
}

func TestGroupCreate_withoutAReferenceSavesNull(t *testing.T) {
	pm := &groupPM{noReference: true}
	r := &groupResource{provider: AxualProvider{client: fakePM(t, pm.handler(t))}}
	plan := tfsdk.Plan{Schema: stateFor(t, r, groupData("")).Schema, Raw: stateFor(t, r, groupData("", "m1")).Raw}
	config := tfsdk.Config{Schema: plan.Schema, Raw: plan.Raw}
	resp := resource.CreateResponse{State: emptyState(t, r)}

	r.Create(context.Background(), resource.CreateRequest{Config: config, Plan: plan}, &resp)

	var saved groupResourceData
	resp.State.Get(context.Background(), &saved)
	if resp.Diagnostics.HasError() || !saved.IamReference.IsNull() {
		t.Errorf("create = %v, saved reference = %s; expected a null reference", resp.Diagnostics, saved.IamReference)
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

func TestGroupUpdate_anEmptyReferenceInternalizesLast(t *testing.T) {
	pm := &groupPM{}

	resp, saved := updateGroup(t, pm, groupData("engineering"), emptyReference("m1"))

	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}
	if got := strings.Join(pm.calls, ", "); got != "PATCH /groups/g1, DELETE /groups/g1/externalize" {
		t.Errorf("calls = %s, expected a PATCH and then internalize", got)
	}
	if hasKey(pm.bodies["PATCH /groups/g1"], "iamReference") {
		t.Errorf("PATCH body %v has iamReference, expected none so that the link is only removed by the internalize", pm.bodies["PATCH /groups/g1"])
	}
	if members, ok := pm.bodies["PATCH /groups/g1"]["members"].([]any); !ok || len(members) != 1 {
		t.Errorf("PATCH body %v, expected the members back under Terraform", pm.bodies["PATCH /groups/g1"])
	}
	if saved.IamReference.IsNull() || saved.IamReference.ValueString() != "" {
		t.Errorf("saved reference = %s, expected the empty string of the plan", saved.IamReference)
	}
}

func TestGroupUpdate_anEmptyReferenceOnAGroupThatIsNotLinkedOnlyPatches(t *testing.T) {
	pm := &groupPM{noReference: true}

	resp, saved := updateGroup(t, pm, groupData("", "m1"), emptyReference("m1"))

	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}
	if got := strings.Join(pm.calls, ", "); got != "PATCH /groups/g1" {
		t.Errorf("calls = %s, expected only a PATCH: there is nothing to unlink", got)
	}
	if saved.IamReference.IsNull() || saved.IamReference.ValueString() != "" {
		t.Errorf("saved reference = %s, expected the empty string of the plan", saved.IamReference)
	}
}

func TestGroupUpdate_aKeptLinkIsOnlyAPatchWithTheSameReference(t *testing.T) {
	pm := &groupPM{}

	resp, saved := updateGroup(t, pm, groupData("engineering"), groupData("engineering"))

	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed: %v", resp.Diagnostics)
	}
	if got := strings.Join(pm.calls, ", "); got != "PATCH /groups/g1" {
		t.Errorf("calls = %s, expected only a PATCH: the link is not touched", got)
	}
	if pm.bodies["PATCH /groups/g1"]["iamReference"] != "engineering" {
		t.Errorf("PATCH body %v, expected the same reference", pm.bodies["PATCH /groups/g1"])
	}
	if saved.IamReference.ValueString() != "engineering" {
		t.Errorf("saved reference = %q, expected engineering", saved.IamReference.ValueString())
	}
}

// The state of an IAM-managed group has no members, so the plan cannot show the lists that Platform
// Manager stores. Unlinking must not clear them unseen: the next plan shows them.
func TestGroupUpdate_unlinkingWithoutMembersKeepsTheStoredLists(t *testing.T) {
	pm := &groupPM{}

	resp, saved := updateGroup(t, pm, groupData("engineering"), emptyReference())

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

	resp, saved := updateGroup(t, pm, groupData("engineering"), emptyReference("m1"))

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

// A refused update must not leave the group unlinked: then the old stored members would count again.
func TestGroupUpdate_aRefusedUpdateWhileUnlinkingKeepsTheLink(t *testing.T) {
	pm := &groupPM{refuse: "PATCH /groups/g1"}

	resp, _ := updateGroup(t, pm, groupData("engineering"), emptyReference("m1"))

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when Platform Manager refuses the update")
	}
	if got := strings.Join(pm.calls, ", "); got != "PATCH /groups/g1" {
		t.Errorf("calls = %s, expected no internalize after the refused PATCH", got)
	}
}

// A group that is linked outside Terraform still has the old lists in the state. Read removes them, so the plan
// does not show them as removed.
func TestGroupRead_aLinkMadeOutsideRemovesTheListsFromState(t *testing.T) {
	pm := &groupPM{}
	r := &groupResource{provider: AxualProvider{client: fakePM(t, pm.handler(t))}}
	before := groupData("", "m1")
	before.Managers = uidSet("m1")
	state := stateFor(t, r, before)
	resp := resource.ReadResponse{State: state}

	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

	var saved groupResourceData
	resp.State.Get(context.Background(), &saved)
	if saved.IamReference.ValueString() != "stored-ref" || !saved.Members.IsNull() || !saved.Managers.IsNull() {
		t.Errorf("saved reference %q, members %s, managers %s; expected the link and no lists",
			saved.IamReference.ValueString(), saved.Members, saved.Managers)
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
}

func TestGroupRead_keepsAnEmptyReferenceForAGroupThatIsNotLinked(t *testing.T) {
	pm := &groupPM{noReference: true}
	r := &groupResource{provider: AxualProvider{client: fakePM(t, pm.handler(t))}}
	state := stateFor(t, r, emptyReference())
	resp := resource.ReadResponse{State: state}

	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

	var saved groupResourceData
	resp.State.Get(context.Background(), &saved)
	if saved.IamReference.IsNull() || saved.IamReference.ValueString() != "" {
		t.Errorf("saved reference = %s, expected the empty string to stay", saved.IamReference)
	}
}

func TestGroupRead_aLinkMadeOutsideReplacesTheEmptyReference(t *testing.T) {
	pm := &groupPM{}
	r := &groupResource{provider: AxualProvider{client: fakePM(t, pm.handler(t))}}
	state := stateFor(t, r, emptyReference())
	resp := resource.ReadResponse{State: state}

	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

	var saved groupResourceData
	resp.State.Get(context.Background(), &saved)
	if saved.IamReference.ValueString() != "stored-ref" {
		t.Errorf("saved reference = %s, expected the link that now exists, so that the plan unlinks it again", saved.IamReference)
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
	data := groupData("engineering", "m1")

	resp := modifyPlan(t, data, data, nil)

	if resp.Diagnostics.ErrorsCount() != 1 || resp.Diagnostics.Errors()[0].Summary() != "Members and managers not supported" {
		t.Errorf("diagnostics = %v, expected one error about members of an IAM-managed group", resp.Diagnostics)
	}
}

func TestGroupModifyPlan_namesTheKeptLinkWhenMembersAreSetAndTheReferenceIsLeftOut(t *testing.T) {
	state := groupData("engineering")
	config := groupData("", "m1")
	plan := groupData("", "m1")
	plan.IamReference = types.StringUnknown()

	resp := modifyPlan(t, config, plan, &state)

	if resp.Diagnostics.ErrorsCount() != 1 || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `linked to IAM group "engineering"`) ||
		!strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `iam_reference = ""`) {
		t.Errorf("diagnostics = %v, expected an error that names the kept link and says how to unlink", resp.Diagnostics)
	}
}

func TestGroupModifyPlan_keepsTheLinkWhenTheReferenceIsLeftOut(t *testing.T) {
	state := groupData("engineering")
	config := groupData("")
	plan := groupData("")
	plan.IamReference = types.StringUnknown()

	resp := modifyPlan(t, config, plan, &state)

	if len(resp.Diagnostics) != 0 {
		t.Errorf("diagnostics = %v, expected none", resp.Diagnostics)
	}
	if got := plannedReference(t, resp); got.ValueString() != "engineering" {
		t.Errorf("planned reference = %s, expected the link the group has", got)
	}
}

func TestGroupModifyPlan_planNullForANewGroupWithoutAReference(t *testing.T) {
	config := groupData("")
	plan := groupData("")
	plan.IamReference = types.StringUnknown()

	resp := modifyPlan(t, config, plan, nil)

	if got := plannedReference(t, resp); !got.IsNull() {
		t.Errorf("planned reference = %s, expected null and not unknown", got)
	}
}

func TestGroupModifyPlan_keepsNullWhenTheGroupIsNotLinked(t *testing.T) {
	state := groupData("", "m1")
	config := groupData("", "m1", "m2")
	plan := groupData("", "m1", "m2")
	plan.IamReference = types.StringUnknown()

	resp := modifyPlan(t, config, plan, &state)

	if len(resp.Diagnostics) != 0 {
		t.Errorf("diagnostics = %v, expected none", resp.Diagnostics)
	}
	if got := plannedReference(t, resp); !got.IsNull() {
		t.Errorf("planned reference = %s, expected null: an update of another field must not show iam_reference as unknown", got)
	}
}

func TestGroupModifyPlan_warnsWhenAnEmptyReferenceInternalizes(t *testing.T) {
	state := groupData("engineering")
	data := emptyReference()

	resp := modifyPlan(t, data, data, &state)

	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 || resp.Diagnostics.Warnings()[0].Summary() != "The group becomes Axual-managed again" {
		t.Errorf("diagnostics = %v, expected one warning about becoming Axual-managed", resp.Diagnostics)
	}
}

func TestGroupModifyPlan_isQuietWhenAnEmptyReferenceIsOnAGroupThatIsNotLinked(t *testing.T) {
	state := groupData("", "m1")
	data := emptyReference("m1")

	resp := modifyPlan(t, data, data, &state)

	if len(resp.Diagnostics) != 0 {
		t.Errorf("diagnostics = %v, expected none", resp.Diagnostics)
	}
}

func TestGroupModifyPlan_warnsWhenLinkingAGroupThatHasStoredMembers(t *testing.T) {
	state := groupData("", "m1")
	data := groupData("engineering")

	resp := modifyPlan(t, data, data, &state)

	if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 || resp.Diagnostics.Warnings()[0].Summary() != "Stored members and managers stay in Platform Manager" {
		t.Errorf("diagnostics = %v, expected one warning that the stored members stay", resp.Diagnostics)
	}
}

func TestGroupModifyPlan_isQuietWhenLinkingAGroupWithoutMembers(t *testing.T) {
	state := groupData("")
	data := groupData("engineering")

	resp := modifyPlan(t, data, data, &state)

	if len(resp.Diagnostics) != 0 {
		t.Errorf("diagnostics = %v, expected none", resp.Diagnostics)
	}
}

func TestGroupModifyPlan_isQuietForAnOrdinaryGroup(t *testing.T) {
	state := groupData("", "m1")
	data := groupData("", "m1", "m2")

	resp := modifyPlan(t, data, data, &state)

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
		"":                                     true,
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
		strings.Repeat("\u00e9", 255):          true,
		strings.Repeat("\u00e9", 256):          false,
		strings.Repeat("\U0001F600", 127):      true,
		strings.Repeat("\U0001F600", 128):      false,
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
