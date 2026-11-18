resource "axual_cluster" "tf_test_ic2_cluster" {
  for_each      = toset(["a", "b"])
  name          = "tf-test-ic2-cluster-${each.key}"
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

resource "axual_instance" "tf_test_ic2_instance" {
  name            = "tf-test-ic2-instance"
  short_name      = "tftestic2"
  clusters        = [for c in axual_cluster.tf_test_ic2_cluster : c.id]
  ca_certificates = [file(local.instance_ca_certificate_file)]
}

# One schema registry per Cluster, written in parallel on the same Instance.
resource "axual_instance_cluster" "tf_test_ic2" {
  for_each = axual_cluster.tf_test_ic2_cluster
  instance = axual_instance.tf_test_ic2_instance.id
  cluster  = each.value.id
  schema_registry = {
    type           = "APICURIO"
    url            = local.schema_registry_url
    authentication = "No-Auth"
    properties     = { "tf.test.step" = var.step }
  }
}

variable "step" {
  type = string
}
