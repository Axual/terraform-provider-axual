resource "axual_cluster" "tf_test_cluster" {
  name          = "tf-test-cluster-renamed"
  description   = "Axual's TF Test Cluster, updated"
  location      = "Utrecht"
  provider_type = "apache-kafka"
  broker = {
    bootstrap_servers = local.kafka_bootstrap_servers
    protocol          = "MTLS"
    certificate       = file(local.kafka_client_certificate_file)
    private_key       = file(local.kafka_client_private_key_file)
    truststore        = file(local.kafka_truststore_file)
    ca_validation     = false
  }
  topic_pattern                    = "{tenant}-{instance}-{environment}-{topic}"
  group_pattern                    = "{tenant}-{instance}-{environment}-{group}"
  transactional_pattern            = "{tenant}-{instance}-{environment}-{transactional.id}"
  kafka_version                    = "4.2.0"
  bootstrap_urls_by_authentication = { MUTUAL_TLS = local.kafka_bootstrap_servers }
  kafka_listeners                  = { MUTUAL_TLS = local.kafka_bootstrap_servers }
  ssl_auth_mode                    = "certificate"
  oauth_principals_prefix          = "tf-test-"
  billing_cloud_enabled            = true
}
