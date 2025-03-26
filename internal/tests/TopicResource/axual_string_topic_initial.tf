resource "axual_topic" "topic-test" {
  name             = "test-topic"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties = {
    propertyKey1 = "propertyValue1"
    propertyKey2 = "propertyValue2"
  }
  description = "Demo of deploying a topic via Terraform"
}