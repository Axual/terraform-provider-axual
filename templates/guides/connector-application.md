---
page_title: "Terraform setup with Axual Trial environment"
---

## Connector Application Support

- To create and start a connector application, we need these Axual Terraform resources:
  - **axual_topic**
  - **axual_environment**
  - **axual_application**. These are connector application specific properties: 
    - application_type     = "Connector"
    - application_class defined with a plugin class name. All supported plugin class names are listed here: https://docs.axual.io/connect/Axual-Connect/developer/connect-plugins-catalog/connect-plugins-catalog.html
      - For example: application_class = "com.couchbase.connect.kafka.CouchbaseSinkConnector"
    - type="SINK" or type="SOURCE"
  - **axual_application_principal** These are connector application specific properties:
    - private_key = file("certs/example-connector.key")
      - Please note that the value needs to be a string
      - Private key(private_key) is marked as "Sensitive" and doesn't get shown in server logs
  - **axual_application_access_grant**
  - **axual_application_access_grant_approval**
  - **axual_application_deployment**
    - Please include **depends_on** like in the example below so Terraform Provider knows the correct order of execution when creating or deleting multiple resources.
    - Configuration(configs) is marked as "Sensitive" and doesn't get shown in server logs
    - Creating axual_application_deployment starts the connector
    - Updating axual_application_deployment stops the connector if it was running, updates the config, and starts it
    - Deleting axual_application_deployment stops the connector if it was running, then deletes it

  - To read more about Connect Applications on Axual Platform: https://docs.axual.io/connect/Axual-Connect/developer/index-developer.html

### Example Resources:
```shell
resource "axual_topic" "test-topic" {
  name = test-topic-name"
  key_type = "String"
  value_type = "String"
  owners = axual_group.team-test.id
  retention_policy = "delete"
  properties = { }
  description = "Logs from all applications"
}

resource "axual_environment" "test_env" {
  name = "test-env-name"
  short_name = "test-env-short-name"
  description = "test env description"
  color = "#4686f0"
  visibility = "Public"
  authorization_issuer = "Auto"
  instance = "1be6269156d14ab09f40ea5133316a33"
  owners = axual_group.team-test.id
}

resource "axual_application" "connector_test_application" {
  name    = "Connector Application Name"
  application_type     = "Connector"
  application_class = "com.couchbase.connect.kafka.CouchbaseSinkConnector"
  short_name = "connector_app_short_name"
  application_id = "connector_app1"
  owners = axual_group.team-test.id
  visibility = "Public"
  description = "Connect Application description"
  type="SINK"
#  depends_on = [axual_topic_config.logs_in_production, axual_topic.support] # This is a workaround when all resources get deleted at once, to delete topic_config and topic before application. Mentioned in index.md
}

resource "axual_application_principal" "test_application_principal" {
  environment = axual_environment.test_env.id
  application = axual_application.connector_test_application.id
  principal = file("certs/example-connector.pem")
  private_key = file("certs/example-connector.key")
}

resource "axual_application_access_grant" "test_application_access_grant" {
  application = axual_application.connector_test_application.id
  topic = axual_topic.test-topic.id
  environment = axual_environment.test_env.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.dev_dashboard_in_dev_principal ]
}

resource "axual_application_access_grant_approval" "test_application_access_grant_approval" {
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_dev.id
}

resource "axual_application_deployment" "dev_dashboard_in_dev_principal" {
  environment = axual_environment.test_env.id
  application = axual_application.connector_test_application.id
  configs = {
    "config.action.reload"= "restart",
    "header.converter"= "",
    "key.converter"= "",
    "tasks.max"= "1",
    "topics"= "3",
    "topics.regex"= "",
    "value.converter"= "",
    "couchbase.bootstrap.timeout"= "30s",
    "couchbase.bucket"= "2",
    "couchbase.network"= "test@test.com",
    "couchbase.seed.nodes"= "1",
    "couchbase.username"= "q",
    "couchbase.durability"= "NONE",
    "couchbase.persist.to"= "NONE",
    "couchbase.replicate.to"= "NONE",
    "errors.deadletterqueue.context.headers.enable"= "false",
    "errors.deadletterqueue.topic.name"= "",
    "errors.deadletterqueue.topic.replication.factor"= "3",
    "errors.log.enable"= "false",
    "errors.log.include.messages"= "false",
    "errors.retry.delay.max.ms"= "60000",
    "errors.retry.timeout"= "0",
    "errors.tolerance"= "none",
    "couchbase.log.document.lifecycle"= "false",
    "couchbase.log.redaction"= "NONE",
    "couchbase.n1ql.create.document"= "true",
    "couchbase.n1ql.operation"= "UPDATE",
    "couchbase.n1ql.where.fields"= "",
    "predicates"= "",
    "couchbase.client.certificate.password"= "[hidden]",
    "couchbase.client.certificate.path"= "",
    "couchbase.enable.hostname.verification"= "true",
    "couchbase.enable.tls"= "false",
    "couchbase.trust.certificate.path"= "",
    "couchbase.trust.store.password"= "[hidden]",
    "couchbase.trust.store.path"= "",
    "couchbase.default.collection"= "_default._default",
    "couchbase.document.expiration"= "0",
    "couchbase.document.id"= "",
    "couchbase.document.mode"= "DOCUMENT",
    "couchbase.remove.document.id"= "false",
    "couchbase.retry.timeout"= "0",
    "couchbase.sink.handler"= "com.couchbase.connect.kafka.handler.sink.UpsertSinkHandler",
    "couchbase.topic.to.collection"= "",
    "couchbase.subdocument.create.document"= "true",
    "couchbase.subdocument.create.path"= "true",
    "couchbase.subdocument.operation"= "UPSERT",
    "couchbase.subdocument.path"= "",
    "transforms"= "",
    "couchbase.password"= "1",
  }
  depends_on = [ axual_application_principal.test_application_principal,
    axual_application_access_grant.test_application_access_grant,
    axual_application_access_grant_approval.test_application_access_grant_approval
  ]
}
```