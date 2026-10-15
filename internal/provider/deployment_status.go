package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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
			if waitErr := sleepCtx(ctx, statusRetryDelay); waitErr != nil {
				return nil, waitErr
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

// isTransientError reports a 5xx, a timeout, or a connection that was refused or dropped. Anything
// else, such as a 4xx, an untrusted certificate, a wrong URL or a refused token, fails the same way
// when sent again.
func isTransientError(err error) bool {
	var httpErr *webclient.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= http.StatusInternalServerError
	}
	var netErr net.Error
	return (errors.As(err, &netErr) && netErr.Timeout()) ||
		errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}
