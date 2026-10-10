package webclient

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func findAgainst(t *testing.T, status int, body string, field string, value string) []string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	client := &Client{HTTPClient: server.Client(), ApiURL: server.URL}
	uids, err := client.FindUidsByField("things/search/findByName", url.Values{"name": {value}}, "things", field, value)
	if err != nil {
		t.Fatalf("find error = %v", err)
	}
	return uids
}

func TestFindUidsByFieldKeepsOnlyExactMatches(t *testing.T) {
	body := `{"_embedded":{"things":[{"uid":"a","name":"Team"},{"uid":"b","name":"team"},{"uid":"c","name":"Team A"}]}}`
	if got := findAgainst(t, http.StatusOK, body, "name", "Team"); !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("uids = %v, expected [a]", got)
	}
}

func TestFindUidsByFieldReadsASingleObjectAndANestedField(t *testing.T) {
	body := `{"uid":"u1","emailAddress":{"email":"ann@example.com"}}`
	if got := findAgainst(t, http.StatusOK, body, "emailAddress.email", "ann@example.com"); !reflect.DeepEqual(got, []string{"u1"}) {
		t.Errorf("uids = %v, expected [u1]", got)
	}
}

func TestFindUidsByFieldTreatsNotFoundAsNoResult(t *testing.T) {
	if got := findAgainst(t, http.StatusNotFound, ``, "name", "x"); len(got) != 0 {
		t.Errorf("uids = %v, expected none", got)
	}
}
