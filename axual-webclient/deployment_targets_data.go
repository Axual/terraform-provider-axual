package webclient

// DeploymentTargetIneligibilityReason explains why a deployment target is not eligible for a
// given application, e.g. a missing plugin or a group not authorized on the target cluster.
type DeploymentTargetIneligibilityReason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// DeploymentTarget is one entry of GET /applications/{applicationId}/deployment-targets. Real
// registered Kafka Connect clusters and the legacy Axual Connect pseudo-target are both returned
// with Type "Connect"; only the shape of Id differs (a registered cluster's id vs.
// "axualconnect-<instance-short-name>"). Metadata is a free-form map; the two keys this provider
// reads are "kafkaAuthMethod" ("MTLS" or "SASL_SCRAM") and "supportedConnectorPlugins".
type DeploymentTarget struct {
	Id                   string                                `json:"id"`
	Type                 string                                `json:"type"`
	Name                 string                                `json:"name"`
	Metadata             map[string]interface{}                `json:"metadata"`
	Eligible             bool                                  `json:"eligible"`
	IneligibilityReasons []DeploymentTargetIneligibilityReason `json:"ineligibilityReasons"`
}

type DeploymentTargetsResponse struct {
	DeploymentTargets []DeploymentTarget `json:"deploymentTargets"`
}

// KafkaAuthMethod reads the "kafkaAuthMethod" metadata entry ("MTLS" or "SASL_SCRAM"), or "" when
// absent or not a string.
func (t DeploymentTarget) KafkaAuthMethod() string {
	value, ok := t.Metadata["kafkaAuthMethod"].(string)
	if !ok {
		return ""
	}
	return value
}
