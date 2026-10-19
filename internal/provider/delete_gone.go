package provider

import (
	webclient "axual-webclient"
	"errors"
	"net/http"
	"strings"
)

// isGone reports a 404, or the 400 "No application found" that Platform Manager answers when the
// application of a credential is already deleted.
func isGone(err error) bool {
	if errors.Is(err, webclient.NotFoundError) {
		return true
	}
	var httpErr *webclient.HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest &&
		strings.Contains(httpErr.Body, "No application found")
}

// ignoreGone returns nil when err only says the resource is already gone.
func ignoreGone(err error) error {
	if isGone(err) {
		return nil
	}
	return err
}
