resource "axual_cluster" "tf_test_ic_cluster" {
  name          = "tf-test-ic-cluster"
  provider_type = "apache-kafka"
  broker = {
    bootstrap_servers = local.kafka_bootstrap_servers
    protocol          = "MTLS"
    certificate       = file(local.kafka_client_certificate_file)
    private_key       = file(local.kafka_client_private_key_file)
    truststore        = file(local.kafka_truststore_file)
    ca_validation     = false
  }
  topic_pattern         = "{tenant}-{instance}-{environment}-{topic}"
  group_pattern         = "{tenant}-{instance}-{environment}-{group}"
  transactional_pattern = "{tenant}-{instance}-{environment}-{transactional.id}"
  ssl_auth_mode         = "fullChainCertificate"
}

resource "axual_instance" "tf_test_ic_instance" {
  name            = "tf-test-ic-instance"
  short_name      = "tftestic"
  clusters        = [axual_cluster.tf_test_ic_cluster.id]
  ca_certificates = [file(local.instance_ca_certificate_file)]
}
