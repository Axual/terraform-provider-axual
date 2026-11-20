resource "axual_cluster" "tf_query_cluster" {
  name          = "tf-query-cluster"
  description   = "Cluster for the terraform query tests"
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

resource "axual_instance" "tf_query_instance" {
  name                       = "tf-query-instance"
  short_name                 = "tfqueryinst"
  clusters                   = [axual_cluster.tf_query_cluster.id]
  ca_certificates            = [file(local.instance_ca_certificate_file)]
  granular_browse_permission = true
}
