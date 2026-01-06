---
page_title: "Managing application access to topics"
---

This guide explains how to manage access grants through Terraform. The workflow depends on the environment's `authorization_issuer` setting.

## Environment Authorization Types

| `authorization_issuer` | Grant Behavior |
|------------------------|----------------|
| `Auto` | Grants are approved automatically by the system |
| `Topic owner` | Grants require explicit approval from Topic Owner |

**Important**: In Topic owner environments, the Topic Owner must create an `axual_application_access_grant_approval` resource to approve the grant. In Auto environments, this is optional—the grant is already approved by the system. Creating the approval resource in Auto environments adopts it into Terraform state, which is only needed if you want to revoke via Terraform later.

---

## Key Resources

| Resource | Purpose |
|----------|---------|
| `axual_application_access_grant` | Request access to a topic (created by Application Owner) |
| `axual_application_access_grant_approval` | Approve a grant and manage it in Terraform state. **Deleting this resource revokes the grant.** Created by Topic Owner; can be deleted by either Application Owner or Topic Owner to revoke. |
| `axual_application_access_grant_rejection` | Reject a pending grant (Topic owner environments only, because Auto grants are never pending) |

---

## Grant Lifecycle

### Auto Environment

```
┌─────────┐    ┌──────────┐    ┌─────────┐
│ Created │───►│ Approved │───►│ Revoked │
└─────────┘    └──────────┘    └─────────┘
                (automatic)     (delete approval)
```

1. Grant is created → automatically approved by the system → application can produce/consume
2. *(Optional)* Topic Owner creates approval resource → adopts the grant into Terraform state
3. *(If step 2 was done)* Application Owner or Topic Owner deletes approval resource → grant is revoked

### Topic Owner Environment

```
                         ┌───────────┐
                    ┌───►│ Approved  │───┐
                    │    └───────────┘   │ (delete approval)
                    │                    ▼
┌─────────┐    ┌─────────┐          ┌─────────┐
│ Created │───►│ Pending │          │ Revoked │
└─────────┘    └─────────┘          └─────────┘
                    │
                    ├───►┌──────────┐
                    │    │ Rejected │
                    │    └──────────┘
                    │
                    └───►┌───────────┐
                         │ Cancelled │
                         └───────────┘
```

| From | To | Action |
|------|----|--------|
| Created | Pending | Grant created (automatic) |
| Pending | Approved | Topic Owner creates approval |
| Pending | Rejected | Topic Owner creates rejection |
| Pending | Cancelled | Application Owner deletes grant |
| Approved | Revoked | Application Owner or Topic Owner deletes approval |

**Terminal states (Revoked, Rejected, Cancelled):** Delete the grant and recreate to request access again.

---

## Application Owner

### Requesting Access

Create an `axual_application_access_grant` resource:

```hcl
resource "axual_application_access_grant" "my_app_consume_logs" {
  application = axual_application.my_app.id
  topic       = data.axual_topic.logs.id
  environment = data.axual_environment.dev.id
  access_type = "CONSUMER"  # or "PRODUCER"
  depends_on  = [
    axual_application_principal.my_app_principal,
    axual_topic_config.logs_in_dev
  ]
}
```

- **Auto environment**: Grant is approved automatically by the system. Topic Owner can optionally create approval resource to manage it in Terraform (required for revoking via Terraform).
- **Topic owner environment**: Grant starts in **Pending** status. Topic Owner must create approval resource to approve.

### Pending Grant: Cancel vs Reject (Topic Owner Environment Only)

A pending grant can be resolved in two ways:

| Action | Who | Status Change | How |
|--------|-----|---------------|-----|
| **Cancel** | Application Owner | Pending → Cancelled | Delete `axual_application_access_grant` |
| **Reject** | Topic Owner | Pending → Rejected | Create `axual_application_access_grant_rejection` |

**Cancel** - Application Owner withdraws their access request:

```bash
terraform destroy -target=axual_application_access_grant.my_app_consume_logs
```

**Reject** - Topic Owner denies the access request (see [Rejecting a Grant](#rejecting-a-grant-topic-owner-only)).

**Note**: Once approved, the Application Owner cannot delete the grant directly—it must be revoked first.

### Changing Grant Attributes

Grant attributes (`access_type`, `topic`, `application`, `environment`) cannot be updated in place.

**Both environment types:**
1. Topic Owner deletes their `axual_application_access_grant_approval` (this revokes the grant)
2. Application Owner deletes the `axual_application_access_grant` resource
3. Application Owner recreates the `axual_application_access_grant` with the new attributes
4. Topic Owner recreates the `axual_application_access_grant_approval`

In Auto environments, step 4 adopts the auto-approved grant into Terraform state.

### Re-requesting Access After Rejection or Revocation

Delete the `axual_application_access_grant` resource and recreate it:

```bash
terraform destroy -target=axual_application_access_grant.my_app_consume_logs
terraform apply
```

---

## Topic Owner

### Approving a Grant

Reference the grant using a data source:

```hcl
data "axual_application_access_grant" "grant" {
  application = data.axual_application.requesting_app.id
  topic       = axual_topic.my_topic.id
  environment = data.axual_environment.dev.id
  access_type = "CONSUMER"
}
```

Create an approval resource:

```hcl
resource "axual_application_access_grant_approval" "approve_grant" {
  application_access_grant = data.axual_application_access_grant.grant.id
}
```

**Behavior by environment type:**
- **Topic owner**: This calls the approve API and changes status from Pending to Approved
- **Auto**: The grant is already approved; this adopts it into Terraform state for management

### Rejecting a Grant (Topic Owner Only)

```hcl
resource "axual_application_access_grant_rejection" "reject_grant" {
  application_access_grant = data.axual_application_access_grant.grant.id
  reason                   = "Access not authorized for this application"  # optional
}
```

**Why Topic owner only?** In Auto environments, grants are never in Pending status—they're approved immediately. You can only reject a Pending grant. To deny access in Auto environments, revoke the grant instead (delete the approval resource).

**Note**: Rejection is final. The Application Owner must delete and recreate the grant to request access again.

### Revoking an Approved Grant

**Both Application Owner and Topic Owner can revoke** an approved grant by deleting the `axual_application_access_grant_approval` resource:

```bash
terraform destroy -target=axual_application_access_grant_approval.approve_grant
```

This changes the grant status to **Revoked** in both Auto and Topic owner environments. Access is revoked immediately.

**Note:** Application Owners cannot revoke by deleting their `axual_application_access_grant` resource—only by deleting the approval resource. Attempting to delete the grant while status is Approved will fail with: "Please Revoke this grant before attempting to delete it."

After revocation, the Application Owner's `axual_application_access_grant` resource becomes orphaned but can be deleted later without API calls (just cleans up Terraform state).

---

## Multi-Repository Setup

In a multi-team GitOps setup:

| Team | Repository | Resources |
|------|------------|-----------|
| Application Team | `app-team-repo` | `axual_application_access_grant` |
| Topic Team | `topic-team-repo` | `axual_application_access_grant_approval` (or `_rejection` in Topic owner) |

The Topic Team references the Application Team's grant using a **data source**, not a direct resource reference.

### Revoking in Multi-Repo Setup

**Either team can revoke independently—no coordination required.**

Both Application Owner and Topic Owner can revoke by deleting `axual_application_access_grant_approval`:

1. Delete `axual_application_access_grant_approval` from the repo that contains it
2. Access is revoked immediately—**done**

The `axual_application_access_grant` resource in Application Owner's repo becomes orphaned but harmless. Application Owner can delete it later to clean up their Terraform state:

```bash
# In app-team-repo (no API call made, just cleans up state)
terraform destroy -target=axual_application_access_grant.my_grant
```

See the [Multi-Repo Guide](multi-repo) for detailed setup instructions.

---

## Common Scenarios

### Scenario 1: Auto Environment - Grant Flow

1. **Application Owner** creates `axual_application_access_grant` → auto-approved by system
2. Application can now produce/consume
3. *(Optional)* **Topic Owner** creates `axual_application_access_grant_approval` → adopts grant into Terraform state for management (required if Topic Owner wants to revoke later via Terraform)

### Scenario 2: Topic Owner Environment - Approval Flow

1. **Application Owner** creates `axual_application_access_grant` → Status: **Pending**
2. **Topic Owner** creates `axual_application_access_grant_approval` → Status: **Approved**
3. Application can now produce/consume

### Scenario 3: Topic Owner Environment - Rejection

1. **Application Owner** creates `axual_application_access_grant` → Status: **Pending**
2. **Topic Owner** creates `axual_application_access_grant_rejection` → Status: **Rejected**
3. To try again: Application Owner deletes and recreates the grant

### Scenario 4: Revoking Access (Both Environment Types)

1. Grant is **Approved**
2. **Either Application Owner or Topic Owner** deletes `axual_application_access_grant_approval` → Status: **Revoked**, access revoked immediately
3. Application can no longer produce/consume
4. *(Optional cleanup)* Application Owner deletes orphaned `axual_application_access_grant` (no API call, just state cleanup)
5. To restore: Application Owner recreates grant, Topic Owner creates approval again

### Scenario 5: Changing CONSUMER to PRODUCER

1. **Application Owner or Topic Owner** deletes `axual_application_access_grant_approval` → Status: **Revoked**
2. **Application Owner** deletes `axual_application_access_grant`
3. **Application Owner** creates new `axual_application_access_grant` with `access_type = "PRODUCER"`
4. **Topic Owner** creates new `axual_application_access_grant_approval`

---

## Limitations

- **Revocation only via approval resource (Terraform provider limitation).** Both Application Owner and Topic Owner can revoke by deleting `axual_application_access_grant_approval`. However, deleting `axual_application_access_grant` while status is Approved will fail with: "Please Revoke this grant before attempting to delete it."
- Grant attributes cannot be updated in place. The grant must be revoked, deleted, and recreated.
- After rejection or revocation, the Application Owner should delete their grant resource before requesting access again.

## Notes

- **Both owners can revoke**: Either Application Owner or Topic Owner can revoke by deleting the approval resource.
- **Orphaned grants are harmless**: After revocation, the grant resource in Application Owner's repo can remain without affecting access.
- **Cleanup is no-op**: Deleting a revoked grant makes no API calls—it only removes the resource from Terraform state.

---

## Re-approving After Revocation

**A revoked grant cannot be re-approved.** The grant must be deleted and recreated first.

### The Problem

If Topic Owner tries to re-approve a revoked grant:

```
1. Create grant + approval     → Status: Approved ✅
2. Delete approval             → Status: Revoked
3. Create approval again       → ERROR ❌
```

```
│ Error: Error: Failed to approve grant
│
│ Only Pending grants can be approved
│ Current status of the grant is: Revoked
```

### Why This Happens

The approval resource can only approve grants in **Pending** status. Once a grant is revoked, it stays in **Revoked** status—it doesn't go back to Pending.

### The Issue with Multi-Repo

**There's no way to tell by looking at the `.tf` file which grants can be approved.**

The Topic Owner's `.tf` file might have:

```hcl
resource "axual_application_access_grant_approval" "approve_grant" {
  application_access_grant = data.axual_application_access_grant.grant.id
}
```

But this will fail if the underlying grant is in Revoked status. The Topic Owner has no visibility into the grant's current state from their Terraform configuration.

### Correct Workflow to Restore Access After Revocation

**Coordination is required:**

1. **Application Owner** deletes `axual_application_access_grant` (removes from state)
2. **Application Owner** recreates `axual_application_access_grant` (new grant in Pending/Approved status)
3. **Topic Owner** creates `axual_application_access_grant_approval` (now succeeds)
