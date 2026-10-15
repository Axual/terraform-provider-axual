package provider

import (
	webclient "axual-webclient"
	"testing"
)

// TestApplicationDeleteRetriesAConflict pins which answers the application delete retries: any 409, whatever
// its text (AXPD-12049 shows it varies by platform version), and the 403 optimistic-lock answer.
func TestApplicationDeleteRetriesAConflict(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "no error", err: nil, expected: false},
		{name: "commit conflict", err: &webclient.HTTPError{StatusCode: 409, Body: `{"message":"Could not commit changes!"}`}, expected: true},
		{name: "conflict with other wording", err: &webclient.HTTPError{StatusCode: 409, Body: `{"message":"Optimistic locking failed"}`}, expected: true},
		{name: "optimistic lock as 403", err: &webclient.HTTPError{StatusCode: 403, Body: "ObjectOptimisticLockingFailureException"}, expected: true},
		{name: "not found", err: webclient.NotFoundError, expected: false},
		{name: "forbidden", err: &webclient.HTTPError{StatusCode: 403, Body: `{"message":"Could not commit changes!"}`}, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isConflict(test.err); got != test.expected {
				t.Errorf("isConflict() = %t, expected %t", got, test.expected)
			}
		})
	}
}
