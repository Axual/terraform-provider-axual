---
page_title: "Using Data Sources"
---

Currently we support data sources for the folloing resources:
- axual_group
- axual_environment
- axual_topic
- axual_application
- axual_application_access_grant


### Examples usage 

- To define a `axual_group` data source:

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

- To define  a `axual_environment` data source:

```shell
data "axual_environment" "dev" {
  short_name = "dev"
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

- To define  a `axual_topic` data source:

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

- To define  a `axual_application` data source:

```shell
data "axual_application" "logs_producer" {
  short_name = "logs_producer"
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

- To define  a `axual_application_access_grant` data source:

```shell
data "axual_application_access_grant" "logs_producer_produce_to_logs_in_dev" {
 id = "cc56541c7e7449f99d06432431456c83"
}
```
Now we can use this data source when creating a resource: 

```shell
resource "axual_application_access_grant_approval" "logs_producer_produce_to_logs_in_dev_approval" {
  application_access_grant = "data.logs_producer_produce_to_logs_in_dev.id"
}
```
