---
page_title: "Connector Application Support"
---

## Connector Application Support

To create and start a connector application, we need these Axual Terraform resources:
  1. `axual_topic`
  2. `axual_environment`
  3. `axual_application` 
      - These are connector application specific properties: 
      - `application_type = "Connector"`
        - `application_class` defined with a plugin class name. All supported plugin class names are listed here: https://docs.axual.io/connect/Axual-Connect/developer/connect-plugins-catalog/connect-plugins-catalog.html.
          - For example: `application_class = "com.couchbase.connect.kafka.CouchbaseSinkConnector"`.
        - `type="SINK"` or `type="SOURCE"`
  4. `axual_application_principal` 
      - These are connector application specific properties:
      - `private_key = file("certs/example-connector.key")`
          - Please note that the value needs to be a string.
          - Private key(`private_key`) is marked as "Sensitive" and doesn't get shown in server logs.
  5. `axual_application_access_grant`
  6. `axual_application_access_grant_approval`
  7. `axual_application_deployment`
     - Please include `depends_on` like in the example below so Terraform Provider knows the correct order of execution when creating or deleting multiple resources.
       - Configuration(`configs`) is marked as "Sensitive" and doesn't get shown in server logs.
       - Creating `axual_application_deployment` starts the connector.
       - Updating `axual_application_deployment` stops the connector if it was running, updates the config, and starts it.
       - Deleting `axual_application_deployment` stops the connector if it was running, then deletes it.

To read more about Connect Applications on Axual Platform: https://docs.axual.io/connect/Axual-Connect/developer/index-developer.html

### Example Resources:
- When trying out this example, make sure to:
 - replace `members` UID
 - replace `environment` UID or create/import your own environment. The environment for this example has to have: `Visibility`: Public and `Authorization Issuer`: Grant Owner.
 - replace CERT and Private key required for Application Principal.
- Also notice that in `axual_application_deployment` the configuration option: `"topic" = "tf-testing-source-1-2"` needs to match the topic name. 

```shell

locals {
  retention_time = 604800000
}

resource "axual_group" "team-integrations" {
  name          = "team-Integrations2"
  phone_number  = "+6112356789"
  email_address = "test.user@axual.com"
  members       = [
    "40a11a71e0374cb8986d4bf7394f2ccb",
  ]
}

resource "axual_topic" "topic-log-source-1" {
  name = "tf-testing-source-1-2"
  key_type = "String"
  value_type = "String"
  owners = axual_group.team-integrations.id
  retention_policy = "delete"
  properties = { }
  description = "Demo of deploying a topic via Terraform"
}

resource "axual_topic_config" "topic-log-source-1-in-cicd" {
  partitions = 1
  retention_time = local.retention_time
  topic = axual_topic.topic-log-source-1.id
  environment = "09412f2ac3cb436598c68f5822ec3572"
  properties = { "segment.ms"="600012", "retention.bytes"="-1" }
}

resource "axual_application" "connector_test_application" {
  name    = "tf-logsource2"
  application_type     = "Connector"
  application_class = "org.apache.kafka.connect.axual.utils.LogSourceConnector"
  short_name = "tf_logsource2"
  application_id = "terraform-log-source2"
  owners = axual_group.team-integrations.id
  visibility = "Public"
  description = "Demo of deploying connector via Terraform"
  type="SOURCE"
}

resource "axual_application_principal" "connector_axual_application_principal" {
  environment = "09412f2ac3cb436598c68f5822ec3572"
  application = axual_application.connector_test_application.id
  principal = file("certs/example-connector-cert.pem")
  private_key = file("certs/example-connector-private-key.key")
}

resource "axual_application_access_grant" "connector_axual_application_access_grant_logsource-1" {
  application = axual_application.connector_test_application.id
  topic = axual_topic.topic-log-source-1.id
  environment =  "09412f2ac3cb436598c68f5822ec3572"
  access_type = "PRODUCER"
  depends_on = [
    axual_topic_config.topic-log-source-1-in-cicd,
    axual_application_principal.connector_axual_application_principal,
  ]
}

resource "axual_application_access_grant_approval" "connector_axual_application_access_grant_approval-logsource-1"{
  application_access_grant = axual_application_access_grant.connector_axual_application_access_grant_logsource-1.id
}

resource "axual_application_deployment" "connector_axual_application_deployment" {
  environment =  "09412f2ac3cb436598c68f5822ec3572"
  application = axual_application.connector_test_application.id
  configs = {
    "logger.name" = "tflogger",
    "throughput" = "10",
    "topic" = "tf-testing-source-1-2",
    "key.converter" = "StringConverter",
    "value.converter" = "StringConverter",
    "header.converter" = "",
    "config.action.reload" = "restart",
    "tasks.max" = "1",
    "errors.log.include.messages" = "false",
    "errors.log.enable" = "false",
    "errors.retry.timeout" = "0",
    "errors.retry.delay.max.ms" = "60000",
    "errors.tolerance" = "none",
    "predicates" = "",
    "topic.creation.groups" = "",
    "transforms" = ""
  }
  depends_on = [
    axual_application_access_grant_approval.connector_axual_application_access_grant_approval-logsource-1,
  ]
}

```