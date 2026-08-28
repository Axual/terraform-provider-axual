# Full test bed for the AXPD-11717 Flink SQL implementation.
# Separate directory = separate terraform.tfstate, so this cannot touch
# resources managed by examples/axual/*.tf.
#
# Scope: everything below is created and destroyed by THIS Terraform config only.
# The "dev" instance and its "jupiter" Kafka cluster are pre-existing, shared
# infrastructure - referenced read-only via data sources, never created or
# destroyed here. (There is no axual_instance *resource* in this provider, only a
# data source - a brand-new instance cannot be managed by Terraform at all.)
#
# The Flink Cluster (axual_flink_cluster.test_cluster below) IS managed by this
# config, since none currently exists on this instance-cluster.
#
# IMPORTANT: never `terraform state rm` any resource here. Let `terraform destroy`
# handle dependency order on its own - a FLINK_SQL deployment's target cannot be
# changed after creation, and DELETE requires its original target Flink Cluster to
# still exist. Manually removing a resource from state before destroying its
# dependents can permanently strand a deployment (undeletable, unretargetable,
# only fixable via DB-level cleanup - this happened once during testing, see
# TESTING.md).

variable "instance_short_name" {
  type    = string
  default = "dev"
}

variable "group_name" {
  type    = string
  default = "Admins"
}

variable "your_email" {
  type    = string
  default = "admin@axual.com"
}

variable "instance_cluster_id" {
  description = "Uid of the existing Instance-Cluster to register the Flink Cluster on"
  type        = string
  default     = "f076e00035ab49a1be62916b5691b569" # "jupiter" cluster on the "dev" instance
}

variable "flink_api_token" {
  description = "Ververica namespace-scoped API token"
  type        = string
  sensitive   = true
  default     = "5849d1bd-915e-432f-8995-a7f173a657d7" # pass via TF_VAR_flink_api_token instead
}

# Read-only references to pre-existing, shared infrastructure - never created or destroyed here.
data "axual_instance" "test_instance" {
  short_name = var.instance_short_name
}

data "axual_group" "test_group" {
  name = var.group_name
}

data "axual_user" "me" {
  email = var.your_email
}

############################
# 1) Environment
############################
resource "axual_environment" "flink_test" {
  name                 = "tf-flink-sql-full-test"
  short_name           = "tfflinkfull"
  description          = "Isolated environment for testing the Flink SQL provider changes"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

############################
# 2) Flink Cluster
############################
resource "axual_flink_cluster" "test_cluster" {
  instance_id       = data.axual_instance.test_instance.id
  cluster_id        = var.instance_cluster_id
  name              = "tf-flink-sql-full-test-cluster"
  description       = "Flink Cluster for the AXPD-11717 full example"
  url               = "https://ververica.qa.np.westeurope.azure.axual.cloud"
  workspace         = "defaultworkspace"
  namespace         = "default"
  deployment_target = "default-target"
  api_token         = var.flink_api_token
}

############################
# 3) Flink SQL Application
############################
resource "axual_application" "flink_app" {
  name             = "tf-flink-sql-full-test-app"
  application_type = "FLINK_SQL"
  short_name       = "tf_flink_sql_full_test_app"
  application_id   = "io.axual.terraform.flinksqlfulltest"
  owners           = data.axual_group.test_group.id
  visibility       = "Public"
  description      = "Full FLINK_SQL example: 2 consume + 2 produce topics (String, Avro, Protobuf)"
}

# FLINK_SQL apps need an axual_application_credential before an access grant can be
# created (AXPD-11717 claims otherwise - not true against the real backend).
resource "axual_application_credential" "flink_app_credential" {
  application = axual_application.flink_app.id
  environment = axual_environment.flink_test.id
  target      = "KAFKA"
}

############################
# 4) Schemas for the Avro/Protobuf topics
############################
resource "axual_schema_version" "avro_value_schema" {
  # chomp() strips the trailing newline file() reads - the API trims it server-side on
  # save, so leaving it in causes a permanent diff (and schema versions are immutable,
  # so that diff becomes a hard error: "API does not allow update of schema version").
  body        = chomp(file("avro-schemas/simple_avro_value.avsc"))
  version     = "1.0.0"
  description = "Simple Avro value schema for the Flink SQL consume topic"
  type        = "AVRO"
}

resource "axual_schema_version" "protobuf_value_schema" {
  body        = chomp(file("protobuf-schemas/simple_protobuf_value.proto"))
  version     = "1.0.0"
  description = "Simple Protobuf value schema for the Flink SQL produce topic"
  type        = "PROTOBUF"
}

############################
# 5) Four topics: 2 consume (String, Avro), 2 produce (String, Protobuf)
############################
resource "axual_topic" "string_in" {
  name             = "tf-flink-string-in"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Consume topic: plain String key/value"
}

resource "axual_topic" "avro_in" {
  name             = "tf-flink-avro-in"
  key_type         = "String"
  value_type       = "AVRO"
  value_schema     = axual_schema_version.avro_value_schema.schema_id
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Consume topic: String key, Avro value"
}

resource "axual_topic" "string_out" {
  name             = "tf-flink-string-out"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Produce topic: plain String key/value"
}

resource "axual_topic" "protobuf_out" {
  name             = "tf-flink-protobuf-out"
  key_type         = "String"
  value_type       = "PROTOBUF"
  value_schema     = axual_schema_version.protobuf_value_schema.schema_id
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Produce topic: String key, Protobuf value"
}

############################
# 6) Topic configs (all 4, in the new environment)
############################
resource "axual_topic_config" "string_in_config" {
  partitions     = 1
  retention_time = 864000 # 10 days
  topic          = axual_topic.string_in.id
  environment    = axual_environment.flink_test.id
  properties     = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_topic_config" "avro_in_config" {
  partitions           = 1
  retention_time       = 864000
  topic                = axual_topic.avro_in.id
  environment          = axual_environment.flink_test.id
  properties           = { "segment.ms" = "600012", "retention.bytes" = "-1" }
  value_schema_version = axual_schema_version.avro_value_schema.id
}

resource "axual_topic_config" "string_out_config" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.string_out.id
  environment    = axual_environment.flink_test.id
  properties     = { "segment.ms" = "600012", "retention.bytes" = "-1" }
}

resource "axual_topic_config" "protobuf_out_config" {
  partitions           = 1
  retention_time       = 864000
  topic                = axual_topic.protobuf_out.id
  environment          = axual_environment.flink_test.id
  properties           = { "segment.ms" = "600012", "retention.bytes" = "-1" }
  value_schema_version = axual_schema_version.protobuf_value_schema.id
}

############################
# 7) Access grants: 2 CONSUMER (string_in, avro_in), 2 PRODUCER (string_out, protobuf_out)
############################
resource "axual_application_access_grant" "consume_string_in" {
  application = axual_application.flink_app.id
  topic       = axual_topic.string_in.id
  environment = axual_environment.flink_test.id
  access_type = "CONSUMER"
  depends_on = [
    axual_topic_config.string_in_config,
    axual_application_credential.flink_app_credential,
  ]
}

resource "axual_application_access_grant_approval" "consume_string_in_approval" {
  application_access_grant = axual_application_access_grant.consume_string_in.id
}

resource "axual_application_access_grant" "consume_avro_in" {
  application = axual_application.flink_app.id
  topic       = axual_topic.avro_in.id
  environment = axual_environment.flink_test.id
  access_type = "CONSUMER"
  depends_on = [
    axual_topic_config.avro_in_config,
    axual_application_credential.flink_app_credential,
  ]
}

resource "axual_application_access_grant_approval" "consume_avro_in_approval" {
  application_access_grant = axual_application_access_grant.consume_avro_in.id
}

resource "axual_application_access_grant" "produce_string_out" {
  application = axual_application.flink_app.id
  topic       = axual_topic.string_out.id
  environment = axual_environment.flink_test.id
  access_type = "PRODUCER"
  depends_on = [
    axual_topic_config.string_out_config,
    axual_application_credential.flink_app_credential,
  ]
}

resource "axual_application_access_grant_approval" "produce_string_out_approval" {
  application_access_grant = axual_application_access_grant.produce_string_out.id
}

resource "axual_application_access_grant" "produce_protobuf_out" {
  application = axual_application.flink_app.id
  topic       = axual_topic.protobuf_out.id
  environment = axual_environment.flink_test.id
  access_type = "PRODUCER"
  depends_on = [
    axual_topic_config.protobuf_out_config,
    axual_application_credential.flink_app_credential,
  ]
}

resource "axual_application_access_grant_approval" "produce_protobuf_out_approval" {
  application_access_grant = axual_application_access_grant.produce_protobuf_out.id
}

############################
# 8) Flink SQL Deployment - reads both consume topics, writes to both produce topics.
#    generate_tables_sql = true auto-generates the CREATE TABLE DDL from the topics'
#    schemas, so the script only needs the transformation logic. SELECT * avoids
#    guessing the auto-generated column names for the Avro/Protobuf value fields.
############################
resource "axual_application_deployment" "flink_deployment" {
  environment = axual_environment.flink_test.id
  application = axual_application.flink_app.id
  target_id   = axual_flink_cluster.test_cluster.id
  # Single INSERT INTO ... SELECT only. Ververica's SQL Gateway rejects multiple bare
  # INSERT INTO statements unless wrapped in a BEGIN STATEMENT SET ... END; block (its
  # multi-sink mode) - but Axual's own SQL validator rejects that exact wrapper as a
  # FORBIDDEN_STATEMENT_TYPE ("Statements other than CREATE TEMPORARY TABLE / INSERT
  # INTO ... SELECT are not allowed"). The two validators contradict each other, so
  # writing to two topics from one deployment is not currently possible - confirmed
  # against the real backend, worth raising with the platform team.
  #
  # Also confirmed: the auto-generated table for a PROTOBUF-value topic exposes the
  # value as a single opaque `value BYTES` column, not per-field columns like Avro
  # does - Ververica rejected `SELECT * FROM tf_flink_avro_in` into
  # tf_flink_protobuf_out with "Column types of query result and sink ... do not
  # match. Query schema: [key, id, payload], Sink schema: [key, value: BYTES]".
  # Producing a real protobuf-encoded byte value needs a format-aware function this
  # plain SQL doesn't have, so this deployment transforms avro_in -> string_out
  # instead (String sinks use simple [key, value] STRING columns). tf_flink_string_in
  # and tf_flink_protobuf_out are still fully created, configured, and
  # access-granted (per the 2 consume + 2 produce requirement) even though this
  # single statement doesn't use them.
  sql_script          = <<-SQL
    BEGIN STATEMENT SET;
    INSERT INTO tf_flink_string_out (key, `value`)
    SELECT key, payload FROM tf_flink_avro_in;
    INSERT INTO tf_flink_string_out (key, `value`)
    SELECT key, `value` FROM tf_flink_string_in;
    END;
  SQL
  generate_tables_sql = true
  task_size           = "S"

  depends_on = [
    axual_application_access_grant_approval.consume_string_in_approval,
    axual_application_access_grant_approval.consume_avro_in_approval,
    axual_application_access_grant_approval.produce_string_out_approval,
    axual_application_access_grant_approval.produce_protobuf_out_approval,
  ]
}

output "flink_cluster_id" {
  value = axual_flink_cluster.test_cluster.id
}

output "flink_application_id" {
  value = axual_application.flink_app.id
}

output "flink_deployment_id" {
  value = axual_application_deployment.flink_deployment.id
}

############################
# Simple String Key-Value Example
############################

resource "axual_environment" "string_test" {
  name                 = "tf-flink-string-test"
  short_name           = "tfflinkstring"
  description          = "Simple Flink SQL example with string topics only"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_application" "string_app" {
  name             = "tf-flink-string-app"
  application_type = "FLINK_SQL"
  short_name       = "tf_flink_string_app"
  application_id   = "io.axual.terraform.flinksqlstring"
  owners           = data.axual_group.test_group.id
  visibility       = "Public"
  description      = "Simple Flink SQL app with string topics"
}

resource "axual_application_credential" "string_app_cred" {
  application = axual_application.string_app.id
  environment = axual_environment.string_test.id
  target      = "KAFKA"
}

resource "axual_topic" "string_input" {
  name             = "tf-flink-string-input"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Input topic for string transformation"
}

resource "axual_topic_config" "string_input_config" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.string_input.id
  environment    = axual_environment.string_test.id
  properties     = {}
}

resource "axual_topic" "string_output" {
  name             = "tf-flink-string-output"
  key_type         = "String"
  value_type       = "String"
  owners           = data.axual_group.test_group.id
  retention_policy = "delete"
  properties       = {}
  description      = "Output topic for string transformation"
}

resource "axual_topic_config" "string_output_config" {
  partitions     = 1
  retention_time = 864000
  topic          = axual_topic.string_output.id
  environment    = axual_environment.string_test.id
  properties     = {}
}

resource "axual_application_access_grant" "string_input_consume" {
  application = axual_application.string_app.id
  topic       = axual_topic.string_input.id
  environment = axual_environment.string_test.id
  access_type = "CONSUMER"

  depends_on = [
    axual_application_credential.string_app_cred,
    axual_topic_config.string_input_config
  ]
}

resource "axual_application_access_grant_approval" "string_input_consume_approval" {
  application_access_grant = axual_application_access_grant.string_input_consume.id
}

resource "axual_application_access_grant" "string_output_produce" {
  application = axual_application.string_app.id
  topic       = axual_topic.string_output.id
  environment = axual_environment.string_test.id
  access_type = "PRODUCER"

  depends_on = [
    axual_application_credential.string_app_cred,
    axual_topic_config.string_output_config
  ]
}

resource "axual_application_access_grant_approval" "string_output_produce_approval" {
  application_access_grant = axual_application_access_grant.string_output_produce.id
}

resource "axual_application_deployment" "string_deployment" {
  environment         = axual_environment.string_test.id
  application         = axual_application.string_app.id
  target_id           = axual_flink_cluster.test_cluster.id
  generate_tables_sql = true
  task_size           = "S"

  sql_script = <<-SQL
    BEGIN STATEMENT SET;
    INSERT INTO tf_flink_string_output (`key`, `value`)
    SELECT CONCAT(`key`, '_processed'), CONCAT(`value`, '_processed') FROM tf_flink_string_input;
    END;
  SQL

  depends_on = [
    axual_application_access_grant_approval.string_input_consume_approval,
    axual_application_access_grant_approval.string_output_produce_approval,
  ]
}
