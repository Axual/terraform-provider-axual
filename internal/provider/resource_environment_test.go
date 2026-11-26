package provider

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// deadlockLost is the 403 that Platform Manager answered when an environment delete lost a MySQL deadlock to an
// application delete of the same grants (Platform Manager master 8ca43848c, MySQL 8.4).
const deadlockLost = `{"detail":"Request processing failed: org.springframework.dao.CannotAcquireLockException: JDBC exception executing SQL [Deadlock found when trying to get lock; try restarting transaction] [SELECT grant_id FROM application_access_2_application_access_grant WHERE application_access_id IN (?) ORDER BY application_access_id, grant_id FOR UPDATE]","status":403}`

func environmentToDelete() *environmentResourceData {
	return &environmentResourceData{
		Id:         types.StringValue("e1"),
		Viewers:    types.SetNull(types.StringType),
		Properties: types.MapNull(types.StringType),
		Settings:   types.MapNull(types.StringType),
	}
}

// An environment delete that loses a lock to a parallel application or topic delete of the same grants is
// retried until it works (AXPD-12798, AXPD-12837). Other errors are not retried, and a gone environment is fine.
func TestEnvironmentDeleteRetriesALostLock(t *testing.T) {
	t.Cleanup(shorten(&environmentDeleteDelay))
	optimisticLockLost := `{"detail":"Request processing failed: org.springframework.orm.ObjectOptimisticLockingFailureException: Row was updated or deleted by another transaction"}`
	for name, tc := range map[string]struct {
		answers   []string // status:body per DELETE; the last one repeats
		wantError string   // "": the delete works
		wantCalls int
	}{
		"optimistic lock lost, then deleted": {[]string{"403:" + optimisticLockLost, "204:"}, "", 2},
		"deadlock lost twice, then deleted":  {[]string{"403:" + deadlockLost, "403:" + deadlockLost, "204:"}, "", 3},
		"conflict 409, then deleted":         {[]string{"409:{}", "204:"}, "", 2},
		"conflict every time":                {[]string{"403:" + deadlockLost}, "CannotAcquireLockException", 5},
		"another 403 is not retried":         {[]string{`403:{"detail":"Access denied"}`}, "Access denied", 1},
		"already gone":                       {[]string{"404:"}, "", 1},
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			client := fakePM(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || r.URL.Path != "/environments/e1" {
					t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusTeapot)
					return
				}
				answer := tc.answers[min(calls, len(tc.answers)-1)]
				calls++
				status, body, _ := strings.Cut(answer, ":")
				code := map[string]int{"204": 204, "403": 403, "404": 404, "409": 409}[status]
				w.WriteHeader(code)
				_, _ = w.Write([]byte(body))
			})
			resp := deleteResource(t, NewEnvironmentResource(AxualProvider{client: client}), environmentToDelete())
			if calls != tc.wantCalls {
				t.Errorf("DELETE calls = %d, expected %d", calls, tc.wantCalls)
			}
			if tc.wantError == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected the delete to work, got %v", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), tc.wantError) {
				t.Fatalf("expected an error with %q, got %v", tc.wantError, resp.Diagnostics)
			}
		})
	}
}
