---
page_title: "Data Source: axual_topic"
---
Use this data source to get an axual topic in Self-Service, you can reference it by name.

## Example Usage

```hcl
data "axual_topic" "logs" {
 name = "logs"
}
```

## Argument Reference

- name - (Required) The topic name.

## Attribute Reference

This data source exports the following attributes in addition to the one listed above:

- id Topic unique identifier.
- description The description of the topic.
- key_type The key type and reference to the schema (if applicable).
- value_type The value type and reference to the schema (if applicable).
- owners The team owing this topic.
- retention_policy Determines what to do with messages after a certain period.
- properties Advanced (Kafka) properties for a topic in a given environment.