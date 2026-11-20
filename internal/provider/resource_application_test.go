package provider

import (
	webclient "axual-webclient"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestIsConflict pins which answers the deletes and STOP retry as a conflict: any 409 whatever its text,
// which varies by platform version, and the 403 optimistic-lock answers of Spring and of JPA. Only an
// HTTPError counts, also when wrapped; the text of the error does not.
func TestIsConflict(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "no error", err: nil, expected: false},
		{name: "commit conflict", err: &webclient.HTTPError{StatusCode: 409, Body: `{"message":"Could not commit changes!"}`}, expected: true},
		{name: "conflict with other wording", err: &webclient.HTTPError{StatusCode: 409, Body: `{"message":"Optimistic locking failed"}`}, expected: true},
		{name: "optimistic lock as 403", err: &webclient.HTTPError{StatusCode: 403, Body: "ObjectOptimisticLockingFailureException"}, expected: true},
		{name: "JPA optimistic lock as 403", err: &webclient.HTTPError{StatusCode: 403, Body: lockLostJPA}, expected: true},
		{name: "JPA optimistic lock as 400", err: &webclient.HTTPError{StatusCode: 400, Body: lockLostJPA}, expected: false},
		{name: "not found", err: webclient.NotFoundError, expected: false},
		{name: "forbidden", err: &webclient.HTTPError{StatusCode: 403, Body: `{"message":"Could not commit changes!"}`}, expected: false},
		{name: "wrapped conflict", err: fmt.Errorf("delete: %w", &webclient.HTTPError{StatusCode: 409}), expected: true},
		{name: "conflict as text only", err: errors.New("status: 409, body: x"), expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isConflict(test.err); got != test.expected {
				t.Errorf("isConflict() = %t, expected %t", got, test.expected)
			}
		})
	}
}

// TestShortNameHint pins when a failed application update says how to change short_name in one apply:
// only for the 400 that Platform Manager answers while the application has principals, credentials or
// deployments, also when the error is wrapped.
func TestShortNameHint(t *testing.T) {
	refused := `{"detail" : "Application ShortName cannot be edited as there are existing Credentials, Deployments, or Principals for this application.", "status" : 400}`
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "short name refused", err: &webclient.HTTPError{StatusCode: 400, Body: refused}, expected: true},
		{name: "wrapped", err: fmt.Errorf("update: %w", &webclient.HTTPError{StatusCode: 400, Body: refused}), expected: true},
		{name: "other bad request", err: &webclient.HTTPError{StatusCode: 400, Body: `{"detail" : "Invalid application type"}`}, expected: false},
		{name: "other status", err: &webclient.HTTPError{StatusCode: 403, Body: refused}, expected: false},
		{name: "text only", err: errors.New(refused), expected: false},
		{name: "no error", err: nil, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hint := shortNameHint(test.err)
			if got := strings.Contains(hint, "replace_triggered_by"); got != test.expected {
				t.Errorf("shortNameHint() = %q, expected a hint: %t", hint, test.expected)
			}
		})
	}
}
