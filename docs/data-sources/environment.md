---
page_title: "Data Source: axual_environment"
---
This data source allows you to retrieve an existing environment from Self-Service by referencing either its `name` or `short_name`.
While both options are available, it is recommended to use `short_name` for better uniqueness and consistency.
You must provide at least one of `name` or `short_name`. If both are specified, the environment data will be resolved and exported based on the `short_name`.

## Example Usage

```hcl
data "axual_environment" "frontend_developers" {
 short_name = "FE dev"
}
```

```hcl
data "axual_environment" "frontend_developers" {
 name = "Frontend Developers"
}
```

## Argument Reference

- name - (Optional) The environment name.
- short_name - (Optional) The environment shortName.

## Attribute Reference

This data source exports the following attributes in addition to the one listed above:

- id environment unique identifier.
- short_name A short name that will uniquely identify this environment.
- description A text describing the purpose of the environment.
- color The color used display the environment
- visibility Possible valuese are Public or Private. Private environments are only visible to the owning group (your team). They are not included in dashboard visualisations.
- authorization_issuer This indicates if any deployments on this environment should be AUTO approved or requires approval from Stream Owner. For private environments, only AUTO can be selected.
- owners The id of the team owning this environment.
- instance The id of the instance where this environment should be deployed.
- retention_time The time in milliseconds after which the messages can be deleted from all topics.
- partitions Defines the number of partitions configured for every topic of this tenant.
- properties Environment-wide properties for all topics and applications.