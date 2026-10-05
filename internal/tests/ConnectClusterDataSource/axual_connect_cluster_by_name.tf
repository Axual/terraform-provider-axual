data "axual_connect_cluster" "by_name" {
  instance_id = local.connect_cluster_instance_id
  cluster_id  = local.connect_cluster_cluster_id
  name        = local.connect_cluster_sasl_name
}
