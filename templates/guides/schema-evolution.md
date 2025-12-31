---
page_title: "Schema Evolution Guide"
---

This guide explains how schemas and schema versions work in Axual Platform Manager (PM) and how they relate to the Terraform provider resources.

## Concepts

### Schema in Platform Manager

A **Schema** in Axual Platform Manager is a parent entity that groups related schema versions together. It is identified by the combination of `namespace` and `name` in the schema body (for AVRO schemas).

For example, this AVRO schema body defines a Schema with the full name `io.axual.qa.keys.GitOpsTestKey`:

```json
{
  "type": "record",
  "name": "GitOpsTestKey",
  "namespace": "io.axual.qa.keys",
  "fields": [
    { "name": "keyId", "type": "string" }
  ]
}
```

### Schema Version in Platform Manager

A **Schema Version** is a specific version of a Schema. Multiple schema versions can belong to the same parent Schema as long as they share the same `namespace` and `name`. Each schema version has:
- A `version` string (e.g., "1.0.0", "2.0.0")
- A `body` containing the actual schema definition
- An optional `description`

### How axual_schema_version Relates to These Concepts

The Terraform resource `axual_schema_version` creates a **Schema Version** in Platform Manager. When you create an `axual_schema_version`:

1. The provider parses the schema body to extract `namespace` and `name`
2. If a Schema with that `namespace + name` already exists, the new version is added to it
3. If no such Schema exists, a new Schema is created automatically

This means multiple `axual_schema_version` resources with the same `namespace + name` in their body will belong to the same parent Schema.

```
axual_schema_version (v1.0.0)  ─┐
                                ├──► Parent Schema (io.axual.qa.keys.GitOpsTestKey)
axual_schema_version (v2.0.0)  ─┘
```

## Topic and Topic Config Relationship

- `axual_topic.key_schema` and `axual_topic.value_schema` reference the **parent Schema** (via `schema_id` attribute)
- `axual_topic_config.key_schema_version` and `axual_topic_config.value_schema_version` reference specific **Schema Versions** (via `id` attribute)

```hcl
resource "axual_topic" "logs" {
  key_schema   = axual_schema_version.key_v1.schema_id    # References parent Schema
  value_schema = axual_schema_version.value_v1.schema_id  # References parent Schema
}

resource "axual_topic_config" "logs_in_dev" {
  topic                = axual_topic.logs.id
  key_schema_version   = axual_schema_version.key_v1.id   # References specific version
  value_schema_version = axual_schema_version.value_v1.id # References specific version
}
```

### Understanding schema_id Equivalence

An important property of `schema_id`: **all schema versions belonging to the same parent Schema have the same `schema_id`**.

```hcl
# Both schemas have the same namespace+name (io.axual.qa.keys.GitOpsTestKey)
resource "axual_schema_version" "key_v1" {
  body    = file("schemas/key_v1.avsc")  # namespace: io.axual.qa.keys, name: GitOpsTestKey
  version = "1.0.0"
}

resource "axual_schema_version" "key_v2" {
  body    = file("schemas/key_v2.avsc")  # namespace: io.axual.qa.keys, name: GitOpsTestKey
  version = "2.0.0"
}

# These two expressions are EQUIVALENT:
# axual_schema_version.key_v1.schema_id == axual_schema_version.key_v2.schema_id
```

This means you can reference `schema_id` from **any version** of the same schema in your topic definition:

```hcl
resource "axual_topic" "logs" {
  # All of these are equivalent - they all reference the same parent Schema:
  key_schema = axual_schema_version.key_v1.schema_id
  # key_schema = axual_schema_version.key_v2.schema_id  # Same value!
}
```

**How this simplifies workflows:**

1. **Flexible referencing**: You can reference `schema_id` from whichever schema version is most convenient in your Terraform configuration. Changing the reference from `key_v1.schema_id` to `key_v2.schema_id` results in **no changes** for Terraform.

2. **Clean module design**: When passing schema references between Terraform modules, you only need to pass one schema version's `schema_id` - it works regardless of which version you choose.

3. **Refactoring safety**: If you rename schema version resources or reorganize your Terraform code, as long as the underlying schemas have the same `namespace + name`, the `schema_id` values remain identical.

**Key distinction:**
- `.schema_id` → Parent Schema UID (same for all versions with same namespace+name)
- `.id` → Schema Version UID (unique for each version)

## Evolving Schema Versions

### Backward Compatible Evolution (Single Apply)

You can evolve schema versions in `axual_topic_config` without changing the topic's schema, as long as the new schema version belongs to the same parent Schema.

**Example: Adding an optional field**

Original schema (v1.0.0):
```json
{
  "type": "record",
  "name": "GitOpsTestKey",
  "namespace": "io.axual.qa.keys",
  "fields": [
    { "name": "keyId", "type": "string" }
  ]
}
```

Evolved schema (v2.0.0) - same `namespace + name`, new optional field:
```json
{
  "type": "record",
  "name": "GitOpsTestKey",
  "namespace": "io.axual.qa.keys",
  "fields": [
    { "name": "keyId", "type": "string" },
    { "name": "source", "type": ["null", "string"], "default": null }
  ]
}
```

Terraform configuration:
```hcl
# Original version
resource "axual_schema_version" "key_v1" {
  body    = file("schemas/key_v1.avsc")
  version = "1.0.0"
}

# Evolved version - same parent Schema
resource "axual_schema_version" "key_v2" {
  body    = file("schemas/key_v2.avsc")
  version = "2.0.0"
}

resource "axual_topic" "logs" {
  key_schema = axual_schema_version.key_v1.schema_id  # Stays the same
}

resource "axual_topic_config" "logs_in_dev" {
  topic              = axual_topic.logs.id
  key_schema_version = axual_schema_version.key_v2.id  # Updated to v2
}
```

This change requires only a single `terraform apply` because:
- The topic's `key_schema` remains unchanged (same parent Schema)
- Only the topic_config's `key_schema_version` is updated

### Incompatible Schema Changes (Requires force=true)

If you need to use a schema version that is not backward compatible (e.g., changed field types, removed fields, added required fields without defaults), you must set `force = true` on the `axual_topic_config` resource.

**Example: Breaking changes**

Incompatible schema - changed field type and added required field:
```json
{
  "type": "record",
  "name": "GitOpsTestKey",
  "namespace": "io.axual.qa.keys",
  "fields": [
    { "name": "keyId", "type": "long" },
    { "name": "requiredField", "type": "string" }
  ]
}
```

Terraform configuration:
```hcl
resource "axual_schema_version" "key_incompatible" {
  body    = file("schemas/key_incompatible.avsc")
  version = "99.0.0"
}

resource "axual_topic_config" "logs_in_dev" {
  topic              = axual_topic.logs.id
  key_schema_version = axual_schema_version.key_incompatible.id
  force              = true  # Required for incompatible changes
}
```

Without `force = true`, you will receive an error:
```
Incompatible schema_version change identified. Retry with force=true in the axual_topic_config resource
```

## Changing the Topic's Base Schema

A topic once created with a specific key or value schema must continue to use the same schema for its lifetime. **If the schema itself needs changing then there is no option but to delete and recreate the topic.** This is by design and not a bug.

If you attempt to change `axual_topic.key_schema` or `axual_topic.value_schema` while a topic_config exists, you will receive an error:
```
Value Schema cannot be changed because the stream has active stream configuration.
```

To change the base schema:
1. Delete all `axual_topic_config` resources for the topic
2. Delete the `axual_topic` resource
3. Recreate the topic with the new schema
4. Recreate the topic configs

## Why Schema Body Updates Are Not Allowed

You cannot update the `body` of an existing `axual_schema_version`. This is by design for the following reasons:

1. **Version integrity**: Schema versions are immutable to ensure consistency. If you need a different schema body, create a new schema version.

2. **Cascading impact**: A schema may be used by many topics, and those topics may have many topic configs across different environments. Each topic config results in ACLs being configured in Kafka brokers.

3. **Safety**: Automatically deleting and recreating schemas would cascade to deleting topics, topic configs, and ACLs. This could cause significant unintended data loss and service disruption.

**The correct approach** is to create a new `axual_schema_version` resource with:
- The same `namespace + name` (to belong to the same parent Schema)
- A different `version` string
- The updated schema body

Then update your `axual_topic_config` resources to reference the new schema version.
