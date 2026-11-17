package provider

import (
	webclient "axual-webclient"
	"strings"
	"testing"
)

// The log messages of the deployment requests show the config keys and never the values, because the values can hold passwords.
func TestDeploymentRequestLogMessagesHideConfigValues(t *testing.T) {
	configs := map[string]string{"connection.password": "S3cr3t-Passw0rd", "topics": "orders-secret-topic"}
	messages := map[string]string{
		"create": createRequestLogMessage(webclient.ApplicationDeploymentCreateRequest{
			Application: "app1", Environment: "env1", Configs: configs, TargetId: "target1", TargetVersion: "1.0.0"}),
		"update": updateRequestLogMessage(webclient.ApplicationDeploymentUpdateRequest{Configs: configs, TargetId: "target1"}),
	}
	for name, message := range messages {
		for key, value := range configs {
			if strings.Contains(message, value) {
				t.Errorf("%s message shows the config value of %q: %s", name, key, message)
			}
			if !strings.Contains(message, key) {
				t.Errorf("%s message does not show the config key %q: %s", name, key, message)
			}
		}
	}
}

func TestDeploymentRequestLogMessagesWithoutConfigs(t *testing.T) {
	if got := createRequestLogMessage(webclient.ApplicationDeploymentCreateRequest{Application: "app1", Environment: "env1"}); !strings.Contains(got, "config keys []") {
		t.Errorf("create message = %q, expected an empty list of config keys", got)
	}
	if got := updateRequestLogMessage(webclient.ApplicationDeploymentUpdateRequest{ClearTarget: true}); !strings.Contains(got, "clear target true") {
		t.Errorf("update message = %q, expected clear target true", got)
	}
}
