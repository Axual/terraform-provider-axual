resource "axual_instance_cluster" "tf_test_ic" {
  instance = axual_instance.tf_test_ic_instance.id
  cluster  = axual_cluster.tf_test_ic_cluster.id
  schema_registry = {
    type           = "APICURIO"
    url            = local.schema_registry_url
    authentication = "Basic-Auth"
    username       = local.schema_registry_username
    password       = local.schema_registry_password
  }
}
