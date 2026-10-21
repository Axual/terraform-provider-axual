resource "axual_instance_cluster" "tf_test_ic" {
  instance = axual_instance.tf_test_ic_instance.id
  cluster  = axual_cluster.tf_test_ic_cluster.id
  schema_registry = {
    type           = "APICURIO"
    url            = local.schema_registry_url
    authentication = "Basic-Auth"
    username       = local.schema_registry_username
    password       = local.schema_registry_password
    version        = "3.0"
  }
}

resource "axual_environment" "tf_test_ic_env" {
  name                 = "tf-test-ic-environment"
  short_name           = "tftesticenv"
  description          = "Environment on a Terraform-made Instance"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Auto"
  instance             = axual_instance.tf_test_ic_instance.id
  owners               = data.axual_group.test_group.id
  depends_on           = [axual_instance_cluster.tf_test_ic]
}

resource "axual_schema_version" "tf_test_ic_key" {
  body        = file("avro-schemas/tf_test_ic_key.avsc")
  version     = "1.0.0"
  description = "Key schema of the Instance-Cluster test"
}

resource "axual_schema_version" "tf_test_ic_value" {
  body        = file("avro-schemas/tf_test_ic_value.avsc")
  version     = "1.0.0"
  description = "Value schema of the Instance-Cluster test"
}

resource "axual_topic" "tf_test_ic_topic" {
  name             = "tf-test-ic-topic"
  key_type         = "AVRO"
  key_schema       = axual_schema_version.tf_test_ic_key.schema_id
  value_type       = "AVRO"
  value_schema     = axual_schema_version.tf_test_ic_value.schema_id
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "AVRO topic on a Terraform-made Instance"
}

resource "axual_topic_config" "tf_test_ic_topic_config" {
  partitions           = 1
  retention_time       = 864000
  topic                = axual_topic.tf_test_ic_topic.id
  environment          = axual_environment.tf_test_ic_env.id
  key_schema_version   = axual_schema_version.tf_test_ic_key.id
  value_schema_version = axual_schema_version.tf_test_ic_value.id
  properties           = {}
}
