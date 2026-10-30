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
The Grant request will remain in the Pending state until approved or rejected by the Topic team.

- The Topic team reviews the request from the Application team and records its decision in an `axual_application_access_grant_decision`: `decision = "APPROVED"` approves, `decision = "DENIED"` with a `reason` rejects. The decision finds the grant by application name or UID, topic, environment and access type, so no data source is needed. The two repositories can apply in either order.

- If the Topic team later decides to revoke the Application team's access, it changes the decision to `decision = "DENIED"` with a `reason`. Removing the decision block revokes as well (set `revoke_on_destroy = false` to only stop managing it).

- The Application team sees the answer in its own repository: the example's `logs_access` output shows `status` and `decision_reason` (the plan shows a new status under "Changes to Outputs"), and its optional `check` block warns until the grant is Approved. When the grant is Rejected or Revoked, every plan warns. To ask again after a `DENIED`, the Topic team first changes the decision to `"APPROVED"`, then the Application team changes `request_again`. The text of `request_again` isn't sent to the platform, so the reason for a request goes in the PR to the topic repository.

### 5. Topic team applies again
- In the order above, the Topic team's first apply finds no request yet: the plan warns that the decision waits for a request. Apply the topic repository again after the Application team's apply (in CI: the topic pipeline's next run). The plan shows `current_status: "Pending" -> "Approved"`, and the apply approves the request. After that, the Application team's `status` is `Approved`, with `decided_by` and `decided_at`.
- When the request already exists when the Topic team applies a decision (for example a decision added to a topic that exists already), that apply approves it at once.

### Alternative flow
- Instead of the Application Team creating the grant, it is possible for the Topic Team to create the Grant as well. In that case, the Topic Team creates both the grant and the decision in its repository, and the decision points at the grant with `grant_id = axual_application_access_grant.x.id`. This replaces the main flow for that grant: remove the main-flow decision for it, and the Application team doesn't create the grant. If the instance disables self-approval, the topic team made the request itself, so the platform doesn't offer it the approval: the apply that creates the decision fails until another topic owner approves the request in the UI, then apply again. Please see the configuration under the comment `ALTERNATIVE FLOW SETUP` in [Topic Team's Terraform Resources](https://github.com/Axual/terraform-provider-axual/blob/master/examples/3-team-guide/topic-team/main.tf)

### Revoking access

- The Application Team revokes its own access by deleting its `axual_application_access_grant`.
- This needs a member of the group that owns the application, or a user with the `APPLICATION_ADMIN`
  role. If the tenant lets only resource managers update and deploy, the member must also be a
  resource manager of that group.
- The Topic Team revokes by setting `decision = "DENIED"` with a `reason`.
- After a revoke, the Topic Team's decision stays in place; nothing fails.
- To get access again, the Application Team adds its grant back, or changes `request_again`; the same
  grant reopens. Under `decision = "APPROVED"` the Topic Team's next apply approves it. After
  `decision = "DENIED"`, the Topic Team changes the decision to `"APPROVED"` first.
- A revoke in the UI lasts only until the next request while the decision is `APPROVED`; the Topic
  Team's plans warn until then. To revoke so that it lasts, set `decision = "DENIED"`. See
  [Managing application access to topics](manage-application-access-to-topics).

### Removing the setup

The order matters:

- **Main flow: application, then topic, then admin.** Destroying the application repository first deletes its grant, which revokes it; then the topic repository is destroyed cleanly. The topic repository can go first too, because removing a decision revokes its grant (`revoke_on_destroy` defaults to `true`), but then the application repository's `data "axual_topic"` fails and blocks its own destroy: remove that data source first.
- **Alternative flow: topic, then application, then admin.** The Topic team owns the grant there. Destroying the application repository first fails with "Cannot delete the last principal with active grants".
- **Admin last:** both teams use its environments and groups.
