data "axual_connect_cluster" "neither" {
  instance_id = local.connect_cluster_instance_id
  cluster_id  = local.connect_cluster_cluster_id
}
