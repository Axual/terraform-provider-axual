resource "axual_stream_config" "gitops_test_stream_config_2" {
  partitions = 1
  retention_time = 1000
  stream = axual_stream.gitops_test_stream2.id
  environment = "7237a4093d7948228d431a603c31c904"
  properties = {"segment.ms"="600012", "retention.bytes"="1"}
}

resource "axual_stream_config" "gitops_test_stream_config_3" {
  partitions = 1
  retention_time = 1001
  stream = axual_stream.gitops_test_stream3.id
  environment = "7237a4093d7948228d431a603c31c904"
  properties = {"segment.ms"="60002", "retention.bytes"="1"}
}