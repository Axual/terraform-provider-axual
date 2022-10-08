#Logged in Terraform User(by default kubernetes@axual.com) needs to have application admin rights(for create access request) and stream admin rights(for revoking access request) or be owner of the application and the stream (by being user in the same group as the application's and stream's owner group)
resource "axual_application_principal" "test_application_principal" {
  environment = "7237a4093d7948228d431a603c31c904"
  application = axual_application.gitops_test_application_2.id
  principal = file("${path.module}/test3.pem")
}
#Environment needs to support OAUTH for OAUTH bearer application principal to work
#resource "axual_application_principal" "test_application_principal5" {
#  environment = "7237a4093d7948228d431a603c31c904"
#  application = axual_application.gitops_test_application_3.id
#  principal = "axual-test-0000"
#  custom = true
#}