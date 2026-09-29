---
page_title: "Connector Application Support"
---

## Connector Application Support

A Connector application runs either on **legacy Axual Connect (AC)** or on a **registered Kafka
Connect cluster (KC)** (see the [`axual_connect_cluster`](../data-sources/connect_cluster.md) data
source). Which one applies depends only on whether `target_id` is set on the
`axual_application_deployment`.

There are two ways to write the Terraform for it:

| | [Deployment first (recommended)](#deployment-first-recommended) | [Deployment last (the original flow)](#deployment-last-the-original-flow) |
| --- | --- | --- |
| Order | deployment → principal or credential → grant + approval → `axual_application_deployment_state` | principal → grant + approval → deployment |
| Who starts the connector | `axual_application_deployment_state` (`state = "RUNNING"` or `"STOPPED"`) | `axual_application_deployment`, on create and after every update |
| AC with a certificate | yes | yes |
| KC on an `MTLS` cluster | yes | yes |
| KC on a `SASL_SCRAM` cluster | yes | **no** |
| Drift (a connector stopped outside Terraform, or failed) | shown in the plan, fixed by the next apply | not detected |

The deployment-first order is the order the Self-Service UI uses: the deployment target is chosen
first, and everything else is added to it. It is the only order that works on a `SASL_SCRAM` Kafka
Connect cluster, because Platform Manager creates that credential only for a deployment whose
target is already stored. On a KC `MTLS` cluster it also writes the certificate's private key
straight to the Connect cluster's Vault.

Existing configurations in the original flow keep working unchanged: `autostart` defaults to `true`.

## Deployment first (recommended)

1. `axual_application_deployment` with `autostart = false`: stores the target and the configs. It
   needs no principal, credential or grant, and does not start the connector.
2. `axual_application_principal` (AC, or KC on an `MTLS` cluster) or `axual_application_credential`
   with `target = "KAFKA"` (KC on a `SASL_SCRAM` cluster), with
   `depends_on = [axual_application_deployment...]`.
3. `axual_application_access_grant`, with `depends_on` on the principal or credential, and
   `axual_application_access_grant_approval`.
4. [`axual_application_deployment_state`](../resources/application_deployment_state.md) with
   `state = "RUNNING"`, with `depends_on` on the approval and on the principal or credential.

The `depends_on` entries also decide the destroy order, which runs backwards: the state resource
stops (and resets) the connector, then the grant is revoked, then the principal or credential is
removed, and the deployment last. Platform Manager refuses to delete the last principal while a
grant is still approved, so keep the grant's `depends_on` on the principal or credential.

Updating the configs of a deployment with `autostart = false` stops the connector if it is running,
updates the configs, and starts it again only if it was running before.

### Example: AC, KC on MTLS and KC on SASL_SCRAM

```hcl
data "axual_instance" "dta" {
  name = "DTA"
}

data "axual_connect_cluster" "mtls" {
  instance_id = data.axual_instance.dta.id
  cluster_id  = "YOUR_KAFKA_CLUSTER_ID"
  name        = "jsonlog"
}

data "axual_connect_cluster" "sasl" {
  instance_id = data.axual_instance.dta.id
  cluster_id  = "YOUR_KAFKA_CLUSTER_ID"
  name        = "sasl"
}

locals {
  connectors = {
    # target_id = null: legacy Axual Connect
    ac_tls  = { target_id = null, target_version = null, auth = "certificate" }
    kc_tls  = { target_id = data.axual_connect_cluster.mtls.id, target_version = "1.0.0", auth = "certificate" }
    kc_sasl = { target_id = data.axual_connect_cluster.sasl.id, target_version = "1.0.0", auth = "sasl" }
  }
}

resource "axual_application" "connector" {
  for_each          = local.connectors
  name              = "example-${each.key}"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "example${replace(each.key, "_", "")}"
  application_id    = "example.${each.key}"
  owners            = axual_group.team-integrations.id
  type              = "SINK"
  visibility        = "Public"
}

# 1. The deployment: target and configs, not started.
resource "axual_application_deployment" "connector" {
  for_each       = local.connectors
  application    = axual_application.connector[each.key].id
  environment    = axual_environment.example.id
  target_id      = each.value.target_id
  target_version = each.value.target_version
  autostart      = false
  configs = {
    "topics"          = axual_topic.example.name,
    "endpoint"        = "https://your-http-endpoint.example.com/",
    "key.converter"   = "org.apache.kafka.connect.storage.StringConverter",
    "value.converter" = "org.apache.kafka.connect.storage.StringConverter",
    "tasks.max"       = "1",
  }
}

# 2. The principal (certificate) or the credential (SASL_SCRAM), now that the target is known.
resource "axual_application_principal" "connector" {
  for_each    = { for k, v in local.connectors : k => v if v.auth == "certificate" }
  application = axual_application.connector[each.key].id
  environment = axual_environment.example.id
  principal   = file("certs/example-connector-cert.pem")
  private_key = file("certs/example-connector-private-key.key")
  active      = true
  depends_on  = [axual_application_deployment.connector]
}

resource "axual_application_credential" "connector" {
  for_each    = { for k, v in local.connectors : k => v if v.auth == "sasl" }
  application = axual_application.connector[each.key].id
  environment = axual_environment.example.id
  target      = "KAFKA"
  depends_on  = [axual_application_deployment.connector]
}

# 3. Topic access.
resource "axual_application_access_grant" "connector" {
  for_each    = local.connectors
  application = axual_application.connector[each.key].id
  topic       = axual_topic.example.id
  environment = axual_environment.example.id
  access_type = "CONSUMER"
  depends_on = [
    axual_topic_config.example,
    axual_application_principal.connector,
    axual_application_credential.connector,
  ]
}

resource "axual_application_access_grant_approval" "connector" {
  for_each                 = local.connectors
  application_access_grant = axual_application_access_grant.connector[each.key].id
}

# 4. Run it.
resource "axual_application_deployment_state" "connector" {
  for_each               = local.connectors
  application_deployment = axual_application_deployment.connector[each.key].id
  state                  = "RUNNING"
  depends_on = [
    axual_application_access_grant_approval.connector,
    axual_application_principal.connector,
    axual_application_credential.connector,
  ]
}
```

To stop a connector, set `state = "STOPPED"` and apply. To read what it is doing right now, use the
read-only `current_state` attribute.

## Deployment last (the original flow)

This flow works for AC and for KC on an `MTLS` cluster, and it is how existing configurations are
written. It needs no `axual_application_deployment_state`: creating the deployment starts it.

To create and start a connector application, we need these Axual Terraform resources:
  1. `axual_topic`
  2. `axual_environment`
  3. `axual_application`
      - These are connector application specific properties: 
      - `application_type = "Connector"`
        - `application_class` defined with a plugin class name. All supported plugin class names are listed here: https://docs.axual.io/connect/Axual-Connect/connect-plugins-catalog/connect-plugins-catalog.html.
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
       - Creating `axual_application_deployment` starts the connector, provided an active `axual_application_principal` already exists for this application and environment - create and activate the principal first.
       - Updating `axual_application_deployment` stops the connector if it was running, updates the config, and starts it.
       - Deleting `axual_application_deployment` stops the connector if it was running, then deletes it.
     - For a **KC** cluster, also set `target_id` (the [`axual_connect_cluster`](../data-sources/connect_cluster.md) data source's `id`) and `target_version` (the plugin version to deploy).
     - A KC cluster configured for `SASL_SCRAM` is refused in this flow with a message pointing to the deployment-first flow.

To read more about Connect Applications on Axual Platform: https://docs.axual.io/connect/Axual-Connect/index-developer.html

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
  active = true
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

resource "axual_application_access_grant_approval" "connector_axual_application_access_grant_approval-logsource-1" {
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
    "config.action.reload" = "restart",
    "tasks.max" = "1",
    "errors.log.include.messages" = "false",
    "errors.log.enable" = "false",
    "errors.retry.timeout" = "0",
    "errors.retry.delay.max.ms" = "60000",
    "errors.tolerance" = "none"
  }
  depends_on = [
    axual_application_access_grant_approval.connector_axual_application_access_grant_approval-logsource-1,
  ]
}

```

## Moving a connector from the original flow to deployment first

1. Add `autostart = false` to the `axual_application_deployment`. The plan shows an in-place change
   of `autostart` only; applying it changes nothing on the platform.
2. Add an `axual_application_deployment_state` for the deployment with `state = "RUNNING"`.
   Creating it for a connector that is already running sends nothing to the platform. You can also
   import it instead: `terraform import axual_application_deployment_state.<LOCAL NAME> <APPLICATION DEPLOYMENT UID>`.
3. Move the `depends_on` entries to the deployment-first order shown above.
