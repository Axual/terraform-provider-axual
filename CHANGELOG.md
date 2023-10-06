# Changelog

All notable changes to this project will be documented in this file.

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
