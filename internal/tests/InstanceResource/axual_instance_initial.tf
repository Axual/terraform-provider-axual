resource "axual_instance" "tf_test_instance" {
  name            = "tf-test-instance"
  short_name      = "tftestinst"
  description     = "Axual's TF Test Instance"
  clusters        = [axual_cluster.tf_test_instance_cluster_a.id]
  ca_certificates = [file(local.instance_ca_certificate_file)]
}
