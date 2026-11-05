package provider

import (
	webclient "axual-webclient"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// decisionFields is the topic team's last decision on a grant, as the application team sees it.
type decisionFields struct {
	DecisionReason types.String `tfsdk:"decision_reason"`
	DecidedAt      types.String `tfsdk:"decided_at"`
	DecidedBy      types.String `tfsdk:"decided_by"`
}

func (decisionFields) null() decisionFields {
	return decisionFields{types.StringNull(), types.StringNull(), types.StringNull()}
}

func (decisionFields) unknown() decisionFields {
	return decisionFields{types.StringUnknown(), types.StringUnknown(), types.StringUnknown()}
}

// decisionFieldsOf returns the decision of an Approved, Rejected or Revoked grant. A Pending or
// Cancelled grant gets nulls: a reopened grant still carries the comment, time and user of the
// earlier decision, and they must not look like an answer to the new request.
func decisionFieldsOf(g webclient.GrantView) decisionFields {
	f := decisionFields{}.null()
	switch g.Status {
	case "Rejected", "Revoked":
		f.DecisionReason = stringOrNull(g.Comment)
		fallthrough
	case "Approved":
		f.DecidedAt = stringOrNull(g.ProcessedAt)
		f.DecidedBy = stringOrNull(g.ProcessedBy)
	}
	return f
}

// grantName names a grant in messages: "Log Scraper (CONSUMER on logs, dev)".
func grantName(g webclient.GrantView) string {
	return fmt.Sprintf("%s (%s on %s, %s)", firstNonEmptyString(g.ApplicationName, g.ApplicationUid), g.AccessType,
		firstNonEmptyString(g.TopicName, g.TopicUid), firstNonEmptyString(g.EnvironmentName, g.EnvironmentUid))
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func decisionReasonAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: "The reason the topic team gave when it rejected or revoked the grant. Null unless the status is `Rejected` or `Revoked`.",
		Computed:            true,
	}
}

func decidedAtAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: "When the grant was approved, rejected or revoked. Null while the grant is `Pending` or `Cancelled`, also when it was decided before and has been requested again.",
		Computed:            true,
	}
}

func decidedByAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: "Who approved, rejected or revoked the grant. Null while the grant is `Pending` or `Cancelled`.",
		Computed:            true,
	}
}
