# axual_application_access_grant_rejection (Resource)

Application Access Grant Rejection: Reject a request to access stream

## Note
- Rejection cannot be edited
- Only Pending grants can be rejected
- Read more: https://docs.axual.io/axual/2023.1/self-service/topic-authorizations.html#contents

## Usage
- To reject/deny a grant create an application_access_grant_rejection with the application_access_grant id.

## Required Roles
- TOPIC ADMIN or be part of the Team that owns the Stream

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `application_access_grant` (String) Application Access Grant Unique Identifier.

### Optional

- `reason` (String) Reason for denying approval.

## Example Usage

```terraform
resource "axual_application_access_grant_rejection" "scraper_produce_logs_staging_rejection" {
  application_access_grant = axual_application_access_grant.scraper_produce_to_logs_in_staging.id
}
```

## Import

Import is not currently supported.