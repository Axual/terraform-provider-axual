resource "axual_cluster" "tf_test_cluster" {
  name          = "tf-test-cluster"
  provider_type = "aiven"
  broker = {
    bootstrap_servers = local.kafka_bootstrap_servers
    protocol          = "MTLS"
  }
}
