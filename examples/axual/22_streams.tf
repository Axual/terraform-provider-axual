resource "axual_stream" "gitops_test_stream2" {
  name = "gitops_test_stream2"
  key_type = "String"
  value_type = "String"
  owners = axual_group.gitops_test.id
  retention_policy = "delete"
  properties = { }
}

resource "axual_stream" "gitops_test_stream3" {
  name = "gitops_test_stream3"
  key_type = "String"
  value_type = "String"
  owners = axual_group.gitops_test.id
  retention_policy = "delete"
  properties = { }
}