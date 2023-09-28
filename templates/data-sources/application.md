---
page_title: "Data Source: axual_application"
---
Use this data source to get an axual application in Self-Service, you can reference it by name.

## Example Usage


```hcl
data "axual_application" "logs_producer" {
  name = "logs_producer"
}
```

## Argument Reference

- name - (Required) The application name.

## Attribute Reference

This data source exports the following attributes in addition to the one listed above:

- id Application unique identifier.
- description The description of the application.
- application_type Axual Application type. Possible values are Custom.
- application_id The Application Id of the Application, usually a fully qualified class name. Must be unique. The application ID, used in logging and to determine the consumer group (if applicable).
- short_name Unique human-readable name for the application.
- owners The team owing this application.
- type Application software. Possible values: Java, Pega, SAP, DotNet, Bridge
- visibility Defines the visibility of this application. Possible values are Public and Private.