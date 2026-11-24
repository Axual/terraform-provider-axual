package provider

import (
	"context"
	"testing"
)

// The data sources promise an empty list, not null, when an object has no managers or viewers.
func TestUidStringSetIsEmptyNotNull(t *testing.T) {
	for name, uids := range map[string][]string{"nil": nil, "empty": {}} {
		set := uidStringSet(context.Background(), uids)
		if set.IsNull() || set.IsUnknown() || len(set.Elements()) != 0 {
			t.Errorf("%s: got %v, expected a known empty set", name, set)
		}
	}
	if set := uidStringSet(context.Background(), []string{"a", "b"}); len(set.Elements()) != 2 {
		t.Errorf("got %v, expected 2 elements", set)
	}
}
