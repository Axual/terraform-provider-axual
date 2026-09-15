package provider

import (
	"errors"
	"testing"
)

// TestIsDeleteConflict pins the matching rule of the delete retry: only the status is matched, never
// the message. AXPD-12049 shows the wording comes from the platform's Spring version, and the same
// cascade failure has also surfaced as a 403, so a stricter match would silently stop retrying.
func TestIsDeleteConflict(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "no error", err: nil, expected: false},
		{name: "commit conflict", err: errors.New(`status: 409, body: {"message":"Could not commit changes!"}`), expected: true},
		{name: "conflict with other wording", err: errors.New(`status: 409, body: {"message":"Optimistic locking failed"}`), expected: true},
		{name: "not found", err: errors.New(`status: 404, body: {"message":"Not found"}`), expected: false},
		{name: "forbidden", err: errors.New(`status: 403, body: {"message":"Could not commit changes!"}`), expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isDeleteConflict(test.err); got != test.expected {
				t.Errorf("isDeleteConflict() = %t, expected %t", got, test.expected)
			}
		})
	}
}
