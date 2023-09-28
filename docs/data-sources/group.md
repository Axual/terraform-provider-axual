---
page_title: "Data Source: axual_group"
---
Use this data source to get an axual group in Self-Service, you can reference it by name.

## Example Usage


```hcl
data "axual_group" "frontend_developers" {
 name = "Frontend Developers"
}
```

## Argument Reference

- name - (Required) The group name.

## Attribute Reference

This data source exports the following attributes in addition to the one listed above:

- id group unique identifier.
- email_address The group email address.
- phone_number The group phone number.
- members The group members.
