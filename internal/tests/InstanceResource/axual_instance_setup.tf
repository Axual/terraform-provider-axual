resource "axual_cluster" "tf_test_instance_cluster_a" {
  name          = "tf-test-instance-cluster-a"
  provider_type = "apache-kafka"
  broker = {
    bootstrap_servers = local.kafka_bootstrap_servers
    protocol          = "MTLS"
    certificate       = file(local.kafka_client_certificate_file)
    private_key       = file(local.kafka_client_private_key_file)
    truststore        = file(local.kafka_truststore_file)
    ca_validation     = false
  }
  topic_pattern = "{tenant}-{instance}-{environment}-{topic}"
  ssl_auth_mode = "fullChainCertificate"
}

resource "axual_cluster" "tf_test_instance_cluster_b" {
  name          = "tf-test-instance-cluster-b"
  provider_type = "apache-kafka"
  broker = {
    bootstrap_servers = local.kafka_bootstrap_servers
    protocol          = "MTLS"
    certificate       = file(local.kafka_client_certificate_file)
    private_key       = file(local.kafka_client_private_key_file)
    truststore        = file(local.kafka_truststore_file)
    ca_validation     = false
  }
  topic_pattern = "{tenant}-{instance}-{environment}-{topic}"
  ssl_auth_mode = "fullChainCertificate"
}
