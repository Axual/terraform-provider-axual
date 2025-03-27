---
page_title: "Data Source: axual_user"
---
The `axual_user` data source allows you to retrieve details about an Axual user in the Self-Service platform. Users can be referenced by their email address.

## Example Usage

```hcl
data "axual_user" "tom" {
 email = "tom@email.com"
}
```

## Argument Reference

- email - (Required) The email address of the user to retrieve.

## Attribute Reference

This data source exports the following attributes in addition to the one listed above:

- `id`
 - The user's unique identifier.
- `first_name`
 - The user's first name.
- `middle_name`
 - The user's middle name.
- `last_name`
 - The user's last name.
- `phone_number`
 - The user's phone number.