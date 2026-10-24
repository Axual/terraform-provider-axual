---
subcategory: ""
page_title: "IAM-managed groups"
description: |-
  How to link an Axual group to a group in your identity system, for example Azure AD, with iam_reference, and how service accounts work with such groups.
---

# IAM-managed groups

Needs Platform Manager 15.0.7 or later. Service accounts need 16.0.0 or later.

A normal group (**Axual-managed**) has a member list that is stored in Axual. An
**IAM-managed** group has no member list that counts. Your identity system decides who is in
the group, each time a person signs in. The link is the `iam_reference` argument: the group
value that your identity system puts in the sign-in token. For Azure AD this is the object id
of the group.

## Before you start

- A Tenant Admin switches on IAM-managed groups for the tenant: Tenants, your tenant, card
  "Group Management", Configure. Terraform cannot do this.
- Creating, linking and unlinking a group needs the `TENANT_ADMIN` role. This is true for a
  person and for a service account.
- If a group was linked outside Terraform (for example in Self-Service), add its `iam_reference`
  to the configuration before you apply. Without it, the plan unlinks the group.

## Create an IAM-managed group

```hcl
resource "axual_group" "payments" {
  name          = "Payments Team"
  email_address = "payments@example.com"
  iam_reference = "3f2a8c1e-5b7d-4e90-9a61-0c4d2e8f7b15"
}
```

The reference must be exactly the value in the token, and it must be unique in the tenant.
The plan refuses a reference that is empty, has a space at the start or end, or has a comma.

## Link or unlink an existing group

Add `iam_reference` to a group to link it. Remove it to make the group Axual-managed again.
Both are updates in place: the group stays, and so do the topics and applications it owns.

| You change | What happens |
|---|---|
| Add `iam_reference` | The group is linked. The members stored in Axual stay, and Platform Manager still uses them in some places (service accounts, group managers, creating applications, topics and environments). To remove them, take them out of `members` and apply, then add `iam_reference`. |
| Change `iam_reference` | The group moves to another IAM group. |
| Remove `iam_reference` | The group is Axual-managed again. Its stored members and managers count again. The plan warns you, and the next plan shows them: add them to `members` and `managers` to keep them, or the next apply removes them. |

~> Check the stored members before you unlink a group. They may be old, because Axual did
not use them while the group was IAM-managed.

If Platform Manager refuses the link, nothing changes. If it refuses another change in the same
apply (for example a name that is taken), the group stays linked, and the next plan shows only
the rest.

## Import and read

Import a group by id or by name. `iam_reference` is filled in. This also works for groups that
Platform Manager created by itself from the group values in sign-in tokens.

```shell
terraform import axual_group.payments name:"Payments Team"
```

```hcl
data "axual_group" "payments" {
  name = "Payments Team"
}

# iam_reference is null (not set) for an Axual-managed group
output "payments_reference" {
  value = data.axual_group.payments.iam_reference
}
```

## Members and managers

For an IAM-managed group, Terraform does not send `members` or `managers` and does not read them
back. If you write them, the plan refuses them. People get in through your identity system. Use a
normal group if you need a list in Terraform.

## Service accounts

A service account has no sign-in token with groups, because your identity system does not know
it. So a service account is a member of a group only through the member list stored in Axual,
also for an IAM-managed group. This does not change when you link or unlink the group, and it
works in every tenant mode.

| | Works? |
|---|---|
| Terraform signs in as a service account with `TENANT_ADMIN`, and creates, links or unlinks groups | Yes |
| Terraform signs in as a service account that is a member of an IAM-managed group, and manages applications that the group owns | Yes |
| The same service account, but it is not a member of the group | No: "You do not belong to this group/team." |
| A service account without `TENANT_ADMIN` creates or links a group | No: 403 |
| `members = [service account uid]` on an IAM-managed group | The plan refuses it. It works on a normal group. |

To put a service account in an IAM-managed group, a Tenant Admin adds the group to the service
account with `groupIds` in the service account API. For how to sign in as a service account, see
[Service account authentication](service-account-authentication.md).

## Known Platform Manager problems

These Platform Manager problems (seen on 15.0.7, 16.0.1 and the 2026-10-23 master) can affect IAM-managed groups:

- A person who is a member only through the token cannot create an application, topic or environment for the
  group, and cannot save an application in Self-Service. A person who left the IAM group but is still stored can
  still create them.
- A person with more than about 65 groups in the token gets an error on every call.
- A group manager can change the `iam_reference` of a group, although only a Tenant Admin can link or unlink it.

## Errors

| Message | What to do |
|---|---|
| IAM Group Management is disabled for this tenant | Ask a Tenant Admin to switch on IAM-managed groups. |
| Axual Group Management is disabled for this tenant | The tenant allows IAM-managed groups only. Add an `iam_reference`, or switch on Axual-managed groups. |
| ... already exists in this tenant | Another group has this reference. Reference values are unique. |
| Access Denied (403) | The user or service account needs `TENANT_ADMIN`. |
