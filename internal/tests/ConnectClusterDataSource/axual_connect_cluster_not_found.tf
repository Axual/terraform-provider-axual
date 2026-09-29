data "axual_connect_cluster" "missing" {
  instance_id = local.connect_cluster_instance_id
  cluster_id  = local.connect_cluster_cluster_id
  id          = "does-not-exist"
}
