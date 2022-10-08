#Logged in Terraform User(by default kubernetes@axual.com) needs to have application admin rights(for create access request) and stream admin rights(for revoking access request) or be owner of the application and the stream (by being user in the same group as the application's and stream's owner group)
resource "axual_application_access_grant" "terra_grant_1" {
  application = axual_application.gitops_test_application_2.id
  stream = axual_stream.gitops_test_stream2.id
  environment = "7237a4093d7948228d431a603c31c904"
  access_type = "Consumer"
  depends_on = [axual_application_principal.test_application_principal, axual_stream_config.gitops_test_stream_config_2, axual_stream.gitops_test_stream2]
}