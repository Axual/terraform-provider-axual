package provider

import (
	"context"
	"encoding/json"
	"testing"
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
