resource "axual_cluster" "tf_test_cluster" {
  name          = "tf-test-cluster"
  provider_type = "apache-kafka"
  broker = {
    bootstrap_servers = local.kafka_bootstrap_servers
    protocol          = "MTLS"
    certificate       = file(local.kafka_client_certificate_file)
  }
}
