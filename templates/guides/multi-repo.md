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

- **Topic Team**: Manages `axual_topic`, `axual_topic_config`, `axual_schema_version`, and `axual_application_access_grant_decision` (the decision about each access request).
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

- The Topic team reviews the request from the Application team and records its decision in an `axual_application_access_grant_decision`: `decision = "APPROVED"` approves, `decision = "DENIED"` with a `reason` rejects. The decision finds the grant by application name or UID, topic, environment and access type, so no data source is needed. The two repositories can apply in either order.

- If the Topic team later decides to revoke the Application team's access, it changes the decision to `decision = "DENIED"` with a `reason`. Removing the decision block does **not** revoke (unless `revoke_on_destroy = true`).

- The Application team sees the answer in its own plan: `status`, `decision_reason`, `decided_at`, and a warning when the grant is Rejected or Revoked. To ask again, it changes `request_again` in a PR; the topic team approves that new request by changing `review` in a PR.

### Alternative flow
- Instead of the Application Team creating the grant, it is possible for the Topic Team to create the Grant as well. In that case, the Topic Team creates both the grant and the decision in its repository, and the decision refers to the grant with `grant = axual_application_access_grant.x.id`. If the instance disables self-approval, the plan warns that the platform doesn't offer the approval to the user who made the request. Please see the configuration under the comment `ALTERNATIVE FLOW SETUP` in  [Topic Team's Terraform Resources](https://github.com/Axual/terraform-provider-axual/blob/master/examples/3-team-guide/topic-team/main.tf)

### Revoking access

- The Application Team revokes its own access by deleting its `axual_application_access_grant`.
- This needs a member of the group that owns the application, or a user with the `APPLICATION_ADMIN`
  role. If the tenant lets only resource managers update and deploy, the member must also be a
  resource manager of that group.
- The Topic Team revokes by setting `decision = "DENIED"` with a `reason`.
- After a revoke, the Topic Team's decision stays in place; nothing fails.
- To get access again, the Application Team adds its grant back, or changes `request_again`; the same
  grant reopens. If the Application Team removed the grant itself, the Topic Team's next apply approves
  it again (with `decision = "APPROVED"`). If the Topic Team revoked it, the Topic Team approves the new
  request by changing `review` in a PR. When both teams use the same Terraform account, the provider
  can't tell who revoked, so a `review` change is always needed. See
  [Managing application access to topics](manage-application-access-to-topics).
