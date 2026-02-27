resource "axual_environment" "tf-test-env-empty-maps" {
  name                 = "tf-empty-maps"
  short_name           = "tfempty"
  description          = "Environment to test import with empty properties and settings"
  color                = "#19b9be"
  visibility           = "Private"
  authorization_issuer = "Auto"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
  properties           = {}
  settings             = {}
}
