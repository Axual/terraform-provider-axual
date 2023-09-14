# axual_topic (Resource)

A topic represents a flow of information (messages), which is continuously updated. Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html

## Limitations
Axual Terraform Provider does not support AVRO key type and AVRO value type. AVRO key and value type will be supported in the future. To use AVRO key or value type please use the Self Service UI.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `key_type` (String) The key type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#key-type
- `name` (String) The name of the topic. This must be in the format string-string (Needs to contain exactly one dash). The topic name is usually discussed and finalized as part of the Intake session or a follow up.
- `owners` (String) The team owning this topic. Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#stream-owner
- `properties` (Map of String) Advanced (Kafka) properties for a topic in a given environment. Read more: https://docs.axual.io/axual/2023.2/self-service/advanced-features.html#configuring-topic-properties
- `retention_policy` (String) Determines what to do with messages after a certain period. Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#retention-policy
- `value_type` (String) The value type and reference to the schema (if applicable). Read more: https://docs.axual.io/axual/2023.2/self-service/topic-management.html#value-type

### Optional

- `description` (String) A text describing the purpose of the topic.

### Read-Only

- `id` (String) Topic unique identifier

## Example Usage

```hcl
resource "axual_topic" "logs" {
  name = "logs"
  key_type = "String"
  value_type = "String"
  owners = axual_group.team-bonanza.id
  retention_policy = "delete"
  properties = { }
  description = "Logs from all applications"
}
```

For a full example which shows the capabilities of the latest TerraForm provider, check https://github.com/Axual/terraform-provider-axual/tree/master/examples/axual.

## Import

Import is supported using the following syntax:

```shell
terraform import axual_topic.<LOCAL NAME> <TOPIC UID>
terraform import axual_topic.test_topic b21cf1d63a55436391463cee3f56e393
```