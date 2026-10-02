data "axual_connect_cluster" "by_id" {
  instance_id = local.connect_cluster_instance_id
  cluster_id  = local.connect_cluster_cluster_id
  id          = local.connect_cluster_mtls_id
}
