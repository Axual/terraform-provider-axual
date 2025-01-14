# axual_user resources should be imported after creating them in the Authentication Provider and Axual Platform Manager.
resource "axual_user" "admin" {
  first_name    = "Admin"
  last_name     = "Admin"
  email_address = "admin@axual.com"
  roles         = [
    { name = "TENANT_ADMIN" },
    { name = "ENVIRONMENT_AUTHOR" },
  ]
}

resource "axual_user" "application_author" {
  first_name    = "Application"
  last_name     = "Author"
  email_address = "application_author@axual.com"
  roles         = [
    { name = "APPLICATION_AUTHOR" },
  ]
}

resource "axual_user" "topic_author" {
  first_name    = "Topic"
  last_name     = "Author"
  email_address = "topic_author@axual.com"
  roles         = [
    { name = "STREAM_AUTHOR" },
    { name = "SCHEMA_AUTHOR" },
  ]
}

resource "axual_group" "admin-team" {
  name          = "Admin Team"
  members       = [
    axual_user.admin.id
  ]
}

resource "axual_group" "topic-author-team" {
  name          = "Topic Author Team"
  members       = [
    axual_user.topic_author.id,
  ]
}

resource "axual_group" "application-author-team" {
  name          = "Application Author Team"
  members       = [
    axual_user.application_author.id,
  ]
}
#

data "axual_instance" "testInstance"{
  name = "Dev Test Acceptance"
}

resource "axual_environment" "development" {
  name = "tf-development"
  short_name = "tfdev"
  description = "This is the TF development environment. Typo fix"
  color = "#19b9be"
  visibility = "Public"
  authorization_issuer = "Auto"
  instance = data.axual_instance.testInstance.id
  owners = axual_group.admin-team.id
}