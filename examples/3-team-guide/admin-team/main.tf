# Users must be created in the Authentication Provider (Keycloak, Auth0, etc.) and Axual Platform Manager first.
# Then import them into Terraform state using: terraform import axual_user.<name> <USER_UID>
# Or use data sources to reference existing users as shown below.

# Option 1: Use data sources to reference existing users (recommended)
data "axual_user" "admin" {
  email_address = "admin_team@axual.com"
}

data "axual_user" "application_author" {
  email_address = "application_team@axual.com"
}

data "axual_user" "topic_author" {
  email_address = "topic_team@axual.com"
}

resource "axual_group" "admin-team" {
  name          = "Admin Team"
  members       = [
    data.axual_user.admin.id
  ]
}

resource "axual_group" "topic-author-team" {
  name          = "Topic Author Team"
  members       = [
    data.axual_user.topic_author.id,
  ]
}

resource "axual_group" "application-author-team" {
  name          = "Application Author Team"
  members       = [
    data.axual_user.application_author.id,
  ]
}

data "axual_instance" "testInstance"{
  short_name = "dta"
}

resource "axual_environment" "development" {
  name = "tf-development"
  short_name = "tfdev"
  description = "This is the TF development environment. Typo fix"
  color = "#19b9be"
  visibility = "Public"
  authorization_issuer = "Stream owner"
  instance = data.axual_instance.testInstance.id
  owners = axual_group.admin-team.id
}