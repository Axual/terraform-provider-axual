---
page_title: "Data Source: axual_instance"
---
Use this data source to get an axual instance in Self-Service, you can reference it by name.

## Example Usage


```hcl
data "axual_instance" "testInstance" {
 name = "testInstance"
}
```

## Argument Reference

- name - (Required) The group name.

## Attribute Reference

This data source exports the following attributes in addition to the one listed above:

- id group unique identifier.
- short_name The instance short name.
- description The group description.