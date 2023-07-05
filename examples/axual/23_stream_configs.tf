resource "axual_stream_config" "logs_in_dev" {
  partitions = 1
  retention_time = 864000
  stream = axual_stream.logs.id
  environment = axual_environment.development.id
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

resource "axual_stream_config" "logs_in_production" {
  partitions = 2
  retention_time = 86400000
  stream = axual_stream.logs.id
  environment = axual_environment.production.id
  properties = {"segment.ms"="600000", "retention.bytes"="10089"}
}

resource "axual_stream_config" "support_in_production" {
  partitions = 4
  retention_time = 10000000
  stream = axual_stream.support.id
  environment = axual_environment.production.id
  properties = {"segment.ms"="600000", "retention.bytes"="10089"}

}

output "logs_in_dev_id" {
  description = "Logs Stream Config in Development ID"
  value = axual_stream_config.logs_in_dev.id
}

output "logs_in_staging_id" {
  description = "Logs Stream Config in Staging ID"
  value = axual_stream_config.logs_in_staging.id
}


output "logs_in_production_id" {
  description = "Logs Stream Config in Production ID"
  value = axual_stream_config.logs_in_production.id
}

output "support_in_staging_id" {
  description = "Support Stream Config in Staging ID"
  value = axual_stream_config.support_in_staging.id
}

output "support_in_production_id" {
  description = "Support Stream Config in Production ID"
  value = axual_stream_config.support_in_production.id
}