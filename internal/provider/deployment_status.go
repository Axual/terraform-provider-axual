package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// statusAttempts and statusRetryDelay retry a status read that failed for a passing reason. Platform
// Manager answers a Connect deployment's status with 500 for about a minute after the Connect workers
// restart. Variables so the unit tests can shorten them.
var (
	statusAttempts   = 12
	statusRetryDelay = 10 * time.Second
)

// getDeploymentStatus reads the live status of a deployment. A 5xx or a failed request is read
// again; any other error is returned at once.
func getDeploymentStatus(ctx context.Context, client *webclient.Client, id string) (*webclient.ApplicationDeploymentStatusResponse, error) {
	var err error
	for i := 0; i < statusAttempts; i++ {
		if i > 0 {
			tflog.Warn(ctx, fmt.Sprintf("Reading the status of Application Deployment %s failed (attempt %d of %d): %s", id, i, statusAttempts, err))
			time.Sleep(statusRetryDelay)
		}
		var status *webclient.ApplicationDeploymentStatusResponse
		status, err = client.GetApplicationDeploymentStatus(id)
		if err == nil || !isTransientError(err) {
			return status, err
		}
	}
	return nil, fmt.Errorf("after %d attempts: %w", statusAttempts, err)
}

// isTransientError reports whether the same request can succeed when sent again: a 5xx or a failed
// request can, a 4xx cannot.
func isTransientError(err error) bool {
	if errors.Is(err, webclient.NotFoundError) {
		return false
	}
	var httpErr *webclient.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= http.StatusInternalServerError
	}
	return true
}
