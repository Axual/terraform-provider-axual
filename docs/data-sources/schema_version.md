---
page_title: "Data Source: axual_schema_version"
---
Use this data source to get an axual schema_version in Self-Service, you can reference it by the schema full_name (<NAMESPACE>.<NAME>) and the version.

## Example Usage


```hcl
data "axual_schema_version" "ApplicationV1" {
   full_name="io.axual.qa.general.Application"
   version = "1.0.0"
}
```

## Argument Reference

- full_name - (Required) Full name of the schema.
- version - (Required) The version of the schema

## Attribute Reference

This data source exports the following attributes in addition to the one listed above:

- id Schema_version unique identifier.
- body The avro schema.
- description A short description of the schema version
- schema_id The Schema unique identifier
