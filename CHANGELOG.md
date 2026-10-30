# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]
### Added
* `tags` on `axual_topic` and the `axual_topic` data source: the topic tags shown in the Self-Service UI. Leaving out `tags` keeps the tags the topic has; `tags = []` removes them ([#177](https://github.com/Axual/terraform-provider-axual/issues/177))
* Import by name instead of ID: `name:<name>` for `axual_topic`, `axual_environment`, `axual_application` and `axual_group`, `email:<email address>` for `axual_user`, and `name:<topic name>/<environment short name>` for `axual_topic_config`, for example `terraform import axual_topic.payments name:payments`. Works in `import` blocks too ([#114](https://github.com/Axual/terraform-provider-axual/issues/114))
* `terraform query` support (Terraform 1.14 or newer): list resources with filters, so `terraform query -generate-config-out` writes `resource` and `import` blocks for resources that already exist. Not for `axual_flink_cluster`. The access grant list leaves out revoked and cancelled grants unless its `status` filter asks for them
* Resource identity for every resource except `axual_flink_cluster`, so an `import` block can use `identity` instead of `id`
* Guides: `Quick start: export your resources with terraform query` and `Exporting existing resources with terraform query`, and an example in `examples/terraform-query`
* `axual_application_access_grant_decision`: the topic team's standing decision about one application's access. `APPROVED` approves every request from the application, `DENIED` (with a `reason`) rejects or revokes. It finds the grant itself by application name or UID, topic, environment and access type, so it works across repositories, for private applications (Platform Manager 15.0.3 or later), and before the request exists. A request made again is handled like a first request; `DENIED` is the close that lasts. In a single repository, `grant_id = axual_application_access_grant.x.id` replaces the four lookup attributes. `application_uid` keeps the application a decision by name matched first: when the name later belongs to another application (deleted and created again), the plan doesn't approve it and warns until `application` is set to the new UID. `closed_outside_terraform` reports a grant rejected or revoked outside Terraform (for example in the UI) while the decision is `APPROVED`, for use in a `check` block. The provider approves only when the platform offers approve to the signed-in user. The apply checks the grant again and changes nothing when it changed since the plan; when it already reached the planned status, the apply succeeds. Removing the block revokes an Approved grant, like removing an approval did; `revoke_on_destroy = false` only stops managing it. Existing `_approval` and `_rejection` blocks move with a `moved` block (Terraform 1.8 or later). With a list resource for `terraform query` (AXPD-12412)
* `request_again`, `decision_reason`, `decided_at` and `decided_by` on `axual_application_access_grant`, and the decision fields on its data source: the application team sees the answer and asks again by changing `request_again` (AXPD-12412)
* `axual_application_access_grants` data source: lists grants, for example the open requests on a topic (AXPD-12412)

### Changed
* Leaving out `roles` on an `axual_user` now keeps the roles the user has. Only `roles = []` removes them
* Changing `application`, `topic`, `environment` or `access_type` of an `axual_application_access_grant` fails in the plan instead of at apply (AXPD-12412)
* Upgrading from `axual_application_access_grant_approval`: a `moved` block keeps today's behaviour (removing the moved decision revokes), with no diff (AXPD-12412)
* Upgrading from `axual_application_access_grant_approval` on an instance with self-approval disabled: the decision approves only when the platform offers approve to the signed-in user, so it doesn't approve a request this user made itself, while the approval resource did. With `grant_id` the apply then fails, so a deployment with `depends_on` on the decision isn't created; with the lookup attributes (multi-repo) the plan warns (AXPD-12412)
* Approvals, rejects, revokes and cancels of access grants run at most 4 at a time, because Platform Manager can stop responding with about 10 at once. An apply that acts on many grants takes longer, without `-parallelism` (AXPD-12412)
* The access guides describe the platform correctly: Rejected, Revoked and Cancelled grants are closed, not final; a new request reopens the same grant (AXPD-12412)
* A plan that leaves out `configs`, `definition` or `sql_script` of an `axual_application_deployment` that has them stops with `Missing configs` (or `definition`, `sql_script`), unless the application or environment changes. `configs = {}` removes all configs (Platform Manager 16.0.0 or later). Before, the apply removed them and stopped the deployment

### Deprecated
* `axual_application_access_grant_approval` and `axual_application_access_grant_rejection`: use `axual_application_access_grant_decision` (AXPD-12412)

### Fixed
* A STOP that fails with a `5xx` (for example while the Connect worker rebalances) is retried, like START. It is not sent again once the deployment has stopped
* An `axual_application_deployment` deleted outside Terraform no longer breaks `plan` and `destroy` with `no application deployment found`; it is removed from the state and planned again
* `roles = []` on an `axual_user` removes all roles instead of failing with `Required request body is missing` (AXPD-9562)
* A Connector that is not on Kafka Connect, which Platform Manager now reports as `Undeployed` instead of `Undefined`, counts as stopped when an `axual_application_deployment` waits for it to stop before an update or delete
* A deployment status read retries a `5xx` for up to 2 minutes, for example right after the Kafka Connect workers restart. A `plan` can wait that long when a Connect cluster does not answer
* Destroy no longer fails when a resource is already gone, for example removed by another resource in the same destroy. A delete that gets `409 Could not commit changes` is retried
* Destroying the `axual_application_deployment_state` of a `Failed` connector now warns at once when Platform Manager cannot reset it, and says how to fix it
* An `axual_application_deployment` on a Kafka Connect cluster is imported with `autostart = false`. Before, the next apply failed with `Kafka Connect cluster needs autostart = false`
* An `axual_schema_version` `body` that differs from the stored body only in JSON whitespace or key order is not a change any more. Before, the apply failed with `API does not allow update of schema version`
* An `axual_topic_browse_permissions` whose topic configuration was deleted is removed from the state, and its destroy skips users and groups that were deleted meanwhile. Before, plan and destroy failed
* An `axual_application_credential` of a Schema Registry that uses OAuth, imported or exported with `terraform query`, now gets `target = "SCHEMA_REGISTRY"` instead of `"KAFKA"`, so destroying it deletes it. A configuration that sets `target = "KAFKA"` for such a credential must change it to `"SCHEMA_REGISTRY"`

## [3.2.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v3.2.0) - 2026-10-08
### Added
* Service account authentication with a client ID and secret: `client_id` and `client_secret`, or the `AXUAL_CLIENT_ID` and `AXUAL_CLIENT_SECRET` environment variables
* Federated service account authentication, which sends no secret at all: `oidc_token` and `oidc_token_file`, or the `AXUAL_OIDC_TOKEN` and `AXUAL_OIDC_TOKEN_FILE` environment variables. `oidc_token_file` is read again on every token renewal, so a rotated assertion is picked up automatically
* Guide: `Service account authentication`, covering both modes and migrating off username and password
* The provider now verifies its credentials while configuring, so bad credentials fail at the start of a plan instead of partway through an apply
* A member or manager of an `axual_group` may now be a service account as well as a user, on Platform Manager 16.0.0 or later. Adding a service account to a group requires Tenant Admin, and a manager must also be a member
* The `axual_connect_cluster` data source, to look up a registered Kafka Connect cluster by id or by name
* `target_id` on `axual_application_deployment` (Platform Manager 16.0.0 or later): the deployment target. Changing it updates a Connector or KSML deployment in place; a `FLINK_SQL` deployment is replaced
* `target_version` on `axual_application_deployment`, the plugin version to deploy for a Connector targeting a registered Kafka Connect cluster
* The `axual_application_deployment_state` resource: the desired state (`RUNNING` or `STOPPED`) of an Application Deployment. It starts, stops or restarts the deployment to match, shows a deployment stopped outside Terraform or failed as a change in the next plan, exposes the live status as `current_state`, and stops and resets the deployment on destroy. It can be imported by the deployment's Uid
* `autostart` on `axual_application_deployment`, default `true`. With `false` the deployment only stores its target and configs, like the Self-Service UI does, and is created before the principal, credential and access grant; an `axual_application_deployment_state` starts it. This is the order the Connector Application guide now recommends
* Connector deployments on a registered Kafka Connect cluster (Platform Manager 16.0.0 or later), on `MTLS` and `SASL_SCRAM` clusters: the deployment with `autostart = false`, then its `axual_application_principal` or `axual_application_credential`, then the grant, then an `axual_application_deployment_state`. With `autostart = true` such a deployment is refused with a message that explains this order. Legacy Axual Connect works with both `autostart` values
* Examples in `examples/connector-application` for AC+TLS with and without `autostart`, KC+TLS and KC+SASL
* Support for Flink Cluster as `axual_flink_cluster`, including its `schema_registries`
* Support for Axual-managed `FLINK_SQL` Application
* A size-only change of a `FLINK_SQL` Application Deployment is stored without redeploying the job.
  The new size takes effect the next time the job is deployed

### Changed
* A Connector deployment whose `target_id` names a registered Kafka Connect cluster needs `autostart = false`; otherwise it is refused with `Kafka Connect cluster needs autostart = false`. `terraform plan` reports this when the application already exists, otherwise `terraform apply` does, before anything is created
* `axual_group` sends each member and manager as a bare uid and lets Platform Manager resolve whether it names a user or a service account. Nothing changes for existing configurations; it removes the per-member API call that resolving them in the provider would have cost
* The authentication method is determined by which credentials are supplied. Supplying a service account credential takes precedence over `username` and `password`, which are ignored with a warning naming them - a leftover `AXUAL_AUTH_PASSWORD` will not break a pipeline that has moved to a service account
* The provider no longer disables TLS certificate verification process-wide. It now uses its own HTTP transport, leaving `http.DefaultTransport` untouched. Certificate verification behaviour for the provider's own requests is unchanged
* A Connector deployment is now updated in place instead of being replaced when its config changes
* An `axual_application_deployment` can be imported in any state, not only while it is running

### Deprecated
* `username` and `password` authentication. Use a service account instead
* `clientid`, replaced by `client_id`. Both are accepted; setting them to different values is an error
* `authmode` and `audience`. Neither has any effect

### Removed
* The `auth0` authentication mode. `authmode = "auth0"` now reports a clear error

### Fixed
* Rotating the certificate of an `axual_application_principal` no longer fails with "Provider produced inconsistent final plan" when the new certificate is only known at apply, for example when it comes from a module output or another resource ([#175](https://github.com/Axual/terraform-provider-axual/issues/175))
* Importing an `axual_application_credential` with an id that does not exist now reports "Application Credential Not Found" instead of a raw internal error message
* `terraform import` on an `axual_application_principal`, followed by `terraform apply` with the same certificate, no longer triggers a certificate rotation (create new, delete old)
* The error for a refused `axual_application_credential` delete now says "credential" instead of "principal"
* Create a Connector Application Deployment without `configs` (Platform Manager 16.0.0 or later)
* Wait for a deployment to stop before updating it
* Retry the delete of an `axual_application` that the API rejects with a transient commit conflict
  after an access grant was revoked moments earlier (AXPD-12049)
* Treat a failed Connector or KSML Application Deployment as stopped, so a destroy no longer waits
  out the full stop budget first
* Stop a deployment the status endpoint reports as running even when it advertises no actions
* Fix five `go vet` findings in `resource_schema_version.go`, `resource_topic_config.go` and
  `resource_user.go`, so `go test ./internal/provider/` runs with vet enabled

## [3.1.1](https://github.com/Axual/terraform-provider-axual/releases/tag/v3.1.1) - 2026-09-22
### Fixed
* `axual_schema_version` data source: fix looking up any schema version other than the latest
  one, which failed with "Schema version matching the name you requested was not found" even
  though the version existed

## [3.1.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v3.1.0) - 2026-06-30
### Added
* Allow rotating a Connector's `axual_application_principal`

### Fixed
* Retrieve an **Instance** using `findByName` or `findByShortName` endpoints instead of `findByAttributes`
* Read `axual_application_access_grant` using the `GET /application_access_grants/{uid}` endpoint instead of the `findByAttributes` search

## [3.0.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v3.0.0) - 2026-03-24
### Added
* Import support for `axual_application_principal`
* Import support for `axual_application_credential`
* Import support for `axual_topic_browse_permissions`
* User creation via Terraform is no longer supported.
* Removed `bootstrapServers` field references as the field has been removed from the `/clusters` API endpoints

### Changed
* Updated `axual_user` resource documentation to reflect the removal of user creation capability
* Removed unused API response fields across multiple data structures for improved maintainability

### Fixed
* Fix `axual_environment` import limitation

## [2.9.1](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.9.1) - 2026-01-18
### Fixed
* Fixed the compatibility matrix
* Fixed documentation about grants, clarifying that terraform plan does not automatically refresh the state.

## [2.9.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.9.0) - 2026-01-18
### Added
* Import support for `axual_application_access_grant` resource
* Import support for `axual_application_access_grant_approval` resource
* Import support for `axual_application_access_grant_rejection` resource
* Creating an `axual_application_access_grant_approval` in environment with automatic authorization issuer is not required.
* Grant Delete now handles terminal states (Revoked, Rejected, Cancelled) gracefully - just removes from state

### Changed
* Updated "Managing application access to topics" guide with import instructions and auto-revoke documentation
* Updated resource documentation for all three grant-related resources

### Fixed
* Grant Delete no longer fails with "Please Revoke first" error - approved grants are auto-revoked
* Approval and Rejection Read functions now properly handle NotFoundError and status changes

## [2.8.2](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.8.2) - 2026-01-06
* Fix Application Access Grant failing to update when status is "Approved"
* Fix Application Deployment state not being saved when START operation times out
* Add retry logic for deployment START operation to handle transient failures
* Increase HTTP client timeout from 10s to 30s
* Rewrite "Managing application access to topics" guide

## [2.8.1](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.8.1) - 2026-01-02
### Added
* Fix for issues: https://github.com/Axual/terraform-provider-axual/issues/133
* New guide for schemas
* Rewrote the guide for data sources
* Fix CHANGELOG.md
* Fix test against PM 12.0.0

## [2.8.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.8.0) - 2025-12-23
### Added
* Support for Axual-managed `KSML` Application

## [2.7.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.7.0) - 2025-11-04

### Added
* Support for **Protobuf** and **JSON Schema** types in `axual_schema_version` resource
* Support for **Kafka Streams**, **Pyton**, **KSML** and **Other** application types in `axual_application` resource

## [2.6.1](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.6.1) - 2025-10-08
* Refactor documentation to separate user and developer audiences

## [2.6.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.6.0) - 2025-10-02
* Add User data source
* Guide for JSON schema
* Added support for searching by shortName in Instance, Environment, Application Data Sources.
* Removed trial guide
* Added Support for importing Application Deployment
* Retrieve an application using `findByName` or `findByShortName` endpoints instead of `findByAttributes`
* Fixed `axual_topic_config` to allow in-place updates of `key_schema_version` and `value_schema_version` fields instead of forcing resource replacement
* Added boolean `force` attribute to `axual_topic_config` resource to force updates with incompatible schema version changes

## [2.5.6](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.5.6) - 2025-06-06
* Add User data source
* Guide for JSON schema
* Removed trial guide

## [2.5.5](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.5.5) - 2025-03-07
* Removed unused method for `/groups/{uid}/members/{uid}`
* Auth0 authentication support for Axual Trial Environment

## [2.5.4](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.5.4) - 2025-02-21
* Added clarity for `axual_application_credential` documentation.

## [2.5.3](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.5.3) - 2025-02-21
* Added support for "compact,delete" topic retention policy
* Support for Application Credential.
* Fix for Terraform Provider crash when creating Topic Configuration took too long.

## [2.5.2](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.5.2) - 2025-01-27
* Fixes for documentation

## [2.5.1](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.5.1) - 2025-01-27
* Added multi-repo gitops guide for 3 teams: admin, topic and application teams
* Fix dependency issues in Terraform provider

## [2.5.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.5.0) - 2025-01-05
* Added schema owner to the SchemaVersion resource and datasource
* Added settings field to environment resource
* Properties in Topic and Topic Config can now be omitted
* Properties and settings in Environments can now be omitted
* Terraform import support for: `axual_environment`, `axual_topic` and `axual_schema_version`

## [2.4.2](https://github.com/Axual/terraform-provider-axual/releases/tag/2.4.2) - 2024-12-12
* Added `Instance` data source.
* Added error handling for environment, group and topic data sources.
* Added AVRO schema support to Topic data source.
* Refactored running acceptance tests to be simpler.

## [2.4.1](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.4.1) - 2024-09-23
* Documentation improvements: Concise front page and improved Connector guide.
* Fixed a bug where user can't delete group's phone number and email address.
* Fixed a bug where user can't delete all members of a group.

## [2.4.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.4.0) - 2024-09-16
* Updated shortName in Environment resource to have min length 1
* Support for Viewers (Environment, Topic, Application) and Group managers
* Support for Topic Browse Permissions: Users and Groups can be added with new resource axual_topic_browse_permissions
* Added waiting and retry when creating/updating/deleting all resources resulting in a Kafka resource.

## [2.3.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.3.0) - 2024-04-25
* Update to terraform-plugin-framework v1.7.0

## [2.2.3](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.2.3) - 2024-04-15
* Documentation updates.

## [2.2.2](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.2.2) - 2024-04-15
* Documentation updates: application, application_deployment, application_principal. Connector application guide.

## [2.2.1](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.2.1) - 2024-04-15
* Documentation update: Connector application guide.

## [2.2.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.2.0) - 2024-04-10
* Feature: Support Connector Applications
* Bugfix: When trying to approve an application access request made from the UI in terraform, it fails with an error
* Bugfix: inconsistent datatype of Members property in Links and replace deprecated endpoint
---
## [2.1.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.1.0) - 2023-10-04
* Support Managing Environments
* Support Authorizing Application Access Grants in Terraform.
* Support Managing Avro Schemas
* Rename Stream to Topic
* Support data sources for group, environment topic, application, schema version and application access grant

## [2.0.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v2.0.0) - 2023-09-21
* Feature: Add Schema Version resource support
* Feature: Add AVRO schema type support for Topic and TopicConfig
* Fix: Replace Stream expression with Topic in the docs and code source

## [1.1.3](https://github.com/Axual/terraform-provider-axual/releases/tag/v1.1.3) - 2023-08-07
* Bug: changing resource state outside Terraform does not trigger re-creating of resources
* Bug: env variables are not being used for authentication

## [1.1.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v1.1.0) - 2023-07-12
* Environment resource
* Managing access with Terraform. Doing requests to get a grant and possibility for another team to approve, reject, cancel, revoke the grant.


## [1.0.3](https://github.com/Axual/terraform-provider-axual/releases/tag/v1.0.3) - 2023-04-19
* Documentation update describing that Stream has been renamed to Topic in Self-Service UI.

## [1.0.2](https://github.com/Axual/terraform-provider-axual/releases/tag/v1.0.2) - 2022-10-13
* Documentation update. Moved guides into separate folder for guides. Fixed the version to latest in documentation examples.

## [1.0.1](https://github.com/Axual/terraform-provider-axual/releases/tag/v1.0.1) - 2022-10-11
* Documentation update on using Terraform Provider with Axual Trial

## [1.0.0](https://github.com/Axual/terraform-provider-axual/releases/tag/v1.0.0) - 2022-10-08
* Support Managing Topics
* Support Managing Topic Configuration 
* Support Managing Applications
* Support Managing Application Authentication with SSL and Oauthbearer
* Support Managing Application Authorization through Application Access Grants for auto approved environemts
* Support Managing Users
* Support Managing Groups
* Support Managing Users
