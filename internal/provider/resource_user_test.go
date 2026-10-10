package provider

import (
	"context"
	"encoding/json"
	"testing"

	webclient "axual-webclient"
)

// AXPD-9562: no roles must reach PATCH /users/{id}/roles as [], not null; the endpoint refuses
// null with "Required request body is missing".
func TestUserRequestSendsAnEmptyRolesList(t *testing.T) {
	body, err := json.Marshal(createUserRequestFromData(context.Background(), &userResourceData{}).Roles)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(body) != "[]" {
		t.Errorf("roles body = %s, expected []", body)
	}
}

// A user with no roles is stored as an empty set, not null, so `roles = []` stays consistent.
func TestMapUserKeepsAnEmptyRoleSet(t *testing.T) {
	data := userResourceData{}
	mapUserResponseToData(context.Background(), &data, &webclient.UserResponse{})
	if data.Roles == nil || len(data.Roles) != 0 {
		t.Errorf("roles = %v, expected an empty set", data.Roles)
	}
}
