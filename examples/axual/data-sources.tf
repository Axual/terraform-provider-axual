
data "axual_group" "tfds_group" {
    name = "terraform-application-owners"
}

resource "axual_environment" "tfds" {
  name = "tfds"
  short_name = "tfds"
  description = "This an environment for demo terraform data sources"
  color = "#4686f0"
  visibility = "Public"
  authorization_issuer = "Stream owner"
  instance = "1be6269156d14ab09f40ea5133316a33"
  owners = data.axual_group.tfds_group.id
}

resource "axual_application" "tfds_app" {
  name    = "TerraForm_application_owner"
  application_type  = "Custom"
  short_name = "tfds_app"
  application_id = "io.axual.tfds.app"
  owners = data.axual_group.tfds_group.id
  type = "Java"
  visibility = "Public"
  description = "application for demostrating terraform data sources"
}
resource "axual_application_principal" "tfds_app_in_tfds_principal" {
  environment = axual_environment.tfds.id
  application = axual_application.tfds_app.id
  principal = file("certs/certificate.pem")
}

data "axual_topic" "algorithms" {
    name = "algorithms"
}

resource "axual_application_access_grant" "tfds_app_produce_to_algorithms_in_tfds" {
  application = axual_application.tfds_app.id
  topic = data.axual_topic.algorithms.id
  environment = axual_environment.tfds.id
  access_type = "PRODUCER"
}

data "axual_schema_version" "Application" {
    id = "85acd22f4e3b483c9e4cbd405cc8098f"
}

output "Application" {
    value = data.axual_schema_version.Application
}

resource "axual_topic" "avro_topic_with_ds" {
  name = "avroTopicDS"
  key_type = "AVRO"
  key_schema = data.axual_schema_version.Application.id
  value_type = "AVRO"
  value_schema = data.axual_schema_version.Application.id
  owners = data.axual_group.tfds_group.id
  retention_policy = "delete"
  properties = { }
  description = "avro topic created using external data source"
}

# resource "axual_topic_config" "avro_topic_with_ds_in_tfds" {
#   partitions = 1
#   retention_time = 864000
#   topic = axual_topic.avro_topic_with_ds.id
#   environment = axual_environment.tfds.id
#   key_schema_version = data.axual_schema_version.Application.id
#   value_schema_version = data.axual_schema_version.Application.id
#   properties = {"segment.ms"="600012", "retention.bytes"="1"}
# }