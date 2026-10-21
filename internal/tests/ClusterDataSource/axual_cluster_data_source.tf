resource "axual_cluster" "tf_test_ds_cluster" {
  name          = "tf-test-ds-cluster"
  description   = "Cluster the data source looks up"
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
  bootstrap_urls_by_authentication = { MUTUAL_TLS = local.kafka_bootstrap_servers }
}

data "axual_cluster" "tf_test_ds_cluster" {
  name       = axual_cluster.tf_test_ds_cluster.name
  depends_on = [axual_cluster.tf_test_ds_cluster]
}
