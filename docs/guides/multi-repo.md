---
page_title: "Declarative Multi-Repo Gitops Setup"
---
### Overview

This guide demonstrates how to use Axual Self Service with the Axual Terraform provider across three teams:

1. **Application Team**: Requests permissions to produce to or consume from a topic owned by the Topic Team.
2. **Topic Team**: Approves or rejects application access requests to their topics.
3. **Admin Team**: Manages users, groups and environments in Self-Service.

**Key Practices**:
- Each team manages its own Terraform state independently.
- Teams utilize dedicated Terraform users configured with the minimum required privileges.
- Teams reference resources from other teams by utilizing Terraform data sources.

### Outcomes

The teams will have the following responsibilities and roles:

- **Admin Team**: Manages `axual_user`, `axual_group`, and `axual_environment` resources.
  - Roles: `TENANT_ADMIN`, `ENVIRONMENT_ADMIN`

- **Application Team**: Manages `axual_application`, `axual_application_principal`, and `axual_application_access_grant` resources.
  - Roles: `APPLICATION_AUTHOR`

- **Topic Team**: Manages `axual_topic`, `axual_topic_config`, `axual_schema_version`, and approval/rejection of `axual_application_access_grant`.
  - Roles: `STREAM_AUTHOR`, `SCHEMA_AUTHOR`

## Scenario
The Application Team submits a request to produce to a topic. The Topic Team, owning the topic, will approve or reject this request.

### Setup
- Please do `terraform init` in each folder and replace username and password with that team's Terraform user credentials.

### 1. Admin Team
- The Admin team creates `Environments`, `Users` and `Groups` within its Terraform state using an Admin-level user account.
  - [Admin Team's Terraform Resources](https://github.com/Axual/terraform-provider-axual/blob/master/examples/3-team-guide/admin-team/main.tf)
- Please note that Users can be already registered using Keycloak or another authentication service. In that case, Admin team will define the configuration for these users that already exist and use `terraform import` to import them into `axual_user` resources. Please see more details here [User resource](../../docs/resources/user.md) 

### 2. Topic team
- Topic Team creates `Topic` and `Topic Configuration` in a specific environment.
  This configuration is maintained in a separate Terraform state and managed using a user account with Topic-specific roles.
  - [Topic Team's Terraform Resources](https://github.com/Axual/terraform-provider-axual/blob/master/examples/3-team-guide/topic-team/main.tf)

### 3. Application team
- Application Team creates `Application`, `Application Deployment`, `Application Principal` and `Application Access Grant`(to request to produce to or consume from a topic) in the same environment as `Topic Configuration`.
These resources are managed in a separate Terraform state using a user account with Application-specific roles.
  - [Application Team's Terraform Resources](https://github.com/Axual/terraform-provider-axual/blob/master/examples/3-team-guide/application-team/main.tf)

### 4. The Grant
- The Application team submits a request (an `Application Access Grant`) to either produce to or consume from a Topic
created by the Topic team in the same environment.
The Grant request will remain in a PENDING state until approved or rejected by the Topic team.

- The Topic team reviews the request from the Application team. To approve the request, the Topic team creates an Application Access Grant Approval resource. To reject the request, the Topic team creates an Application Access Grant Rejection resource. 

- If the Topic team later decides to revoke the Application team's access to a Topic, they simply remove the Application Access Grant Approval resource from their Terraform configuration. This action revokes the previously granted access.

### Alternative flow
- Instead of the Application Team creating the grant, it is possible for the Topic Team to create the Grant as well. In that case, the Topic Team would create both the Grant and Grant Approval resources in their repository. Please see the configuration under the comment `ALTERNATIVE FLOW SETUP` in  [Topic Team's Terraform Resources](https://github.com/Axual/terraform-provider-axual/blob/master/examples/3-team-guide/topic-team/main.tf)

### Limitations
- Currently, it is not possible for the Application Team to revoke the Grant Approval. The workaround is that the Application Team would need to ask the Topic Team to revoke the grant by deleting the Grant Approval.