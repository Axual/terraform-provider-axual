---
page_title: "Importing user and group"
---

- Most resources require a Group or User uid, so we need to import them beforehand.
- There are 2 ways to get Group and User UID:
  - Importing User and Group
  - Hardcoding User and Group Ids
- We can get **USER_UID** from going to the "User" page in Self-Service (Settings -> Users -> Our user) and getting the UID from the URL.
- We can get **GROUP_UID** from going to the "Group" page in Self-Service (Settings -> Groups -> Our group) and getting the UID from the URL.

### Importing user and group

- To import, please go to user page and replace these values:
```shell
resource "axual_user" "gitops_user" {
  first_name    = "<REPLACE_WITH _FIRST_NAME>"
  last_name     = "<REPLACE_WITH_LAST_NAME>"
  email_address = "<REPLACE_WITH_EMAIL>"
  roles         = [
    { name = "STREAM_ADMIN" },
    { name = "APPLICATION_ADMIN" },
    { name = "ENVIRONMENT_ADMIN" },
  ]
}
resource "axual_group" "gitops_group" {
 name          = "default"
 members       = [
   axual_user.gitops_user.id,
 ]
}
```

- Now we can import user and group:

```shell
terraform import axual_user.gitops_user <USER_UID>
terraform import axual_group.gitops_group <GROUP_UID>
```

### Hardcoded values
- To use hardcoded values, we replace group UID for owners value in other resources, for example:
```shell
resource "axual_topic" "gitops_test_topic2" {
 name = "gitops_test_topic2"
 key_type = "String"
 value_type = "String"
 owners = "221771776652211ea556db870b084631"
 retention_policy = "delete"
 properties = { }
}
```