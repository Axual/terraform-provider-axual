---
page_title: "Managing application access to topics"
---

This guide explains how to manage access grants through Terraform. The workflow depends on the environment's `authorization_issuer` setting.

## Environment Authorization Types

| `authorization_issuer` | Grant Behavior |
|------------------------|----------------|
| `Auto` | Grants are approved automatically by the system |
| `Stream owner` | Grants require explicit approval from Topic Owner |

**Important**: In Auto environments, access is granted immediately without any approval resource. However, if the Topic Owner wants to manage the grant in Terraform (e.g., to revoke it later), they should create an `axual_application_access_grant_approval` resource.

---

## Key Resources

| Resource | Purpose |
|----------|---------|
| `axual_application_access_grant` | Request access to a topic (created by Application Owner) |
| `axual_application_access_grant_approval` | Approve a grant and manage it in Terraform state. **Deleting this resource revokes the grant.** (created by Topic Owner) |
| `axual_application_access_grant_rejection` | Reject a pending grant (Stream owner environments only, because Auto grants are never pending) |

---

## Grant Lifecycle

### Auto Environment

```
┌─────────┐    ┌──────────┐    ┌─────────┐
│ Created │───►│ Approved │───►│ Revoked │
└─────────┘    └──────────┘    └─────────┘
                (automatic)     (delete approval)
```

1. Grant is created → automatically approved by the system (access is granted)
2. *(Optional)* Topic Owner creates approval resource → adopts the grant into Terraform state
3. Topic Owner deletes approval resource → grant is revoked

### Stream Owner Environment

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
| Approved | Revoked | Topic Owner deletes approval |

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

- **Auto environment**: Grant is approved automatically. Access is fully granted—no further action required from Application Owner.
- **Stream owner environment**: Grant starts in **Pending** status. Wait for Topic Owner to approve.

### Cancelling a Pending Grant (Stream Owner Only)

Delete the `axual_application_access_grant` resource while it's still **Pending**:

```bash
terraform destroy -target=axual_application_access_grant.my_app_consume_logs
```

**Note**: Once approved, the Application Owner cannot delete the grant directly.

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
- **Stream owner**: This calls the approve API and changes status from Pending to Approved
- **Auto**: The grant is already approved; this adopts it into Terraform state for management

### Rejecting a Grant (Stream Owner Only)

```hcl
resource "axual_application_access_grant_rejection" "reject_grant" {
  application_access_grant = data.axual_application_access_grant.grant.id
  reason                   = "Access not authorized for this application"  # optional
}
```

**Why Stream owner only?** In Auto environments, grants are never in Pending status—they're approved immediately. You can only reject a Pending grant. To deny access in Auto environments, revoke the grant instead (delete the approval resource).

**Note**: Rejection is final. The Application Owner must delete and recreate the grant to request access again.

### Revoking an Approved Grant

Delete the `axual_application_access_grant_approval` resource:

```bash
terraform destroy -target=axual_application_access_grant_approval.approve_grant
```

This changes the grant status to **Revoked** in both Auto and Stream owner environments.

---

## Multi-Repository Setup

In a multi-team GitOps setup:

| Team | Repository | Resources |
|------|------------|-----------|
| Application Team | `app-team-repo` | `axual_application_access_grant` |
| Topic Team | `topic-team-repo` | `axual_application_access_grant_approval` (or `_rejection` in Stream owner) |

The Topic Team references the Application Team's grant using a **data source**, not a direct resource reference.

See the [Multi-Repo Guide](multi-repo) for detailed setup instructions.

---

## Common Scenarios

### Scenario 1: Auto Environment - Grant Flow

1. **Application Owner** creates `axual_application_access_grant` → auto-approved, access granted immediately
2. Application can now produce/consume
3. *(Optional)* **Topic Owner** creates `axual_application_access_grant_approval` → adopts grant into Terraform for management

### Scenario 2: Stream Owner Environment - Approval Flow

1. **Application Owner** creates `axual_application_access_grant` → Status: **Pending**
2. **Topic Owner** creates `axual_application_access_grant_approval` → Status: **Approved**
3. Application can now produce/consume

### Scenario 3: Stream Owner Environment - Rejection

1. **Application Owner** creates `axual_application_access_grant` → Status: **Pending**
2. **Topic Owner** creates `axual_application_access_grant_rejection` → Status: **Rejected**
3. To try again: Application Owner deletes and recreates the grant

### Scenario 4: Revoking Access (Both Environment Types)

1. Grant is **Approved**
2. **Topic Owner** deletes `axual_application_access_grant_approval` → Status: **Revoked**
3. Application can no longer produce/consume
4. To restore: Application Owner deletes and recreates grant, Topic Owner creates approval again

### Scenario 5: Changing CONSUMER to PRODUCER

1. **Topic Owner** deletes `axual_application_access_grant_approval` → Status: **Revoked**
2. **Application Owner** deletes `axual_application_access_grant`
3. **Application Owner** creates new `axual_application_access_grant` with `access_type = "PRODUCER"`
4. **Topic Owner** creates new `axual_application_access_grant_approval`

---

## Limitations

- Application Owners cannot revoke their own approved grants. The Topic Owner must delete the approval resource.
- Grant attributes cannot be updated in place. The grant must be revoked, deleted, and recreated.
- After rejection or revocation, the grant must be deleted before requesting access again.
- Rejection is only available in Stream owner environments.
