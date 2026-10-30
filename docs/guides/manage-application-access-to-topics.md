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
- The two repositories can apply in either order: before the request exists the plan only warns, and the topic repository's next apply after the request approves it.
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

1. Revoke the grant in the UI.
2. Open a PR in the topic repository with `decision = "DENIED"` and the reason.

The UI revoke alone lasts only until the next request.

### Removing a Decision

Removing the block revokes an Approved grant, as removing an approval did, and the plan warns. To stop managing a decision and leave the grant as it is, set `revoke_on_destroy = false` and apply before you remove it. For a whole resource block (not one `for_each` key: Terraform doesn't allow instance keys there), a `removed` block works too:

```hcl
removed {
  from = axual_application_access_grant_decision.approve_grant
  lifecycle {
    destroy = false
  }
}
```

When you rename a `for_each` key, add a `moved` block. Without it the plan fails: the old block would revoke the grant that the new one approves.

### Many Decisions in One Apply

Platform Manager can stop responding when it gets about 10 approvals, rejects or revokes at the same time. The provider sends at most 4 of them at a time, so an apply that acts on many grants, for example the first apply of a topic repository, takes longer but doesn't need `-parallelism`.

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

| Platform Manager | Support |
|---|---|
| 15.0.3 and later, 16.x | Public and private applications |
| 15.0.1 and 15.0.2 (before 15.0.3) | Public applications. The requests of private applications aren't visible to the topic owner and show as `Not found`; the warning says the application may be private. Approve those in the UI, or upgrade. |
| Any version with IAM-managed groups | A topic owner who is a member of the owner group only through the IAM doesn't see its requests (a known Platform Manager issue): they show as `Not found`, and the warning names this cause. Run the topic repository as a service account (service accounts keep their membership), set `grant_id`, or approve in the UI. |

A private environment always uses the Auto authorization issuer, so its requests are approved straight away and need no decision.

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

In a single repository, the decision can use `grant_id = axual_application_access_grant.my_grant.id` instead of the four lookup attributes.

Remove the `data "axual_application_access_grant"` and `data "axual_application"` blocks that only fed the approval: the decision finds the grant itself. Kept, they still fail the whole plan when the application team hasn't created its grant yet, or deleted it.

The plan shows the move, and no API action. When `application` is a UID (for example taken from the grant), the first plan also changes it from the name to the UID, in place. Removing the decision later revokes the grant, like removing the approval did (`revoke_on_destroy` defaults to `true`). A moved rejection gets `decision = "DENIED"` with its reason. Since the decision approves only when the platform offers approve to the signed-in user, it doesn't approve a request this user made itself when self-approval is disabled on the instance, while the approval resource did. In the `grant_id` form the apply then fails.

With Terraform 1.7, use a `removed` block with `lifecycle { destroy = false }` for the old resource and an `import` block for the decision. Don't simply delete an approval: that revokes the grant.

Don't manage one grant with an old resource and a decision at the same time.

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
