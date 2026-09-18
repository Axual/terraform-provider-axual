package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func uidSet(uids ...string) types.Set {
	values := make([]attr.Value, len(uids))
	for i, uid := range uids {
		values[i] = types.StringValue(uid)
	}
	return types.SetValueMust(types.StringType, values)
}

// Members and managers go to Platform Manager as bare uids, which is what lets it resolve a person
// from a service account without the provider asking first.
func TestCreateGroupRequestFromData_sendsBareUids(t *testing.T) {
	data := groupResourceData{
		Name:     types.StringValue("team"),
		Members:  uidSet("member-person", "member-sa"),
		Managers: uidSet("member-person", "member-sa"),
	}

	request, err := createGroupRequestFromData(context.Background(), &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, field := range []struct {
		name string
		got  []string
	}{
		{"members", request.Members},
		{"managers", request.Managers},
	} {
		want := []string{"member-person", "member-sa"}
		if len(field.got) != len(want) {
			t.Fatalf("%s = %q, want %q", field.name, field.got, want)
		}
		for i, uid := range want {
			if field.got[i] != uid {
				t.Errorf("%s[%d] = %q, want the bare uid %q", field.name, i, field.got[i], uid)
			}
		}
	}
}

func TestCreateGroupRequestFromData_emptyListsAreNotNull(t *testing.T) {
	data := groupResourceData{Name: types.StringValue("team")}

	request, err := createGroupRequestFromData(context.Background(), &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if request.Members == nil || len(request.Members) != 0 {
		t.Errorf("members = %#v, want an empty non-nil slice", request.Members)
	}
	if request.Managers == nil || len(request.Managers) != 0 {
		t.Errorf("managers = %#v, want an empty non-nil slice", request.Managers)
	}
}
