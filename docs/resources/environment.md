# axual_environment (Resource)

Environments are used typically to support the application lifecycle, as it is moving from Development to Production.  In Self Service, they also allow you to test a feature in isolation, by making the environment Private. Read more: https://docs.axual.io/axual/2023.1/self-service/environment-management.html#managing-environments

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `authorization_issuer` (String) This indicates if any deployments on this environment should be AUTO approved or requires approval from Stream Owner. For private environments, only AUTO can be selected.
- `color` (String) The color used display the environment
- `instance` (String) The id of the instance where this environment should be deployed.
- `name` (String) A suitable name identifying this environment. This must be in the format string-string (Alphabetical characters, digits and the following characters are allowed: `- `,` _` ,` .`)
- `owners` (String) The id of the team owning this environment.
- `short_name` (String) A short name that will uniquely identify this environment. The short name should be between 3 and 20 characters. no special characters are allowed.
- `visibility` (String) Thi Private environments are only visible to the owning group (your team). They are not included in dashboard visualisations.

### Optional

- `description` (String) A text describing the purpose of the environment.
- `partitions` (Number) Defines the number of partitions configured for every stream of this tenant. This is an optional field. If not specified, default value is 12
- `properties` (Map of String) Environment-wide properties for all topics and applications.
- `retention_time` (Number) The time in milliseconds after which the messages can be deleted from all streams. This is an optional field. If not specified, default value is 7 days (604800000).

### Read-Only

- `id` (String) Environment unique identifier

## Example Usage

```terraform
resource "axual_environment" "dev" {
  name = "team-awesome"
  short_name = "awesome"
  description = "This is a test environment"
  color = "#19b9be"
  visibility = "Private"
  authorization_issuer = "Auto"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = "dd84b3ee8e4341fbb58704b18c10ec5c"
}

resource "axual_environment" "staging" {
  name = "staging"
  short_name = "staging"
  description = "Staging contains close to real world data"
  color = "#3b0d98"
  visibility = "Public"
  authorization_issuer = "Auto"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = "dd84b3ee8e4341fbb58704b18c10ec5c"
}

resource "axual_environment" "production" {
  name = "production"
  short_name = "production"
  description = "Real world production environment"
  color = "#3b0d98"
  visibility = "Public"
  authorization_issuer = "Stream owner"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = "dd84b3ee8e4341fbb58704b18c10ec5c"
  properties = {
    "segment.ms"="60002"
  }

}

output "staging_id" {
  value = axual_environment.staging.id
}
output "production_name" {
  value = axual_environment.staging.name
}
```

## Import

Import is supported using the following syntax:

```shell
terraform import axual_environment.<LOCAL NAME> <ENVIRONMENT UID>
terraform import axual_environment.test_env ab1cf1d63a55436391463cee3f56e393
```