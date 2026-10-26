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
Warning: Grant Log Scraper (CONSUMER on logs, dev) is Rejected

Reason from the topic team: "Not allowed to read logs, see INC-1234".
Change request_again in a PR to ask again, or request again in the UI.
```

### Requesting Again

Change `request_again` (free text) in a PR. When the grant is Rejected, Revoked or Cancelled, the apply sends one request and the same grant reopens. On a Pending or Approved grant a change does nothing; it never revokes.

```hcl
resource "axual_application_access_grant" "log_scraper_reads_logs" {
  # ...
  request_again = "INC-1300: consumer group fixed"
}
```

If the topic team rejects again, nothing more happens until `request_again` changes again.

### Cancelling or Giving Up Access

Delete the `axual_application_access_grant`: a Pending grant is cancelled, an Approved grant is revoked.

### Changing Grant Attributes

`application`, `topic`, `environment` and `access_type` can't be changed in place. To switch from CONSUMER to PRODUCER, add a new grant block for PRODUCER, and remove the CONSUMER block when it's no longer needed (that revokes it).

---

## Topic Team

### Deciding

One block for all decisions on a topic and environment:

```hcl
locals {
  logs_dev_consumers = {
    "Log Scraper" = { decision = "APPROVED", review = "PR-41" }
    "POC Secret"  = { decision = "DENIED", reason = "Not allowed to read logs, see INC-1234" }
  }
}

resource "axual_application_access_grant_decision" "logs_dev_consumers" {
  for_each    = local.logs_dev_consumers
  application = each.key                  # name or UID
  topic       = axual_topic.logs.id
  environment = data.axual_environment.dev.id
  access_type = "CONSUMER"
  decision    = each.value.decision
  reason      = try(each.value.reason, null)
  review      = try(each.value.review, null)
}
```

- **Approve:** merge a PR with `decision = "APPROVED"`. The two repositories can apply in either order: before the request exists, the plan only warns, and the first request is approved by the next apply (`next_request = "approve"`).
- **Reject or revoke:** `decision = "DENIED"` with a `reason`. The application team reads the reason.
- **A request made again** after a reject or revoke is not approved automatically. The plan warns; change `review` (for example to the PR number) in a PR to approve it.
- **Every action is an in-place change** of `current_status` in the plan, for example `"Pending" -> "Approved"`.

### Finding Open Requests

```hcl
data "axual_application_access_grants" "open" {
  topic       = axual_topic.logs.id
  environment = data.axual_environment.dev.id
  statuses    = ["Pending"]
}
```

### Removing a Decision

Removing the block makes no API call: an Approved grant stays Approved, and the plan warns. Set `decision = "DENIED"` first to revoke, or set `revoke_on_destroy = true` so that removing the block revokes.

---

## Multi-Repository Setup

| Team | Repository | Resources |
|------|------------|-----------|
| Application Team | `app-team-repo` | `axual_application_access_grant` |
| Topic Team | `topic-team-repo` | `axual_application_access_grant_decision` |

The decision finds the grant itself, by application name or UID, topic, environment and access type. No data source is needed, and private applications work (Platform Manager 15.0.4 or later).

| Action | Who | How |
|--------|-----|-----|
| Request access | Application team | Create the grant, then a PR into the topic repository |
| Approve | Topic team | Merge the PR with `decision = "APPROVED"` |
| Reject or revoke | Topic team | `decision = "DENIED"` with a `reason` |
| See the answer | Application team | `status`, `decision_reason`, plan warnings |
| Ask again | Application team | Change `request_again` in a PR |
| Approve a request made again | Topic team | Change `review` in a PR |
| Give up access | Application team | Delete the grant (cancels or revokes) |

See the [Multi-Repo Guide](multi-repo) for the full setup.

---

## Auto Environments

Requests are approved straight away. A decision is optional. `decision = "DENIED"` revokes, but the next request is approved again by the platform, and every plan warns about it.

---

## Platform Manager Versions

| Platform Manager | Support |
|---|---|
| 15.0.4 and later, 16.x | Public and private applications |
| 15.0.1 to 15.0.3 | Public applications. Requests of private applications aren't visible to the topic owner; the decision shows `current_status = "Not found"` with a warning. |

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

With Terraform 1.8 or later, move each block:

```hcl
moved {
  from = axual_application_access_grant_approval.approve_grant
  to   = axual_application_access_grant_decision.approve_grant
}

resource "axual_application_access_grant_decision" "approve_grant" {
  application = "Log Scraper"
  topic       = axual_topic.my_topic.id
  environment = data.axual_environment.dev.id
  access_type = "CONSUMER"
  decision    = "APPROVED"
}
```

The plan shows only the move. A moved rejection gets `decision = "DENIED"` with its reason. After the move:

- removing the block no longer revokes, unless `revoke_on_destroy = true`;
- a request made again after a reject or revoke waits for a `review` change.

With Terraform 1.7, use a `removed` block with `lifecycle { destroy = false }` for the old resource and an `import` block for the decision. Don't simply delete an approval: that revokes the grant.

Don't manage one grant with an old resource and a decision at the same time.
