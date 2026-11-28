---
page_title: "Importing user and group"
---

- Most resources need the uid of a group or a user, for example in `owners` or `members`.
- There are 3 ways to get a Group or User uid:
  - Data sources (recommended): read the user or group, without managing it
  - Importing the User or Group (by email address and name, or by UID), so Terraform manages it
  - Hardcoding User and Group Ids
- Terraform cannot create users. A user must exist in Self-Service first.
- We can get **USER_UID** from going to the "User" page in Self-Service (Settings -> Users -> Our user) and getting the UID from the URL.
- We can get **GROUP_UID** from going to the "Group" page in Self-Service (Settings -> Groups -> Our group) and getting the UID from the URL.

### Data sources (recommended)

- A data source reads the user or group. Terraform never changes or deletes it:
```hcl
data "axual_user" "gitops_user" {
  email = "<REPLACE_WITH_EMAIL>"
}

data "axual_group" "gitops_group" {
  name = "default"
}
```

- Use `data.axual_user.gitops_user.id` and `data.axual_group.gitops_group.id` where a uid is needed.

### Importing user and group

~> **Warning:** After an import, Terraform manages the user or group. The first `terraform apply` sets it to
exactly what the configuration says. Roles, members or managers that the configuration leaves out can be removed.
Destroying an imported `axual_user`, or removing its block, deletes the user in Platform Manager (and its Keycloak
login, when Platform Manager manages the logins in Keycloak). Destroying an imported `axual_group` deletes the group.

- Import only when Terraform must manage the user or group. Before the import, copy the real values into the
  configuration. Look them up on the user page and the group page in Self-Service:
```hcl
resource "axual_user" "gitops_user" {
  first_name    = "<REPLACE_WITH_FIRST_NAME>"
  last_name     = "<REPLACE_WITH_LAST_NAME>"
  email_address = "<REPLACE_WITH_EMAIL>"
  # Leave out roles to keep the roles the user has. If you set roles, list ALL roles the
  # user has now: the apply replaces all roles of the user with this list.
}

resource "axual_group" "gitops_group" {
  name     = "default"
  # List ALL current members and managers. The apply removes anyone who is not listed.
  # Leave out managers only when the group has no managers.
  members  = ["<REPLACE_WITH_MEMBER_UID_1>", "<REPLACE_WITH_MEMBER_UID_2>"]
  managers = ["<REPLACE_WITH_MEMBER_UID_1>"]
}
```

- Now we can import user and group, by the user's email address and the group's name, so there is
  no need to look up their UIDs:

```shell
terraform import axual_user.gitops_user 'email:<REPLACE_WITH_EMAIL>'
terraform import axual_group.gitops_group 'name:default'
```

- Importing by UID works too: `terraform import axual_user.gitops_user <USER_UID>`. Topics,
  environments, applications and topic configs can also be imported by name; see the
  Import section of each resource page.

- Run `terraform plan` before any `terraform apply`. The plan must show no change for the user and the group.
  If it shows a change, fix the configuration first.

### Hardcoded values
- To use hardcoded values, we replace group UID for owners value in other resources, for example:
```hcl
resource "axual_topic" "gitops_test_topic2" {
 name = "gitops_test_topic2"
 key_type = "String"
 value_type = "String"
 owners = "221771776652211ea556db870b084631"
 retention_policy = "delete"
 properties = { }
}
```