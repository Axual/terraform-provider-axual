resource "axual_instance" "tf_test_instance" {
  name                          = "tf-test-instance-renamed"
  short_name                    = "tftestinst2"
  description                   = "Axual's TF Test Instance, updated"
  clusters                      = [axual_cluster.tf_test_instance_cluster_a.id, axual_cluster.tf_test_instance_cluster_b.id]
  ca_certificates               = [file(local.instance_ca_certificate_file)]
  authentication_methods        = ["MUTUAL_TLS", "SCRAM_SHA_512"]
  properties                    = { "replication.factor" = "1" }
  granular_browse_permission    = true
  topic_owner_browse_permission = true
  environment_mapping           = false
  self_approval                 = false
  certificate_reuse_prevention  = "TENANT"
  flink_enabled                 = true
}
