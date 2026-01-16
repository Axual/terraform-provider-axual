# Changelog

All notable changes to this project will be documented in this file.

## master (https://github.com/Axual/terraform-provider-axual/tree/master) - 2026-01-15
### Added
* Import support for `axual_application_access_grant` resource
* Import support for `axual_application_access_grant_approval` resource
* Import support for `axual_application_access_grant_rejection` resource
* Auto-revoke on delete: Deleting an approved `axual_application_access_grant` now automatically revokes it first
* Grant Delete now handles terminal states (Revoked, Rejected, Cancelled) gracefully - just removes from state

### Changed
* Updated "Managing application access to topics" guide with import instructions and auto-revoke documentation
* Updated resource documentation for all three grant-related resources

### Fixed
* Grant Delete no longer fails with "Please Revoke first" error - approved grants are auto-revoked
* Approval and Rejection Read functions now properly handle NotFoundError and status changes


## [2.8.2](https://github.com/Axual/terraform-provider-axual/tree/master) - 2026-01-06
* Fix Application Access Grant failing to update when status is "Approved"
* Fix Application Deployment state not being saved when START operation times out
* Add retry logic for deployment START operation to handle transient failures
* Increase HTTP client timeout from 10s to 30s
* Rewrite "Managing application access to topics" guide

## [2.8.1](https://github.com/Axual/terraform-provider-axual/tree/master) - 2026-01-02
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
