package provider

import (
	webclient "axual-webclient"
	"errors"
	"net/http"
	"strings"
)

// isGone reports whether err only says the resource does not exist any more. Platform Manager
// answers some lookups of a deleted resource with 400 instead of 404. A destroy then counts the
// resource as deleted: another resource in the same destroy, or a user, removed it already.
func isGone(err error) bool {
	if errors.Is(err, webclient.NotFoundError) {
		return true
	}
	var httpErr *webclient.HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest &&
		(strings.Contains(httpErr.Body, "No such application access exists") ||
			strings.Contains(httpErr.Body, "converted to null"))
}

// ignoreGone returns nil when err only says the resource is already gone.
func ignoreGone(err error) error {
	if isGone(err) {
		return nil
	}
	return err
}

// isConflict reports a concurrent change of the same row: Platform Manager answers 409 "Could not
// commit changes", or 403 with an ObjectOptimisticLockingFailureException. A retry can succeed.
func isConflict(err error) bool {
	var httpErr *webclient.HTTPError
	if !errors.As(err, &httpErr) {
		return false
	}
	return httpErr.StatusCode == http.StatusConflict ||
		(httpErr.StatusCode == http.StatusForbidden && strings.Contains(httpErr.Body, "OptimisticLocking"))
}

// isGrantClosed reports whether an access grant was revoked, rejected or cancelled.
func isGrantClosed(status string) bool {
	return status == "Revoked" || status == "Rejected" || status == "Cancelled"
}

// deploymentGone reports whether the deployment does not exist any more.
func deploymentGone(client *webclient.Client, id string) bool {
	_, err := client.GetApplicationDeployment(id)
	return isGone(err)
}

// isAlreadyStopped reports a STOP that Platform Manager refused because the deployment is stored as
// STOPPED already, for example by its deployment state in the same destroy. The worker can still
// report the connector as running for a moment, so the status cannot answer this.
func isAlreadyStopped(err error) bool {
	var httpErr *webclient.HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest &&
		strings.Contains(httpErr.Body, "in state STOPPED")
}
