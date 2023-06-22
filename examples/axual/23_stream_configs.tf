resource "axual_stream_config" "logs_in_example" {
  partitions = 1
  retention_time = 864000
  stream = axual_stream.logs.id
  environment = "7237a4093d7948228d431a603c31c904"
  properties = {"segment.ms"="600012", "retention.bytes"="1"}
}

resource "axual_stream_config" "logs_in_staging" {
  partitions = 1
  retention_time = 1001000
  stream = axual_stream.logs.id
  environment = axual_environment.staging.id
  properties = {"segment.ms"="60002", "retention.bytes"="100"}
}

resource "axual_stream_config" "support_in_staging" {
  partitions = 1
  retention_time = 1001
  stream = axual_stream.support.id
  environment = axual_environment.staging.id
  properties = {"segment.ms"="60002", "retention.bytes"="1234"}
}

output "logs_in_staging_id" {
  description = "Logs Stream Config in Staging ID"
  value = axual_stream_config.logs_in_staging.id
}