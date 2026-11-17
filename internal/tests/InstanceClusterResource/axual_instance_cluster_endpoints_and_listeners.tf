resource "axual_instance_cluster" "tf_test_ic" {
  instance = axual_instance.tf_test_ic_instance.id
  cluster  = axual_cluster.tf_test_ic_cluster.id
  schema_registry = {
    type           = "APICURIO"
    url            = local.schema_registry_url
    authentication = "No-Auth"
    listeners      = { public = local.schema_registry_url }
    client_endpoints = {
      public = { api = "APICURIO_V3", url = local.schema_registry_url, used_by = "ALL_APPS" }
    }
  }
}
