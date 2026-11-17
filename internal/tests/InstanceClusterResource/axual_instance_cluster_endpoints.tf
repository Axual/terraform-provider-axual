resource "axual_instance_cluster" "tf_test_ic" {
  instance = axual_instance.tf_test_ic_instance.id
  cluster  = axual_cluster.tf_test_ic_cluster.id
  schema_registry = {
    type           = "APICURIO"
    url            = local.schema_registry_url
    authentication = "Basic-Auth"
    username       = local.schema_registry_username
    password       = local.schema_registry_password
    api            = "APICURIO_V3"
    client_endpoints = {
      internal = { api = "APICURIO_V3", url = local.schema_registry_url, used_by = "MANAGED_APPS" }
      public   = { api = "APICURIO_V3", url = "https://schema-registry.example.com/apis/registry/v3", used_by = "ALL_APPS" }
    }
  }
}
