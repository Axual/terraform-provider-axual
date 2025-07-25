# axual_application_access_grant_approval (Resource)

Application Access Grant Approval: Approve access to a topic

## Note
- Approval cannot be edited
- Revoke grant by destroying application_access_grant_approval
- Read more: https://docs.axual.io/axual/2025.1/self-service/topic-authorizations.html#contents

## Usage
- To approve a grant create an application_access_grant_approval with the grant id
- To revoke a grant, delete the application_access_grant_approval
- To revoke an auto approved grant
  - Create an application_access_grant_approval
  - Then delete that application_access_grant_approval

## Required Roles
- STREAM_ADMIN or be part of the Team that owns the Topic

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `application_access_grant` (String) Application Access Grant Unique Identifier.

## Example Usage

```hcl
resource "axual_application_access_grant_approval" "dash_consume_logs_dev" {
  application_access_grant = axual_application_access_grant.dash_consume_from_logs_in_dev.id
}
```

For a full example which shows the capabilities of the latest TerraForm provider, check https://github.com/Axual/terraform-provider-axual/tree/master/examples/axual.

## Import

Import is not currently supported.