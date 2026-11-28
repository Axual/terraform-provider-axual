# axual_instance_cluster (Resource)

The configuration of one Cluster in one Instance: its schema registry. The Cluster must be in the Instance's `clusters`. Platform Manager stores the credentials in Vault and does not contact the registry to check them.

## Usage
- The Cluster must be in the Instance's `clusters`. Use a reference such as `axual_instance.production.id`, so the Instance is created first and destroyed last.
- `certificate`, `private_key` and `truststore` take PEM. Platform Manager keeps them base64-encoded, the way the Self-Service UI sends them, so the provider does the encoding. An already base64-encoded value is also accepted.
- The password, private key and Keycloak admin password are stored in Vault. Platform Manager never returns them, so they are kept as written in the configuration and cannot be imported.
- Platform Manager does not contact the registry to check the URL or credentials. A wrong value shows up when a topic with a schema is configured.
- Leaving out `listeners` or `properties` keeps what the schema registry has. `listeners = {}` and `properties = {}` remove them.
- Creating the resource fails when the Instance-Cluster already has a schema registry, for example one set up in the Self-Service UI. Import it instead. When an apply cannot tell whether Platform Manager saved the registry (no answer, a `5xx`, or a failed read after the write), it keeps the resource as tainted, so the next apply replaces it instead of refusing it.
- Schema registry credentials of applications (`axual_application_credential` with `target = "SCHEMA_REGISTRY"`, or OAuth in the Self-Service UI) need an `APICURIO` registry with `keycloak` on at least one Cluster of the Instance: Platform Manager creates their clients in that Keycloak. Without it, Platform Manager refuses them with "Failed to create credentials for any Cluster". A `CONFLUENT` registry gets no application credentials: with no `APICURIO` registry in the Instance, Platform Manager answers "Schema Registry credential management is only supported for Apicurio." An OAuth credential also needs the Platform Manager setting `axual.application-management.apicurio.clientSecretSalt`; without it, Platform Manager answers "Schema Registry OAuth is not configured."
- Destroying the resource removes the schema registry from the Instance-Cluster. Platform Manager refuses it while any application on any environment of the Instance has a schema registry credential. It keeps the `keycloak` details and the Keycloak admin password until the Instance is deleted.
- Give the environments of the Instance `depends_on = [axual_instance_cluster.<name>]`. Platform Manager needs the schema registry before it creates a schema registry credential, and refuses to remove it while an application of the Instance has one.
- `client_endpoints` (Platform Manager 17.0.0 or later) are the registry URLs that applications use, by name. `used_by = "ALL_APPS"` endpoints are also shown to users and used by self-managed applications; `MANAGED_APPS` endpoints only by managed applications. Set either `client_endpoints` or `listeners`, not both: when the endpoints are sent, Platform Manager ignores `listeners`. With `listeners`, each name becomes an `ALL_APPS` endpoint with the registry's `api`.
- An endpoint keeps its identity in Platform Manager while its name stays the same: a changed `url`, `api` or `used_by` changes it in place. Renaming an endpoint removes it and adds a new one.
- Removing `client_endpoints` from the configuration keeps the endpoints in Platform Manager, because the attribute is also computed. To remove them all, set `client_endpoints = {}`. While the configuration sets neither `client_endpoints` nor `listeners`, endpoints added outside Terraform, for example in the Self-Service UI, are kept too.
- `api` (Platform Manager 17.0.0 or later) is the API Platform Manager itself uses at `url`. When it is not set, Platform Manager derives it from `type`: `CONFLUENT_V7` for `CONFLUENT`, and `APICURIO_V3` for `APICURIO`. Platform Manager refuses `api = "APICURIO_V2"`, because it no longer uses that API itself; `client_endpoints` still accept `APICURIO_V2`.
- Platform Manager 17.0.0 changes a stored `APICURIO_V2` to `APICURIO_V3`. A configuration with `api = "APICURIO_V2"` then plans a change that fails with `400`: use `APICURIO_V3`, or leave `api` out.

## Platform Manager version
- Platform Manager 15.0.9 or later. `api` and `client_endpoints` need Platform Manager 17.0.0 or later; with an older one, the plan fails. Leave them out there: they stay empty, and the provider sends nothing new.
- Platform Manager 15.0.x and 16.0.x use the Apicurio Registry 2.x API (`/apis/registry/v2`); Platform Manager 17.0.0 uses the 3.x API (`/apis/registry/v3`).
- Platform Manager does not move the schema registry secrets when the Instance's `short_name` or the Cluster's name changes. 15.0.x and 16.0.x then lose the whole schema registry; newer versions keep it but lose its password and private key. The next plan shows it, with a warning for a lost secret, and the next apply configures it again. Until then, topics with a schema fail.

## Required Roles
- TENANT_ADMIN

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cluster` (String) The ID of the Cluster. Changing it creates a new resource.
- `instance` (String) The ID of the Instance. Changing it creates a new resource.
- `schema_registry` (Attributes) The schema registry topics with schemas on this Cluster use. (see [below for nested schema](#nestedatt--schema_registry))

### Read-Only

- `id` (String) The Instance ID and the Cluster ID, as `<instance>/<cluster>`.

<a id="nestedatt--schema_registry"></a>
### Nested Schema for `schema_registry`

Required:

- `authentication` (String) How Platform Manager signs in to the registry: `Basic-Auth`, `MTLS` or `No-Auth`. `MTLS` is stored as `mTLS`, the spelling of the Self-Service UI.
- `type` (String) `APICURIO` or `CONFLUENT`.
- `url` (String) Base URL of the registry API Platform Manager uses, for example `https://apicurio.example.com/apis/registry/v3`.

Optional:

- `api` (String) The API the registry at `url` speaks: `APICURIO_V2` or `APICURIO_V3` for `APICURIO`, `CONFLUENT_V7` for `CONFLUENT`. Needs Platform Manager 17.0.0 or later, which refuses `APICURIO_V2`; `client_endpoints` still accept it. When it is not set, Platform Manager derives it from `type`: `CONFLUENT_V7` or `APICURIO_V3`. The provider keeps it until `type` or `version` changes.
- `certificate` (String) PEM client certificate for `MTLS`. Requires `private_key`.
- `client_endpoints` (Attributes Map) The registry endpoints applications use, by name. Needs Platform Manager 17.0.0 or later. When set, it replaces all endpoints of the Instance-Cluster, and an endpoint keeps its identity in Platform Manager while its name stays the same. Leaving it out keeps the endpoints Platform Manager has, and `{}` removes them all. When it is not set, the endpoints follow `listeners`. Conflicts with `listeners`, which Platform Manager ignores when the endpoints are sent. (see [below for nested schema](#nestedatt--schema_registry--client_endpoints))
- `keycloak` (Attributes) `APICURIO` only, and not with `No-Auth`: the Keycloak behind the registry. Platform Manager creates the client of every schema registry credential of an application there, Basic-Auth and OAuth, so at least one Cluster of the Instance needs it. Leaving it out removes the stored Keycloak details and admin password. (see [below for nested schema](#nestedatt--schema_registry--keycloak))
- `listeners` (Map of String) Registry URLs applications use, by name. From Platform Manager 17.0.0, these are the `client_endpoints`: a new name becomes an endpoint with the registry's `api` for `ALL_APPS`, and an endpoint whose name is left out is removed. Conflicts with `client_endpoints`.
- `oauth_enabled` (Boolean) `APICURIO` only: whether applications may also sign in with OAuth. Basic authentication is always enabled. Like a Basic-Auth credential, an OAuth credential needs `keycloak` on this or another Cluster of the Instance. An application gets its OAuth credential in the Self-Service UI: `axual_application_credential` makes Basic-Auth credentials only. Default `false`.
- `password` (String, Sensitive) Password for `Basic-Auth`. Platform Manager never returns it, so it is not read back and cannot be imported. This field is Sensitive.
- `private_key` (String, Sensitive) PEM private key of `certificate`. Platform Manager never returns it. This field is Sensitive.
- `properties` (Map of String) Extra settings Platform Manager passes to the registry client.
- `truststore` (String) PEM CA certificates that sign the registry's certificate.
- `username` (String) Username for `Basic-Auth`. Requires `password`.
- `version` (String) The registry version, for example `3.0`.

<a id="nestedatt--schema_registry--client_endpoints"></a>
### Nested Schema for `schema_registry.client_endpoints`

Required:

- `api` (String) The API applications use at this endpoint: `APICURIO_V2`, `APICURIO_V3` or `CONFLUENT_V7`.
- `url` (String) The URL of the endpoint, at most 255 characters.
- `used_by` (String) Which applications use the endpoint: `MANAGED_APPS`, or `ALL_APPS` (managed and self-managed applications; only these endpoints are shown to users).


<a id="nestedatt--schema_registry--keycloak"></a>
### Nested Schema for `schema_registry.keycloak`

Required:

- `admin_client_id` (String) The client the admin user signs in with, usually `admin-cli`.
- `admin_password` (String, Sensitive) Keycloak admin password. Platform Manager never returns it, and signs in to Keycloak with it on every save of the registry, so a wrong one fails the apply. This field is Sensitive.
- `admin_username` (String) Keycloak admin username.
- `apicurio_realm` (String) The realm of the Apicurio clients.
- `master_realm` (String) The realm the admin user signs in to, usually `master`.
- `url` (String) Keycloak base URL.

## Example Usage

```hcl
resource "axual_cluster" "production" {
  name          = "production-ams"
  provider_type = "apache-kafka"
  broker = {
    bootstrap_servers = "kafka-0.example.com:9093"
    protocol          = "MTLS"
    certificate       = file("pm-client.crt")
    private_key       = file("pm-client.key")
    truststore        = file("kafka-ca.crt")
  }
  topic_pattern = "{tenant}-{instance}-{environment}-{topic}"
}

resource "axual_instance" "production" {
  name            = "Production"
  short_name      = "prod"
  clusters        = [axual_cluster.production.id]
  ca_certificates = [file("application-ca.crt")]
}

resource "axual_instance_cluster" "production" {
  instance = axual_instance.production.id
  cluster  = axual_cluster.production.id
  schema_registry = {
    type           = "APICURIO"
    url            = "https://apicurio.example.com/apis/registry/v3" # Platform Manager 15.0.x and 16.0.x: an Apicurio 2.x registry, .../apis/registry/v2
    authentication = "Basic-Auth"
    username       = "platform-manager"
    password       = "YOUR_PASSWORD" # replace with the registry password
  }
}
```

Client endpoints for applications (Platform Manager 17.0.0 or later): an in-cluster URL for managed applications, and a public URL that self-managed applications use too:

```hcl
resource "axual_instance_cluster" "production" {
  instance = axual_instance.production.id
  cluster  = axual_cluster.production.id
  schema_registry = {
    type           = "APICURIO"
    url            = "http://apicurio.kafka.svc:8080/apis/registry/v3"
    api            = "APICURIO_V3"
    authentication = "Basic-Auth"
    username       = "platform-manager"
    password       = "YOUR_PASSWORD" # replace with the registry password
    client_endpoints = {
      in-cluster = { api = "APICURIO_V3", url = "http://apicurio.kafka.svc:8080/apis/registry/v3", used_by = "MANAGED_APPS" }
      public     = { api = "APICURIO_V3", url = "https://apicurio.example.com/apis/registry/v3", used_by = "ALL_APPS" }
    }
  }
}
```

A Confluent registry without authentication:

```hcl
resource "axual_instance_cluster" "test" {
  instance = axual_instance.test.id
  cluster  = axual_cluster.test.id
  schema_registry = {
    type           = "CONFLUENT"
    url            = "https://schema-registry.example.com"
    authentication = "No-Auth"
  }
}
```

## Import

Import is supported using the Instance ID and the Cluster ID, separated by `/`:

```shell
terraform import axual_instance_cluster.<LOCAL NAME> <INSTANCE UID>/<CLUSTER UID>
terraform import axual_instance_cluster.production 9b4b1e3a1e2f4b5f8a1c0d2e3f4a5b6c/3f2a1b0c9d8e7f6a5b4c3d2e1f0a9b8c
```

After importing, add `password`, `private_key` and `keycloak.admin_password`, where they apply, to your configuration: Platform Manager does not return them, so import sets `keycloak.admin_password = ""`, and the next apply sends what the configuration has. Platform Manager signs in to Keycloak with the admin password on every save, so a wrong or empty one fails the apply and the stored one stays. It does not check `password` or `private_key` against the registry, so a wrong one is stored.
