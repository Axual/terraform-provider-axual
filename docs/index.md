# Axual Provider

Axual Provider allows using Axual's Self-Service for Apache Kafka functionality within Terraform configurations.

## Axual Self Service
- Self Service allows fine-grained access control over your applications and topics, who is accessing your topics and for what purpose.
- Self Service displays valuable metadata about topics and the applications interacting with them, such as:
	- The format of the data that’s present on the topic
	- How long until the data is removed from the topic
	- Which applications are the producers and the consumers of this data
- Self Service provides control of your topic properties for individual environments and get an overview of the streaming landscape inside your organization.
	- For details, please refer to Axual Self-Service reference documentation: https://docs.axual.io/axual/2023.2/self-service/index.html

## Features

- User/group management
- Topic and application management
- Security
	- To secure which applications are authorised to access topics, we support
		- SSL (MUTUAL TLS) as a Certificate(PEM)
		- SASL (OAUTHBEARER) as a Custom Principal that specifies the ID referenced in URI and tokens. For example, 'my-client'
- Environment management
- Request, Approval, Revocation, Rejection and Cancellation of Access Requests
## Limitations
- Currently, there is a bug that deleting a resource that is managed by Terraform from UI results in Terraform not being able to recreate the resource again according to .tf configuration file. We do not recommend currently deleting resources managed by Terraform from UI. This bug has been reported to development team and is under investigation.
- Public environments cannot be deleted, private environments can be deleted. This feature will be implemented in the future.
- When deleting all resources at once, application.tf needs to have a dependency to make sure topic and topic_config get deleted first. This bug has been reported to development team and is under investigation.

# Getting started
## Required User Roles
- The Terraform User who is logged in(Default username kubernetes@axual.com), needs to have both of the following user roles:
  - **APPLICATION_ADMIN** - for creating application principal resource(axual_application_principal) and for create access request()
  - **STREAM_ADMIN** - for revoking access request
- Alternatively, they can be the owner of both the application and the topic, which entails being a user in the same group as the owner group of the application and topic.
## Example Usage

First, make sure to define and configure the provider:

```terraform
terraform {
  required_providers {
    axual = {
      source = "Axual/axual"
      version = "1.1.0"
    }
  }
}

# PROVIDER CONFIGURATION
#
# Below example configuration is for when you have deployed Axual Platform locally. Contact your administrator if you
# need the details for your organization's installation.
#
provider "axual" {
  apiurl   = "https://platform.local/api"
  realm    = "axual"
  username = "kubernetes@axual.com" #- or set using env property export AXUAL_AUTH_USERNAME=
  password = "PLEASE_CHANGE_PASSWORD" #- or set using env property export AXUAL_AUTH_PASSWORD=
  clientid = "self-service"
  authurl = "https://platform.local/auth/realms/axual/protocol/openid-connect/token"
  scopes = ["openid", "profile", "email"]
}
```

Next, take a look at the *full* example which shows you the capabilities of the TerraForm Provider for Axual:

```terraform
#
# TERRAFORM PROVIDER EXAMPLE
#
# This TerraForm file shows the capabilities of the TerraForm provider for Axual
# It is tested on the latest version of Axual Platform (2023.2)
#
# NOTE: execute ./init.sh to import the `tenant_admin` and `tenant_admin_group` resources which are created as part of a fresh installation
#
#


#
# GROUPS and USERS
# ----------------
# GROUPS own entities like TOPIC, APPLICATION  and ENVIRONMENT. USERS are members of a GROUP
# Below, three users are declared with certain roles in the system.

#
# Users "john", "jane" and "dwight" have roles which new users typically have
#
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/user
#

resource "axual_user" "john" {
  first_name    = "John"
  last_name     = "Doe"
  email_address = "john.doe@example.com"
  phone_number = "+37253412551"
  roles         = [
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" }
  ]
}

resource "axual_user" "jane" {
  first_name    = "Jane"
  last_name     = "Walker"
  email_address = "jane.walker@example.com"
  phone_number = "+37253412553"
  roles         = [
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" }
  ]
}

resource "axual_user" "dwight" {
  first_name    = "Dwight"
  last_name     = "Corner"
  email_address = "dwight.corner@example.com"
  phone_number = "+37253412553"
  roles         = [
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" }
  ]
}

#
# User "green" has elevated permissions, he has the TENANT_ADMIN role
#

resource "axual_user" "green" {
  first_name    = "Green"
  last_name     = "Stones"
  email_address = "green.stones@example.com"
  phone_number = "+37253412552"
  roles         = [
    { name = "TENANT_ADMIN" },
  ]
}

#
# WARNING: built-in user, execute `init.sh` if you have not done that already
#
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/user#import
#

resource "axual_user" "tenant_admin" {
  first_name    = "Tenant"
  last_name     = "Admin"
  email_address = "kubernetes@axual.com"
  roles         = [
    { name = "TENANT_ADMIN" },
    { name = "APPLICATION_AUTHOR" },
    { name = "ENVIRONMENT_AUTHOR" },
    { name = "STREAM_AUTHOR" },
    { name = "STREAM_ADMIN" },
    { name = "APPLICATION_ADMIN" }
  ]
}

#
# Users "john" and "jane" are members of group "Team Awesome", "dwight" is a member of "Team Bonanza" while "green" is a member of "Team Support"
#
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/group
#

resource "axual_group" "team-awesome" {
  name          = "Team Awesome"
  phone_number="+37253412559"
  email_address="team.awesome@example.com"
  members       = [
    	axual_user.jane.id,
    	axual_user.john.id  ]
}

resource "axual_group" "team-bonanza" {
  name		= "Team Bonanza"
  phone_number	= "+37253412558"
  email_address	= "team.bonanza@example.com"
  members     	= [
	axual_user.dwight.id
  ]
}

resource "axual_group" "team-support" {
  name          = "Team Support"
  phone_number  = "+37253412550"
  email_address = "team.support@example.com"
  members       = [
        axual_user.green.id
  ]
}

#
# WARNING: built-in group, execute `init.sh` if you have not done that already
#

resource "axual_group" "tenant_admin_group" {
 name          = "Tenant Admin Group"
 members       = [
   axual_user.tenant_admin.id,
 ]
}

#
# Below, environments are defined which are in use for the tenant.
# PRIVATE environments can only be used by members of the owning group.
# PUBLIC environments can be used by the entire organization
#

#
# Team Awesome has its own environment, called "team-awesome", which they use as a sandbox
# ENVIRONMENTs "development", "staging" and "production" are environments used by all teams and therefore declared PUBLIC
#
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/environment
#

resource "axual_environment" "team-awesome" {
  name = "team-awesome"
  short_name = "awesome"
  description = "This is the sandbox environment of Team Awesome"
  color = "#4686f0"
  visibility = "Private"
  authorization_issuer = "Auto"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = axual_group.team-awesome.id
}

resource "axual_environment" "development" {
  name = "development"
  short_name = "dev"
  description = "This is the development environment"
  color = "#19b9be"
  visibility = "Public"
  authorization_issuer = "Auto"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = axual_group.tenant_admin_group.id
}

resource "axual_environment" "staging" {
  name = "staging"
  short_name = "staging"
  description = "Staging contains close to real world data"
  color = "#3b0d98"
  visibility = "Public"
  authorization_issuer = "Stream owner"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = axual_group.tenant_admin_group.id
}

resource "axual_environment" "production" {
  name = "production"
  short_name = "production"
  description = "Real world production environment"
  color = "#3b0d98"
  visibility = "Public"
  authorization_issuer = "Stream owner"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = axual_group.tenant_admin_group.id
  properties = {
    "segment.ms"="60002"
  }
}

#
# An APPLICATION is anything that produces or consumes data from a topic.
# In Axual Platform we distinguish CUSTOM and CONNECTOR type applications.
# Note: currently, only CUSTOM applications are supported through the TF Provider for Axual
#
# In the example below, applications "dev_dashboard" and "log_scraper" are declared
#
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/application
#

resource "axual_application" "dev_dashboard" {
  name    = "DeveloperDashboard"
  application_type     = "Custom"
  short_name = "dev_dash"
  application_id = "io.axual.devs.dashboard"
  owners = axual_group.team-awesome.id
  type = "Java"
  visibility = "Public"
  description = "Dashboard with crucial information for Developers"
#  depends_on = [axual_topic_config.logs_in_production, axual_topic.support] # This is a workaround when all resources get deleted at once, to delete topic_config and topic before application. Mentioned in index.md
}

resource "axual_application" "log_scraper" {
  name    = "LogScraper"
  application_type     = "Custom"
  short_name = "log_scraper"
  application_id = "io.axual.gitops.scraper"
  owners = axual_group.team-awesome.id
  type = "Java"
  visibility = "Public"
  description = "Axual's Test Application for finding all Logs for developers"
#  depends_on = [axual_topic_config.logs_in_dev, axual_topic.logs] # This is a workaround when all resources get deleted at once, to delete topic_config and topic before application. Mentioned in index.md
}

#
# Every application has an APPLICATION_PRINCIPAL which defines how the application authenticates
# to Axual Platform. Every APPLICATION_PRINCIPAL is defined per ENVIRONMENT the APPLICATION is used in
#
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/application_principal
#

resource "axual_application_principal" "dev_dashboard_in_dev_principal" {
  environment = axual_environment.development.id
  application = axual_application.dev_dashboard.id
  principal = file("certs/certificate.pem")
}

resource "axual_application_principal" "dev_dashboard_in_staging_principal" {
  environment = axual_environment.staging.id
  application = axual_application.dev_dashboard.id
  principal = file("certs/certificate.pem")
}

resource "axual_application_principal" "log_scraper_in_dev_principal" {
  environment = axual_environment.development.id
  application = axual_application.log_scraper.id
  principal = file("certs/certificate.pem")
}

resource "axual_application_principal" "log_scraper_in_staging_principal" {
  environment = axual_environment.staging.id
  application = axual_application.log_scraper.id
  principal = file("certs/certificate.pem")
}

resource "axual_application_principal" "dev_dashboard_in_production_principal" {
  environment = axual_environment.production.id
  application = axual_application.dev_dashboard.id
  principal = file("certs/certificate.pem")
}

resource "axual_application_principal" "log_scraper_in_production_principal" {
  environment = axual_environment.production.id
  application = axual_application.log_scraper.id
  principal = file("certs/certificate.pem")
}

#
# A Schema is an AVRO definition formatted in JSON.
# In Axual Platform Schemas are used by Topics of data type AVRO (avsc file).
# Note: An attempt at uploading a duplicate schema is rejected with an error message containing the duplicated version
#
# In the example below, schema_version "axual_gitops_test_schema_version1", "axual_gitops_test_schema_version2" and "axual_gitops_test_schema_version3" are declared referencing their respective schema version
#
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/schema_version
#

resource "axual_schema_version" "axual_gitops_test_schema_version1" {
  body = file("avro-schemas/gitops_test_v1.avsc")
  version = "1.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "axual_gitops_test_schema_version2" {
  body = file("avro-schemas/gitops_test_v2.avsc")
  version = "2.0.0"
  description = "Gitops test schema version"
}

resource "axual_schema_version" "axual_gitops_test_schema_version3" {
  body = file("avro-schemas/gitops_test_v3.avsc")
  version = "3.0.0"
  description = "Gitops test schema version"
}

#
# While TOPIC mostly holds metadata, such as the owner and data type,
# the TOPIC_CONFIG configures a TOPIC in an ENVIRONMENT
#
# Below, some TOPICs are declared and configured in different environments and owned by different GROUPs
#
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/topic
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/topic_config

resource "axual_topic" "logs" {
  name = "logs"
  key_type = "String"
  value_type = "String"
  owners = axual_group.team-bonanza.id
  retention_policy = "delete"
  properties = { }
  description = "Logs from all applications"
}

resource "axual_topic_config" "logs_in_dev" {
  partitions = 1
  retention_time = 864000
  topic = axual_topic.logs.id
  environment = axual_environment.development.id
  properties = {"segment.ms"="600012", "retention.bytes"="1"}
}

resource "axual_topic_config" "logs_in_staging" {
  partitions = 1
  retention_time = 1001000
  topic = axual_topic.logs.id
  environment = axual_environment.staging.id
  properties = {"segment.ms"="60002", "retention.bytes"="100"}
}

resource "axual_topic_config" "logs_in_production" {
  partitions = 2
  retention_time = 86400000
  topic = axual_topic.logs.id
  environment = axual_environment.production.id
  properties = {"segment.ms"="600000", "retention.bytes"="10089"}
}

resource "axual_topic" "logs_with_avro" {
  name = "logswithavro"
  key_type = "AVRO"
  key_schema = axual_schema_version.axual_gitops_test_schema_version1.schema_id
  value_type = "AVRO"
  value_schema = axual_schema_version.axual_gitops_test_schema_version2.schema_id
  owners = axual_group.team-bonanza.id
  retention_policy = "delete"
  properties = { }
  description = "Logs from all applications with Avro schema"
}

resource "axual_topic_config" "logs_avro_in_dev" {
  partitions = 1
  retention_time = 864000
  topic = axual_topic.logs_with_avro.id
  environment = axual_environment.development.id
  key_schema_version = axual_schema_version.axual_gitops_test_schema_version2.id
  value_schema_version = axual_schema_version.axual_gitops_test_schema_version1.id
  properties = {"segment.ms"="600012", "retention.bytes"="1"}
}

resource "axual_topic_config" "logs_avro_in_staging" {
  partitions = 1
  retention_time = 1001000
  topic = axual_topic.logs_with_avro.id
  environment = axual_environment.staging.id
  key_schema_version = axual_schema_version.axual_gitops_test_schema_version2.id
  value_schema_version = axual_schema_version.axual_gitops_test_schema_version3.id
  properties = {"segment.ms"="60002", "retention.bytes"="100"}
}

resource "axual_topic_config" "logs_avro_in_production" {
  partitions = 2
  retention_time = 86400000
  topic = axual_topic.logs_with_avro.id
  environment = axual_environment.production.id
  key_schema_version = axual_schema_version.axual_gitops_test_schema_version3.id
  value_schema_version = axual_schema_version.axual_gitops_test_schema_version3.id
  properties = {"segment.ms"="600000", "retention.bytes"="10089"}
}

resource "axual_topic" "support" {
  name = "support"
  key_type = "String"
  value_type = "String"
  owners = axual_group.team-support.id
  retention_policy = "delete"
  properties = { }
  description = "Support tickets from Help Desk"
}

resource "axual_topic_config" "support_in_staging" {
  partitions = 1
  retention_time = 1001
  topic = axual_topic.support.id
  environment = axual_environment.staging.id
  properties = {"segment.ms"="60002", "retention.bytes"="1234"}
}

resource "axual_topic_config" "support_in_production" {
  partitions = 4
  retention_time = 10000000
  topic = axual_topic.support.id
  environment = axual_environment.production.id
  properties = {"segment.ms"="600000", "retention.bytes"="10089"}
}

#
# An APPLICATION_ACCESS_GRANT represents a connection between an APPLICATION and a TOPIC
# Its ACCESS_TYPE is either PRODUCER or CONSUMER, depending on the use case
# The grant refers to the principal, because the principal is used by the application to
# identify itself to the platform
#
# Below, APPLICATION_ACCESS_GRANTs are created for the APPLICATIONs defined above,
# with different ACCESS_TYPEs
#
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/application_access_grant
#

resource "axual_application_access_grant" "dash_consume_from_logs_in_dev" {
  application = axual_application.dev_dashboard.id
  topic = axual_topic.logs.id
  environment = axual_environment.development.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.dev_dashboard_in_dev_principal ]
}

resource "axual_application_access_grant" "log_scraper_consume_from_support_in_dev" {
  application = axual_application.log_scraper.id
  topic = axual_topic.support.id
  environment = axual_environment.development.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.log_scraper_in_dev_principal ]
}

resource "axual_application_access_grant" "dash_consume_from_logs_in_staging" {
  application = axual_application.dev_dashboard.id
  topic = axual_topic.logs.id
  environment = axual_environment.staging.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.dev_dashboard_in_staging_principal ]
}

resource "axual_application_access_grant" "dash_consume_from_support_in_staging" {
  application = axual_application.dev_dashboard.id
  topic = axual_topic.support.id
  environment = axual_environment.staging.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.dev_dashboard_in_staging_principal ]
}

resource "axual_application_access_grant" "scraper_produce_to_logs_in_staging" {
  application = axual_application.log_scraper.id
  topic = axual_topic.logs.id
  environment = axual_environment.staging.id
  access_type = "PRODUCER"
  depends_on = [ axual_application_principal.log_scraper_in_staging_principal ]
}

resource "axual_application_access_grant" "dash_consume_from_logs_in_production" {
  application = axual_application.dev_dashboard.id
  topic = axual_topic.logs.id
  environment = axual_environment.production.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.dev_dashboard_in_production_principal ]
}

resource "axual_application_access_grant" "dash_consume_from_support_in_production" {
  application = axual_application.dev_dashboard.id
  topic = axual_topic.support.id
  environment = axual_environment.production.id
  access_type = "CONSUMER"
  depends_on = [ axual_application_principal.dev_dashboard_in_production_principal ]
}

resource "axual_application_access_grant" "scraper_produce_to_logs_in_production" {
  application = axual_application.log_scraper.id
  topic = axual_topic.logs.id
  environment = axual_environment.production.id
  access_type = "PRODUCER"
  depends_on = [ axual_application_principal.log_scraper_in_production_principal ]
}

#
# An APPLICATION_ACCESS_GRANT can be approved by creating an APPLICATION_ACCESS_GRANT_APPROVAL with
# a reference to the APPLICATION_ACCESS_GRANT which needs to be approved, as can be seen
# in the example below
#
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/application_access_grant_approval
#

resource "axual_application_access_grant_approval" "dash_consume_logs_dev" {
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_dev.id
}

resource "axual_application_access_grant_approval" "dash_consume_logs_staging" {
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_staging.id
}

resource "axual_application_access_grant_approval" "dash_consume_support_production"{
  application_access_grant = axual_application_access_grant.dash_consume_from_support_in_production.id
}

resource "axual_application_access_grant_approval" "log_consume_support_dev"{
  application_access_grant = axual_application_access_grant.log_scraper_consume_from_support_in_dev.id
}

resource "axual_application_access_grant_approval" "dash_consume_logs_production"{
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_production.id
}

resource "axual_application_access_grant_approval" "scraper_produce_logs_production"{
  application_access_grant = axual_application_access_grant.scraper_produce_to_logs_in_production.id
}

#
# To reject an APPLICATION_ACCESS_GRANT, create an APPLICATION_ACCESS_GRANT_REJECTION
#
# Reference: https://registry.terraform.io/providers/Axual/axual/latest/docs/resources/application_access_grant_rejection
#

resource "axual_application_access_grant_rejection" "scraper_produce_logs_staging_rejection" {
  application_access_grant = axual_application_access_grant.scraper_produce_to_logs_in_staging.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `apiurl` (String) URL that will be used by the client for all resource requests
- `authurl` (String) Token url
- `clientid` (String) Client ID to be used for oauth
- `password` (String, Sensitive) Password belonging to the user
- `realm` (String) Axual realm used for the requests
- `username` (String) Username for all requests. Will be used to acquire a token

### Optional
- `scopes` (List of String) OAuth authorization server scopes

## Guides

- Our guides are in the guides folder:
	- How to import user and group: [Importing user and group](guides/importing-user-and-groups.md)
	- Setting up Terraform with Axual Trial: [Axual Trial setup](guides/axual-trial-setup.md)
	- Managing application access to topics: [Axual Trial setup](guides/manage-application-access-to-topics.md)


## Compatibility
 - This terraform provider requires Management API 7.0.7

## Output
Please include output if you want to have detailed information, e.g. for debugging purposes or for data sources.
Example of an output for the environment resource.

```
output "staging_id" {
	value = axual_environment.staging.id
  }
  
  output "staging_name" {
	value = axual_environment.staging.name
  }
```