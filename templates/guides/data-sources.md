---
page_title: "Using Data Sources"
---

Currently we support data sources for the folloing resources:
- axual_group
- axual_environment
- axual_topic
- axual_application
- axual_application_access_grant
- axual_schema_version


### Examples usage 

- To define a `axual_group` data source, provide the group name:

```shell
data "axual_group" "frontend_developers" {
 name = "Frontend Developers"
}
```
Now we can use this data source when creating a resource: 

```shell
resource "axual_topic" "logs" {
  name = "logs"
  key_type = "String"
  value_type = "String"
  owners = data.axual_group.frontend_developers.id
  retention_policy = "delete"
  properties = { }
  description = "Dev topic of type string"
}
```

- To define  a `axual_environment` data source, provide the environment name:

```shell
data "axual_environment" "dev" {
  name = "dev"
}
```
Now we can use this data source when creating a resource: 

```shell
resource "axual_topic_config" "logs_in_dev" {
  partitions = 1
  retention_time = 1001000
  topic = axual_topic.logs.id
  environment = data.axual_environment.dev.id
  properties = {"segment.ms"="60002", "retention.bytes"="100"}
}
```

- To define  a `axual_topic` data source, provide the topic name:

```shell
data "axual_topic" "logs" {
 name = "logs"
}
```
Now we can use this data source when creating a resource: 

```shell
resource "axual_topic_config" "logs_in_dev" {
  partitions = 1
  retention_time = 1001000
  topic = data.axual_topic.logs.id
  environment = data.axual_environment.dev.id
  properties = {"segment.ms"="60002", "retention.bytes"="100"}
}
```

- To define  a `axual_application` data source, provide the application name:

```shell
data "axual_application" "logs_producer" {
  name = "logs_producer"
}
```
Now we can use this data source when creating a resource: 

```shell
resource "axual_application_access_grant" "logs_producer_produce_to_logs_in_dev" {
  application = data.axual_application.logs_producer.id
  topic = data.axual_topic.logs.id
  environment = data.axual_environment.dev.id
  access_type = "PRODUCER"
}
```

- To define  a `axual_schema_version` data source, the schema full name (<NAMESPACE>.<NAME>) and the version:

```shell
data "axual_schema_version" "ApplicationV1" {
   full_name="io.axual.qa.general.Application"
   version = "1.0.0"
}
```
Now we can use this data source when creating a resource: 

```shell
resource "axual_topic" "avro_topic" {
  name = "avro_topic"
  key_type = "AVRO"
  key_schema = data.axual_schema_version.ApplicationV1.schema_id
  value_type = "AVRO"
  value_schema = data.axual_schema_version.ApplicationV1.schema_id
  owners = data.axual_group.frontend_developers.id
  retention_policy = "delete"
  properties = { }
  description = "avro topic created using external data source"
}
```

- To define  a `axual_application_access_grant` data source, provide the application id, the topic id, environment id and the access type (PRODUCER, CONSUMER):

```shell
data "axual_application_access_grant" "logs_producer_produce_to_logs_in_dev" {
   application = axual_application.tfds_app.id
  topic = data.axual_topic.algorithms.id
  environment = data.axual_environment.dev.id
  access_type = "PRODUCER"
}
```
Now we can use this data source when creating a resource: 

```shell
resource "axual_application_access_grant_approval" "logs_producer_produce_to_logs_in_dev_approval" {
  application_access_grant = "data.logs_producer_produce_to_logs_in_dev.id"
}
```
