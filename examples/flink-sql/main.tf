# Minimal Flink SQL example: read an Avro topic, write to a String topic.
#
# Replace every YOUR_* placeholder below with values for your own environment.
# This example uses its own directory, and therefore its own terraform.tfstate,
# so it never touches resources managed by examples/axual/*.tf.

variable "instance_short_name" {
  description = "Short name of the existing Axual instance"
  type        = string
  default     = "YOUR_INSTANCE_SHORT_NAME"
}

variable "group_name" {
  description = "Name of the existing group that will own the created resources"
  type        = string
  default     = "YOUR_GROUP_NAME"
}

variable "instance_cluster_id" {
  description = "Uid of the existing Instance-Cluster to register the Flink Cluster on"
  type        = string
  default     = "YOUR_CLUSTER_ID"
}

variable "flink_url" {
  description = "Base URL of the Ververica Platform, without a path - the platform appends /api/v2 itself"
  type        = string
  default     = "https://vvp.example.com"
}

variable "flink_api_token" {
  description = "Ververica namespace-scoped API token, best passed via TF_VAR_flink_api_token"
  type        = string
  sensitive   = true
  default     = "YOUR_VERVERICA_API_TOKEN"
}

# Pre-existing, shared infrastructure: referenced read-only, never created here.
data "axual_instance" "example" {
  short_name = var.instance_short_name
}

data "axual_group" "example" {
  name = var.group_name
}

resource "axual_environment" "example" {
  name                 = "tf-flink-sql-example"
  short_name           = "tfflinksql"
  description          = "Environment for the minimal Flink SQL example"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.example.id
  owners               = data.axual_group.example.id
}

resource "axual_flink_cluster" "example" {
  instance_id       = data.axual_instance.example.id
  cluster_id        = var.instance_cluster_id
  name              = "tf-flink-sql-example-cluster"
  description       = "Flink Cluster for the minimal Flink SQL example"
  url               = var.flink_url
  workspace         = "YOUR_VERVERICA_WORKSPACE"
  namespace         = "YOUR_VERVERICA_NAMESPACE"
  deployment_target = "YOUR_VERVERICA_DEPLOYMENT_TARGET"
  api_token         = var.flink_api_token

  # Required because the source topic below is AVRO: the platform injects this url into the
  # generated table DDL.
  schema_registries = [
    {
      type = "CONFLUENT"
      url  = "YOUR_SCHEMA_REGISTRY_URL"
    },
  ]
}

resource "axual_application" "example" {
  name             = "tf-flink-sql-example-app"
  application_type = "FLINK_SQL"
  short_name       = "tf_flink_sql_example_app"
  application_id   = "io.axual.terraform.flinksqlexample"
  owners           = data.axual_group.example.id
  visibility       = "Public"
  description      = "Minimal FLINK_SQL application"
}

# A FLINK_SQL application needs credentials in the environment before an access
# grant can be created for it.
resource "axual_application_credential" "example" {
  application = axual_application.example.id
  environment = axual_environment.example.id
  target      = "KAFKA"
}

# chomp() strips the trailing newline file() reads: the API trims it server-side,
# and schema versions are immutable, so leaving it in causes a permanent diff.
resource "axual_schema_version" "avro_in" {
  body        = chomp(file("avro-schemas/simple_avro_value.avsc"))
  version     = "1.0.0"
  description = "Simple Avro value schema for the source topic"
  type        = "AVRO"
}

resource "axual_topic" "avro_in" {
  name             = "tf-flink-avro-in"
  key_type         = "String"
  value_type       = "AVRO"
  value_schema     = axual_schema_version.avro_in.schema_id
  owners           = data.axual_group.example.id
  retention_policy = "delete"
  properties       = {}
  description      = "Source topic: String key, Avro value"
}

resource "axual_topic_config" "avro_in" {
  partitions           = 1
  retention_time       = 864000
  topic                = axual_topic.avro_in.id
  environment          = axual_environment.example.id
  properties           = {}
  value_schema_version = axual_schema_version.avro_in.id
}

resource "axual_topic" "string_out" {
  name             = "tf-flink-string-out"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.example.id
  retention_policy = "delete"
  properties       = {}
  description      = "Target topic: plain String key/value"
}

resource "axual_topic_config" "string_out" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.string_out.id
  environment    = axual_environment.example.id
  properties     = {}
}

resource "axual_application_access_grant" "consume_avro_in" {
  application = axual_application.example.id
  topic       = axual_topic.avro_in.id
  environment = axual_environment.example.id
  access_type = "CONSUMER"

  depends_on = [
    axual_application_credential.example,
    axual_topic_config.avro_in,
  ]
}

resource "axual_application_access_grant_approval" "consume_avro_in" {
  application_access_grant = axual_application_access_grant.consume_avro_in.id
}

resource "axual_application_access_grant" "produce_string_out" {
  application = axual_application.example.id
  topic       = axual_topic.string_out.id
  environment = axual_environment.example.id
  access_type = "PRODUCER"

  depends_on = [
    axual_application_credential.example,
    axual_topic_config.string_out,
  ]
}

resource "axual_application_access_grant_approval" "produce_string_out" {
  application_access_grant = axual_application_access_grant.produce_string_out.id
}

# generate_tables_sql = true auto-generates the CREATE TABLE DDL for the granted
# topics, so sql_script only carries the transformation. Table names are the topic
# names with dashes replaced by underscores.
resource "axual_application_deployment" "example" {
  environment         = axual_environment.example.id
  application         = axual_application.example.id
  target_id           = axual_flink_cluster.example.id
  generate_tables_sql = true
  deployment_size     = "S"

  sql_script = <<-SQL
    INSERT INTO tf_flink_string_out (`key`, `value`)
    SELECT `key`, payload FROM tf_flink_avro_in;
  SQL

  depends_on = [
    axual_application_access_grant_approval.consume_avro_in,
    axual_application_access_grant_approval.produce_string_out,
  ]
}
