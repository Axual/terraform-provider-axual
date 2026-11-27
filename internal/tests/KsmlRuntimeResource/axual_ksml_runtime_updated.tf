resource "axual_ksml_runtime" "tf_test_ksml_runtime" {
  instance_id = data.axual_instance.test_instance.id
  cluster_id  = local.cluster_id
  # Platform Manager does not call the URL when a KSML runtime is registered or updated.
  url    = "https://tf-test-ksml-runtime-updated.example.com"
  name   = "tf-test-ksml-runtime"
  owners = data.axual_group.test_group.id
  # Emptied on update; the next step leaves the attribute out, which must produce no diff.
  authorized_groups = []
  description       = "Axual's TF Test KSML runtime, updated"

  # Added on update; the next step removes the block again, which the PATCH can only express as
  # an explicit null.
  schema_registries = [
    {
      type = "CONFLUENT"
      url  = "https://schema-registry.example.com/apis/ccompat/v7"
    },
  ]
}
