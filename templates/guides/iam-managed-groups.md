---
subcategory: ""
page_title: "IAM-managed groups"
description: |-
  How to link an Axual group to a group in your identity system, for example Azure AD, with iam_reference, and how service accounts work with such groups.
---

# IAM-managed groups

Needs Platform Manager 15.0.0 or later. Service accounts need 16.0.0 or later. If people get into the group through the token, use Platform Manager 16.1.0 or later: older versions have problems for them (see "Known Platform Manager problems").

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
- A group that was linked outside Terraform (for example in Self-Service) keeps its link when the
  configuration has no `iam_reference`, like `tags` and `roles` keep their values. Terraform shows the
  link in the state, and does not show a link that is changed outside Terraform as a change.

## Create an IAM-managed group

```hcl
resource "axual_group" "payments" {
  name          = "Payments Team"
  email_address = "payments@example.com"
  iam_reference = "3f2a8c1e-5b7d-4e90-9a61-0c4d2e8f7b15"
}
```

The reference must be exactly the value in the token, and it must be unique in the tenant.
The plan refuses a reference that has a space at the start or end, or has a comma.

## Link or unlink an existing group

Add `iam_reference` to a group to link it. Write `iam_reference = ""` to make the group Axual-managed
again. Leaving the line out keeps the link the group has. Both are updates in place: the group stays,
and so do the topics and applications it owns.

| You change | What happens |
|---|---|
| Add `iam_reference` | The group is linked. The stored members stay in Axual, and service accounts in the list stay members. From Platform Manager 16.0, a person in the list is a member only when the token has the group, so a stored manager can no longer change the group. Before 16.1.0 the stored members also decide who can create applications, topics and environments for the group. The plan shows the lists as removed from the Terraform state only, with a warning. |
| Change `iam_reference` | The group moves to another IAM group. |
| Leave `iam_reference` out | Nothing changes: the group keeps the link it has, also a link that was made outside Terraform. |
| Write `iam_reference = ""` | The group is Axual-managed again (Terraform prints the change as `"..." -> null`). Its stored members and managers count again. The plan warns you, and the next plan shows them: add them to `members` and `managers` to keep them, or the next apply removes them. |

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
back. If you write them, the plan refuses them. This also holds for a group that is linked outside
Terraform when the configuration has no `iam_reference`: the plan names the link, and you remove
`members` and `managers`, or write `iam_reference = ""`. People get in through your identity
system. Use a normal group if you need a list in Terraform.

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

These problems can affect IAM-managed groups. The first three are fixed in Platform Manager 16.1.0 (seen on 15.0.7 and 16.0.1):

- A person who is a member only through the token cannot create an application, topic or environment for the
  group, and cannot save an application in Self-Service (AXPD-12554). A person who left the IAM group but is still
  stored can still create them.
- Lists and searches, for example a data source that looks up a private application by name, do not find the
  private applications, environments and access requests of the group for a person who is a member only through
  the token. A person who left the group still finds them (AXPD-11986).
- A group manager can change the `iam_reference` of a group, although only a Tenant Admin can link or unlink it
  (AXPD-12601).

Tenant Admins and service accounts are not affected by the first two, so a Terraform pipeline that signs in as one of
them works.

- A person with more than about 65 groups in the token gets an error on every call. Not fixed yet.
- Azure AD "group overage": for a person in about 200 or more groups, Azure AD sends a link instead of the list
  of groups. Platform Manager does not handle this. It was not tested with a real Azure AD.

## Errors

| Message | What to do |
|---|---|
| IAM Group Management is disabled for this tenant | Ask a Tenant Admin to switch on IAM-managed groups. |
| Axual Group Management is disabled for this tenant | The tenant allows IAM-managed groups only. Add an `iam_reference`, or switch on Axual-managed groups. |
| ... already exists in this tenant | Another group has this reference. Reference values are unique. To swap the references of two groups, use three applies: first give group A a temporary value, then give group B the old value of A, then give A the old value of B. |
| Access Denied (403) | The user or service account needs `TENANT_ADMIN`. |
