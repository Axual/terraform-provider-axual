resource "axual_environment" "awesome-env" {
  name = "team-awesome"
  short_name = "awesome"
  description = "This is a test environment"
  color = "#19b9be"
  visibility = "Private"
  authorization_issuer = "Auto"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = "dd84b3ee8e4341fbb58704b18c10ec5c"
}

resource "axual_environment" "staging-env" {
  name = "staging"
  short_name = "staging"
  description = "Staging contains close to real world data"
  color = "#3b0d98"
  visibility = "Public"
  authorization_issuer = "Auto"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = "dd84b3ee8e4341fbb58704b18c10ec5c"
}

resource "axual_environment" "production-env" {
  name = "production"
  short_name = "production"
  description = "Real world production environment"
  color = "#3b0d98"
  visibility = "Public"
  authorization_issuer = "Stream owner"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = "dd84b3ee8e4341fbb58704b18c10ec5c"
  properties = {
    "segment.ms"="60002"
  }

}