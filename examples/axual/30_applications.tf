resource "axual_application" "gitops_test_application_2" {
  name    = "gitops_test_application_2"
  application_type     = "Custom"
  short_name = "gitops_test_application_2"
  application_id = "axual.gitops.test2"
  owners = axual_group.gitops_test.id
  type = "Java"
  visibility = "Public"
  description = "Axual's Test Application for Gitops 2"
  depends_on = [axual_stream_config.gitops_test_stream_config_2, axual_stream.gitops_test_stream2]
}

resource "axual_application" "gitops_test_application_3" {
  name    = "gitops_test_application_3"
  application_type     = "Custom"
  short_name = "gitops_test_application_3"
  application_id = "axual.gitops.test3"
  owners = axual_group.gitops_test.id
  type = "Java"
  visibility = "Public"
  description = "Axual's Test Application for Gitops 3"
  depends_on = [axual_stream_config.gitops_test_stream_config_3, axual_stream.gitops_test_stream2]
}