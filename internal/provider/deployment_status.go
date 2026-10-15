package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/oauth2"
)

// statusAttempts and statusRetryDelay retry a status read that gets a 5xx, for example for about a
// minute after the Connect workers restart. Variables so the unit tests can shorten them.
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
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(statusRetryDelay):
			}
		}
		var status *webclient.ApplicationDeploymentStatusResponse
		status, err = client.GetApplicationDeploymentStatus(id)
		if err == nil || !isTransientError(err) {
			return status, err
		}
	}
	return nil, fmt.Errorf("after %d attempts: %w", statusAttempts, err)
}

// isTransientError reports a 5xx, or a request that got no answer. Anything else, such as a 4xx, a body
// that cannot be read or a refused token, fails the same way when sent again.
func isTransientError(err error) bool {
	var httpErr *webclient.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= http.StatusInternalServerError
	}
	var urlErr *url.Error
	var tokenErr *oauth2.RetrieveError
	return errors.As(err, &urlErr) && !errors.As(err, &tokenErr)
}
