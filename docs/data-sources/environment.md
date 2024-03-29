---
page_title: "Data Source: axual_environment"
---
Use this data source to get an axual environment in Self-Service, you can reference it by name.

## Example Usage


```hcl
data "axual_environment" "frontend_developers" {
 name = "Frontend Developers"
}
```

## Argument Reference

- name - (Required) The environment name.

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