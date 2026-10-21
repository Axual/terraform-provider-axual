resource "axual_instance" "tf_test_instance" {
  name                   = "tf-test-instance-renamed"
  short_name             = "tftestinst2"
  clusters               = [axual_cluster.tf_test_instance_cluster_b.id]
  ca_certificates        = [file(local.instance_ca_certificate_file)]
  authentication_methods = ["MUTUAL_TLS"]
  properties             = {}
}
