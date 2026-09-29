---
page_title: "Connector Application Support"
---

## Connector Application Support

A Connector application runs a Kafka Connect connector (a source or a sink) that Axual manages for
you. It can run in two places:

| | Axual Connect (AC) | Kafka Connect cluster (KC) |
| --- | --- | --- |
| What it is | One shared Connect runtime for the whole instance, set up by the platform operator. All connectors of the instance run on it. | A separate Kafka Connect cluster. A Tenant Admin registers it in Self Service for one instance and Kafka cluster combination, and chooses which teams may deploy connectors on it. One instance and Kafka cluster combination can have several Kafka Connect clusters. |
| Platform Manager version | all versions | 16.0.0 or later |
| How you choose it | leave `target_id` out of `axual_application_deployment` | set `target_id` to the id of the [`axual_connect_cluster`](../data-sources/connect_cluster.md) data source, and `target_version` to the plugin version |
| How the connector logs in to Kafka | a certificate: `axual_application_principal` (TLS) | the Kafka Connect cluster decides: `MTLS` uses an `axual_application_principal` (TLS), `SASL_SCRAM` uses an `axual_application_credential` (SASL) |
| SASL | not supported | supported on a `SASL_SCRAM` cluster |

Platform Manager 16.0.0 added support for Kafka Connect clusters. With an older Platform Manager,
only AC is available.

### Two ways to start a connector

| | `autostart = true` (default) | `autostart = false` |
| --- | --- | --- |
| What starts the connector | creating or updating the `axual_application_deployment` | an [`axual_application_deployment_state`](../resources/application_deployment_state.md) with `state = "RUNNING"` |
| Order | principal → grant → deployment | deployment → principal or credential → grant → deployment state |
| AC+TLS | yes | yes |
| KC+TLS and KC+SASL | no | yes |
| Stop the connector from Terraform | no | `state = "STOPPED"` |
| Notice a connector that stopped or failed | no | yes, in the next plan |

A KC connector needs `autostart = false`, because Platform Manager can only set up its principal
or credential after the deployment exists. With `autostart = true` the provider stops with
`Kafka Connect cluster needs autostart = false`.

### Example files

Each example is a full, separate configuration with its own environment, topic and application. It
uses the HTTP sink connector. Replace the `YOUR_*` values.

| Case | Example |
| --- | --- |
| AC+TLS with `autostart = true` | [`examples/connector-application/ac-tls-autostart`](https://github.com/Axual/terraform-provider-axual/tree/master/examples/connector-application/ac-tls-autostart) |
| AC+TLS with a deployment state | [`examples/connector-application/ac-tls-deployment-state`](https://github.com/Axual/terraform-provider-axual/tree/master/examples/connector-application/ac-tls-deployment-state) |
| KC+TLS with a deployment state | [`examples/connector-application/kc-tls`](https://github.com/Axual/terraform-provider-axual/tree/master/examples/connector-application/kc-tls) |
| KC+SASL with a deployment state | [`examples/connector-application/kc-sasl`](https://github.com/Axual/terraform-provider-axual/tree/master/examples/connector-application/kc-sasl) |

The snippets below show only the connector part. The environment, topic, topic config and
application are the same in every example:

```hcl
resource "axual_application" "example" {
  name              = "tf-example-connector"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "exampleconnector"
  application_id    = "tf.example.connector"
  owners            = data.axual_group.example.id
  type              = "SINK"
  visibility        = "Public"
}
```

- `application_type = "Connector"`.
- `application_class` is the plugin class. The plugins that AC offers are listed in the
  [Axual Connect plugins catalog](https://docs.axual.io/connect/Axual-Connect/connect-plugins-catalog/connect-plugins-catalog.html).
  A Kafka Connect cluster offers the plugins that are installed on it.
- `type` is `"SINK"` or `"SOURCE"`.

## AC+TLS

AC uses a certificate: an `axual_application_principal` with `private_key`. The private key is
sensitive and is not shown in the output. The `configs` of the deployment are sensitive too. AC does
not support SASL: Platform Manager refuses an `axual_application_credential` for an AC connector with
"Select a target Kafka Connect cluster first."

AC supports both ways to start a connector.

### AC+TLS with `autostart = true`

This is the original flow, and existing configurations keep working without a change.

1. `axual_application_principal`, with `active = true`.
2. `axual_application_access_grant`, with `depends_on` on the topic config and the principal, and
   `axual_application_access_grant_approval`.
3. `axual_application_deployment`, with `depends_on` on the approval. Creating it starts the
   connector. The provider checks first that an active principal and an approved grant exist.

```hcl
resource "axual_application_principal" "example" {
  environment = axual_environment.example.id
  application = axual_application.example.id
  principal   = file("certs/connector-cert.pem")        # the connector's certificate (chain), PEM
  private_key = file("certs/connector-private-key.key") # the connector's private key, PEM
  active      = true
}

resource "axual_application_access_grant" "example" {
  application = axual_application.example.id
  topic       = axual_topic.example.id
  environment = axual_environment.example.id
  access_type = "CONSUMER"
  depends_on  = [axual_topic_config.example, axual_application_principal.example]
}

resource "axual_application_access_grant_approval" "example" {
  application_access_grant = axual_application_access_grant.example.id
}

# No target_id: legacy Axual Connect. Creating it starts the connector.
resource "axual_application_deployment" "example" {
  application = axual_application.example.id
  environment = axual_environment.example.id
  configs = {
    "topics"          = axual_topic.example.name,
    "endpoint"        = "https://YOUR_HTTP_ENDPOINT/", # where the HTTP sink connector sends records
    "key.converter"   = "org.apache.kafka.connect.storage.StringConverter",
    "value.converter" = "org.apache.kafka.connect.storage.StringConverter",
    "tasks.max"       = "1",
  }
  depends_on = [axual_application_access_grant_approval.example]
}
```

- Updating the deployment stops the connector, saves the new configs, and starts it again.
- Deleting the deployment stops the connector and then deletes it.

### AC+TLS with a deployment state

1. `axual_application_deployment` with `autostart = false`. It needs no principal or grant yet.
2. `axual_application_principal`, with `depends_on` on the deployment.
3. `axual_application_access_grant`, with `depends_on` on the topic config and the principal, and
   `axual_application_access_grant_approval`.
4. `axual_application_deployment_state`, with `depends_on` on the approval and the principal.

```hcl
resource "axual_application_deployment" "example" {
  application = axual_application.example.id
  environment = axual_environment.example.id
  autostart   = false
  configs = {
    "topics"          = axual_topic.example.name,
    "endpoint"        = "https://YOUR_HTTP_ENDPOINT/", # where the HTTP sink connector sends records
    "key.converter"   = "org.apache.kafka.connect.storage.StringConverter",
    "value.converter" = "org.apache.kafka.connect.storage.StringConverter",
    "tasks.max"       = "1",
  }
}

resource "axual_application_principal" "example" {
  environment = axual_environment.example.id
  application = axual_application.example.id
  principal   = file("certs/connector-cert.pem")        # the connector's certificate (chain), PEM
  private_key = file("certs/connector-private-key.key") # the connector's private key, PEM
  active      = true
  depends_on  = [axual_application_deployment.example]
}

resource "axual_application_access_grant" "example" {
  application = axual_application.example.id
  topic       = axual_topic.example.id
  environment = axual_environment.example.id
  access_type = "CONSUMER"
  depends_on  = [axual_topic_config.example, axual_application_principal.example]
}

resource "axual_application_access_grant_approval" "example" {
  application_access_grant = axual_application_access_grant.example.id
}

resource "axual_application_deployment_state" "example" {
  application_deployment = axual_application_deployment.example.id
  state                  = "RUNNING"
  depends_on             = [axual_application_access_grant_approval.example, axual_application_principal.example]
}
```

## KC+TLS and KC+SASL

A KC connector needs Platform Manager 16.0.0 or later, and it only works with `autostart = false`
and an `axual_application_deployment_state`.

Look up the registered Kafka Connect cluster with the [`axual_connect_cluster`](../data-sources/connect_cluster.md)
data source. Its `auth_method` decides the authentication:

- `MTLS`: an `axual_application_principal` with a certificate (KC+TLS). Because the deployment and
  its target exist first, Platform Manager writes the private key straight to the Kafka Connect
  cluster's Vault.
- `SASL_SCRAM`: an `axual_application_credential` with `target = "KAFKA"` (KC+SASL). Platform
  Manager generates the username and password, and it can only do that when the deployment and its
  target already exist.

The steps are the same for both:

1. `axual_application_deployment` with `target_id`, `target_version` and `autostart = false`.
2. The principal (MTLS) or the credential (SASL_SCRAM), with `depends_on` on the deployment.
3. `axual_application_access_grant`, with `depends_on` on the topic config and the principal or
   credential, and `axual_application_access_grant_approval`.
4. `axual_application_deployment_state`, with `depends_on` on the approval and the principal or
   credential.

### KC+TLS

```hcl
data "axual_connect_cluster" "example" {
  instance_id = data.axual_instance.example.id
  cluster_id  = "YOUR_KAFKA_CLUSTER_ID" # the Kafka cluster the Kafka Connect cluster is registered on
  name        = "YOUR_CONNECT_CLUSTER_NAME" # a Kafka Connect cluster with auth_method MTLS
}

resource "axual_application_deployment" "example" {
  application    = axual_application.example.id
  environment    = axual_environment.example.id
  target_id      = data.axual_connect_cluster.example.id
  target_version = "1.0.0"
  autostart      = false
  configs = {
    "topics"          = axual_topic.example.name,
    "endpoint"        = "https://YOUR_HTTP_ENDPOINT/", # where the HTTP sink connector sends records
    "key.converter"   = "org.apache.kafka.connect.storage.StringConverter",
    "value.converter" = "org.apache.kafka.connect.storage.StringConverter",
    "tasks.max"       = "1",
  }
}

resource "axual_application_principal" "example" {
  environment = axual_environment.example.id
  application = axual_application.example.id
  principal   = file("certs/connector-cert.pem")        # the connector's certificate (chain), PEM
  private_key = file("certs/connector-private-key.key") # the connector's private key, PEM
  active      = true
  depends_on  = [axual_application_deployment.example]
}

resource "axual_application_access_grant" "example" {
  application = axual_application.example.id
  topic       = axual_topic.example.id
  environment = axual_environment.example.id
  access_type = "CONSUMER"
  depends_on  = [axual_topic_config.example, axual_application_principal.example]
}

resource "axual_application_access_grant_approval" "example" {
  application_access_grant = axual_application_access_grant.example.id
}

resource "axual_application_deployment_state" "example" {
  application_deployment = axual_application_deployment.example.id
  state                  = "RUNNING"
  depends_on             = [axual_application_access_grant_approval.example, axual_application_principal.example]
}
```

### KC+SASL

```hcl
data "axual_connect_cluster" "example" {
  instance_id = data.axual_instance.example.id
  cluster_id  = "YOUR_KAFKA_CLUSTER_ID" # the Kafka cluster the Kafka Connect cluster is registered on
  name        = "YOUR_CONNECT_CLUSTER_NAME" # a Kafka Connect cluster with auth_method SASL_SCRAM
}

resource "axual_application_deployment" "example" {
  application    = axual_application.example.id
  environment    = axual_environment.example.id
  target_id      = data.axual_connect_cluster.example.id
  target_version = "1.0.0"
  autostart      = false
  configs = {
    "topics"          = axual_topic.example.name,
    "endpoint"        = "https://YOUR_HTTP_ENDPOINT/", # where the HTTP sink connector sends records
    "key.converter"   = "org.apache.kafka.connect.storage.StringConverter",
    "value.converter" = "org.apache.kafka.connect.storage.StringConverter",
    "tasks.max"       = "1",
  }
}

resource "axual_application_credential" "example" {
  environment = axual_environment.example.id
  application = axual_application.example.id
  target      = "KAFKA"
  depends_on  = [axual_application_deployment.example]
}

resource "axual_application_access_grant" "example" {
  application = axual_application.example.id
  topic       = axual_topic.example.id
  environment = axual_environment.example.id
  access_type = "CONSUMER"
  depends_on  = [axual_topic_config.example, axual_application_credential.example]
}

resource "axual_application_access_grant_approval" "example" {
  application_access_grant = axual_application_access_grant.example.id
}

resource "axual_application_deployment_state" "example" {
  application_deployment = axual_application_deployment.example.id
  state                  = "RUNNING"
  depends_on             = [axual_application_access_grant_approval.example, axual_application_credential.example]
}
```

## Working with a deployment state

These points apply to AC+TLS with a deployment state, KC+TLS and KC+SASL.

- To stop a connector, set `state = "STOPPED"` and apply. The connector is paused and stays on the
  Connect cluster, so `state = "RUNNING"` starts it again quickly.
- While a connector is stopped, its active principal cannot be deleted or replaced. To replace the
  principal, keep the connector running, or remove the deployment state first.
- Removing the `axual_application_deployment_state` stops the connector and removes it from the
  Connect cluster. The deployment and its configs stay. Add the deployment state again to start it.
- Changing the `configs` stops a running connector, saves the new configs, and starts it again. A
  stopped connector stays stopped, and it gets the new configs when it is started.
- `current_state` shows what the connector is doing right now, for example `Running` or `Stopped`.
  If it was stopped outside Terraform, or it failed, the next plan shows a change back to the wanted
  `state`, and the next apply fixes it.

## The `depends_on` entries also set the destroy order

A destroy runs in the opposite order. It stops the connector, then revokes the grant, then deletes
the principal or credential, and deletes the deployment last. Keep these entries:

- The grant `depends_on` the principal or credential. Platform Manager refuses to delete the last
  principal while a grant is still approved.
- The deployment state `depends_on` the approval and the principal or credential, so the connector
  is stopped before its access is removed.
- The topic `depends_on` the environment, as in the example files. A topic has no reference to its
  environment, so without it Terraform deletes both at the same time, and the two deletes can fail
  on the same access grant.

## Moving an AC+TLS connector to a deployment state

1. Add `autostart = false` to the `axual_application_deployment`. The plan shows an in-place change
   of `autostart` only. Applying it changes nothing on the platform.
2. Add an `axual_application_deployment_state` for the deployment with `state = "RUNNING"`.
   Creating it for a connector that is already running sends nothing to the platform. You can also
   import it: `terraform import axual_application_deployment_state.<LOCAL NAME> <APPLICATION DEPLOYMENT UID>`.
3. Change the `depends_on` entries to the deployment-state order shown above.

To read more about Connector applications on the Axual Platform: https://docs.axual.io/connect/Axual-Connect/index-developer.html
