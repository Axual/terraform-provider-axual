resource "axual_instance_cluster" "tf_test_ic" {
  instance = axual_instance.tf_test_ic_instance.id
  cluster  = axual_cluster.tf_test_ic_cluster.id
  schema_registry = {
    type           = "CONFLUENT"
    url            = local.schema_registry_url
    authentication = "No-Auth"
    api            = "APICURIO_V3"
  }
}
