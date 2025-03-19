resource "axual_environment" "tf-test-env" {
  name                 = "tf-development"
  short_name           = "tfdev"
  description          = "This is the terraform testing environment"
  color                = "#19b9be"
  visibility           = "Private"
  authorization_issuer = "Auto"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

data "axual_environment" "tf-test-env-imported" {
  name = axual_environment.tf-test-env.name
}