package webclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteApplicationCredentialErrors(t *testing.T) {
	tests := map[string]struct {
		status int
		body   string
		check  func(err error) bool
	}{
		"404": {http.StatusNotFound, "", func(err error) bool { return errors.Is(err, NotFoundError) }},
		"400": {http.StatusBadRequest, `{"detail":"No application found"}`, func(err error) bool {
			var httpErr *HTTPError
			return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest &&
				httpErr.Body == `{"detail":"No application found"}` &&
				err.Error() == `status: 400, body: {"detail":"No application found"}`
		}},
	}
	for name, test := range tests {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(test.status)
			_, _ = w.Write([]byte(test.body))
		}))
		c := &Client{HTTPClient: server.Client(), ApiURL: server.URL}
		err := c.DeleteApplicationCredential(ApplicationCredentialDeleteRequest{})
		server.Close()
		if !test.check(err) {
			t.Errorf("%s: unexpected error %#v", name, err)
		}
	}
}
