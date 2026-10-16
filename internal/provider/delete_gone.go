package provider

import (
	webclient "axual-webclient"
	"errors"
	"net/http"
	"strings"
)

// goneAnswers are the texts of the 400 that Platform Manager answers instead of a 404 for some
// resources that do not exist.
var goneAnswers = []string{
	"No such application access exists", // access grant
	"converted to null",                 // environment, schema version, deployment status
	"No Group found",                    // group delete
	"No application found",              // credential of a deleted application
}

// isGone reports a 404, or the 400 that Platform Manager answers for some deleted resources.
func isGone(err error) bool {
	if errors.Is(err, webclient.NotFoundError) {
		return true
	}
	var httpErr *webclient.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusBadRequest {
		return false
	}
	for _, answer := range goneAnswers {
		if strings.Contains(httpErr.Body, answer) {
			return true
		}
	}
	// Flink cluster: "Flink Cluster with ID <id> not found"
	return strings.Contains(httpErr.Body, "Flink Cluster with ID") && strings.Contains(httpErr.Body, "not found")
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

// isAlreadyStopped reports a STOP refused because the deployment is already STOPPED. The live
// status can still say Running for a moment, so it cannot tell this.
func isAlreadyStopped(err error) bool {
	var httpErr *webclient.HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest &&
		strings.Contains(httpErr.Body, "in state STOPPED")
}
