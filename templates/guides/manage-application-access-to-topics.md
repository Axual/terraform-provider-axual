---
page_title: "Managing application access to topics"
---

This guide explains how an application gets access to a topic through Terraform: the application team requests it, the topic team decides. The workflow depends on the environment's `authorization_issuer` setting.

## Environment Authorization Types

| `authorization_issuer` | Grant Behavior |
|------------------------|----------------|
| `Auto` | Requests are approved automatically by the system |
| `Stream owner` | Requests wait for the topic team's decision |

---

## Key Resources

| Resource | Who | Purpose |
|----------|-----|---------|
| `axual_application_access_grant` | Application team | Requests access. Deleting it cancels a Pending request or revokes an Approved grant. Shows the topic team's answer: `status`, `decision_reason`, `decided_at`, `decided_by`. |
| `axual_application_access_grant_decision` | Topic team | Records the decision: `decision = "APPROVED"` or `"DENIED"`. The provider approves, rejects or revokes to match. |
| `data.axual_application_access_grants` | Topic team | Lists grants, for example the open requests on a topic. |

`axual_application_access_grant_approval` and `axual_application_access_grant_rejection` are deprecated. See [Moving from the approval and rejection resources](#moving-from-the-approval-and-rejection-resources).

---

## Grant Lifecycle

One grant exists per application, topic, environment and access type, and it keeps its UID.

```
                     ┌──────────┐  decision = "DENIED"   ┌─────────┐
            ┌───────►│ Approved │───────────────────────►│ Revoked │
            │        └──────────┘  (or delete the grant) └─────────┘
┌─────────┐ │                                                 │
│ Pending │─┤        ┌──────────┐                             │
└─────────┘ ├───────►│ Rejected │  decision = "DENIED"        │
     ▲      │        └──────────┘                             │
     │      │        ┌───────────┐                            │
     │      └───────►│ Cancelled │  delete the grant          │
     │               └───────────┘                            │
     └──── request again (request_again, or the UI) ──────────┘
```

| From | To | Action |
|------|----|--------|
| none | Pending (Approved in Auto) | Application team creates the grant |
| Pending | Approved | Topic team: `decision = "APPROVED"` |
| Pending | Rejected | Topic team: `decision = "DENIED"` with a `reason` |
| Pending | Cancelled | Application team deletes the grant |
| Approved | Revoked | Topic team: `decision = "DENIED"`, or the application team deletes the grant |
| Rejected, Revoked, Cancelled | Pending | Application team requests again: the **same** grant reopens |

Rejected, Revoked and Cancelled are closed, not final.

---

## Application Team

### Requesting Access

```hcl
resource "axual_application_access_grant" "log_scraper_reads_logs" {
  application = axual_application.log_scraper.id
  topic       = data.axual_topic.logs.id
  environment = data.axual_environment.dev.id
  access_type = "CONSUMER"
  depends_on  = [axual_application_principal.log_scraper_dev]
}

output "logs_access" {
  value = {
    status = axual_application_access_grant.log_scraper_reads_logs.status
    reason = axual_application_access_grant.log_scraper_reads_logs.decision_reason
  }
}
```

Then open a PR in the topic team's repository that adds the decision (see below).

### Seeing the Answer

`status`, `decision_reason`, `decided_at` and `decided_by` show the topic team's decision. They are null while the grant is Pending. When the grant is Rejected, Revoked or Cancelled, every plan warns:

```
Warning: Grant Log Scraper (CONSUMER on logs, dev) is Rejected by tom@example.com

Reason from the topic team: "Not allowed to read logs, see INC-1234".
Closed by another user than the one who requested it. If the topic team set decision = "DENIED", ask them first: a new request is rejected again. Otherwise change request_again in a PR, or request again in the UI.
```

The plan shows the new `status` under "Changes to Outputs" of `logs_access`. Optionally, a `check` warns until the grant is approved:

```hcl
check "logs_access_approved" {
  assert {
    condition     = axual_application_access_grant.log_scraper_reads_logs.status == "Approved"
    error_message = "Access to logs is ${axual_application_access_grant.log_scraper_reads_logs.status}."
  }
}
```

### Requesting Again

Change `request_again` (free text) in a PR. When the grant is Rejected, Revoked or Cancelled, the apply sends one request and the same grant reopens. On a Pending or Approved grant a change does nothing; it never revokes.

```hcl
resource "axual_application_access_grant" "log_scraper_reads_logs" {
  # ...
  request_again = "INC-1300: consumer group fixed"
}
```

If the topic team rejects again, nothing more happens until `request_again` changes again. A change while the grant is Pending or Approved is stored and does nothing, also not later: after a reject, change the value again.

### Cancelling or Giving Up Access

Delete the `axual_application_access_grant`: a Pending grant is cancelled, an Approved grant is revoked. To accept a rejection, remove the grant block too: there is no API call, the warnings stop, and the grant stays `Rejected` on the platform.

### Changing Grant Attributes

`application`, `topic`, `environment` and `access_type` can't be changed in place. To switch from CONSUMER to PRODUCER, add a new grant block for PRODUCER, and remove the CONSUMER block when it's no longer needed (that revokes it).

---

## Topic Team

### Deciding

One block per application, topic, environment and access type:

```hcl
resource "axual_application_access_grant_decision" "log_scraper_reads_logs" {
  application = "Log Scraper" # name, or UID
  topic       = axual_topic.logs.id
  environment = data.axual_environment.dev.id
  access_type = "CONSUMER"
  decision    = "APPROVED" # or "DENIED" with a reason
  # On destroy, the decision revokes its grant before the topic config is deleted.
  depends_on = [axual_topic_config.logs_dev]
}
```

- `APPROVED` is a standing decision: it approves every request from that application, also a request made again. `DENIED` with a `reason` rejects or revokes, also every later request; the application team reads the reason.
- Once the topic exists, the request and the decision can be created in either order: before the request exists the plan only warns, and the topic repository's next apply after the request approves it.
- A reject or revoke in the UI lasts only until the next request while the decision is `APPROVED`. To keep it closed, set `DENIED`.
- Removing a decision revokes an `Approved` grant.

The rules, the table of what the provider does, many applications on one topic (`for_each`), and an optional `check` for waiting requests are in the [decision resource](../resources/application_access_grant_decision.md).

### Finding Open Requests

```hcl
data "axual_application_access_grants" "open" {
  topic       = axual_topic.logs.id
  environment = data.axual_environment.dev.id
  statuses    = ["Pending"]
}
```

`data.axual_application_access_grants` shows the grants as they were when the plan started, before the decisions of the same run.

### Cutting Access Now

Revoke the grant in the UI, then set `decision = "DENIED"` with the reason in a PR to the topic repository. The UI revoke alone lasts only until the next request. See [Revoking](../resources/application_access_grant_decision.md#revoking).

### Removing a Decision

Removing the block revokes an Approved grant, as removing an approval does. To rename or move a block, add a `moved` block; to stop managing a decision without revoking, set `revoke_on_destroy = false` and apply first, or use `terraform state rm`. Details: [Removing a block](../resources/application_access_grant_decision.md#removing-a-block-or-pointing-it-at-another-grant).

### Many Decisions in One Apply

The provider sends at most 4 approvals, rejects or revokes at a time, to protect Platform Manager. No `-parallelism` is needed.

---

## Multi-Repository Setup

| Team | Repository | Resources |
|------|------------|-----------|
| Application Team | `app-team-repo` | `axual_application_access_grant` |
| Topic Team | `topic-team-repo` | `axual_application_access_grant_decision` |

The decision finds the grant itself, by application name or UID, topic, environment and access type. No data source is needed, and private applications work (Platform Manager 15.0.3 or later). `request_again` text isn't sent to the platform, so the reason for a request belongs in the PR to the topic repository.

| Action | Who | How |
|--------|-----|-----|
| Request access | Application team | Create the grant, then a PR into the topic repository |
| Approve | Topic team | Merge the PR with `decision = "APPROVED"` |
| Reject or revoke | Topic team | `decision = "DENIED"` with a `reason` |
| See the answer | Application team | `status`, `decision_reason`, plan warnings |
| Ask again after `DENIED` | Both, topic team first | Topic PR: `decision = "APPROVED"`; then the application team changes `request_again` |
| Ask again after a reject or revoke in the UI | Application team | Change `request_again`; under `APPROVED` the topic repository's next apply approves it |
| Revoke in the UI | Topic team | Works, but under `APPROVED` the next request is approved again (a warning on every plan). To keep it closed, set `DENIED` in a PR as well |
| Give up access | Application team | Delete the grant (cancels or revokes). Adding it back later is approved again under `APPROVED` |
| CONSUMER to PRODUCER | Both | A new grant block and a new decision. Removing the old decision revokes the old grant |

See the [Multi-Repo Guide](multi-repo) for the full setup.

---

## Auto Environments

Requests are approved straight away. A decision is optional. `decision = "DENIED"` revokes, but the next request is approved again by the platform. The plan warns about it when the decision is created or changes, and when the apply revokes such a request.

---

## Platform Manager Versions

Public and private applications from Platform Manager 15.0.3. On 15.0.1 and 15.0.2 the topic owner can't see the requests of private applications. IAM-managed groups have a known limit. See [Platform Manager versions](../resources/application_access_grant_decision.md#platform-manager-versions).

---

## Import

```shell
# The grant (application team)
terraform import axual_application_access_grant.my_grant <GRANT_UID>

# The decision (topic team): by grant UID, or <topic>/<environment>/<access_type>/<application>
terraform import axual_application_access_grant_decision.my_decision <GRANT_UID>
```

The grant UID is in the Axual Self-Service UI, in `grant_id` of a decision, or in `data.axual_application_access_grants`.

---

## Moving from the Approval and Rejection Resources

With Terraform 1.8 or later, move each `axual_application_access_grant_approval` or `_rejection` block to an `axual_application_access_grant_decision` with a `moved` block: the plan shows no API action, and removing the decision later revokes, as removing the approval did. Don't simply delete the approval and add the decision: that plan fails, because removing the approval revokes the grant. Don't manage one grant with an old resource and a decision at the same time. The example, Terraform 1.7, the old data sources and self-approval: see [Moving from the approval and rejection resources](../resources/application_access_grant_decision.md#moving-from-the-approval-and-rejection-resources).

---

## Sections that moved

Earlier versions of this guide had these sections; links to them land here.

### Approving a Grant

Use `decision = "APPROVED"` on `axual_application_access_grant_decision`; see [Deciding](#deciding).

### Revoking an Approved Grant

Set `decision = "DENIED"` with a `reason`, or remove the decision; see [Cutting Access Now](#cutting-access-now) and [Removing a Decision](#removing-a-decision).

### Re-requesting Access After Rejection or Revocation

Change `request_again`; see [Requesting Again](#requesting-again).

### Re-approving After Revocation

The same grant reopens when the application team asks again (`request_again`, or the UI). Under `decision = "APPROVED"` the topic repository's next apply approves it; after `DENIED`, the topic team changes the decision to `APPROVED` first. See [Deciding](#deciding).

### Rejecting a Grant (Topic Owner Only)

Use `decision = "DENIED"` with a `reason` on `axual_application_access_grant_decision`; see [Deciding](#deciding).

### Revoking in Multi-Repo Setup

The topic team sets `decision = "DENIED"` with a `reason`; the application team can still delete its grant. See [Multi-Repository Setup](#multi-repository-setup).

### Import Order

Import the grant and the decision in any order; see [Import](#import).
