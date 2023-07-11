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
