resource "axual_cluster" "tf_test_cluster" {
  name          = "tf-test-cluster"
  description   = "Axual's TF Test Cluster"
  provider_type = "apache-kafka"
  broker = {
    bootstrap_servers = local.kafka_bootstrap_servers
    protocol          = "MTLS"
    certificate       = file(local.kafka_client_certificate_file)
    private_key       = file(local.kafka_client_private_key_file)
    truststore        = file(local.kafka_truststore_file)
    ca_validation     = false
  }
  ssl_auth_mode = "fullChainCertificate"
}
