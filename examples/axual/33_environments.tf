resource "axual_environment" "test-env" {
  name = "test-env"
  short_name = "testenv"
  description = "This sis a long descripion"
  color = "#19b9be"
  visibility = "Private"
  authorization_issuer = "Auto"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = "dd84b3ee8e4341fbb58704b18c10ec5c"
}

resource "axual_environment" "test-env2" {
  name = "test-env2"
  short_name = "testenv2"
  description = "This sis a long descripion"
  color = "#3b0d98"
  visibility = "Private"
  authorization_issuer = "Auto"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = "dd84b3ee8e4341fbb58704b18c10ec5c"
}

resource "axual_environment" "test-env3" {
  name = "test-env3"
  short_name = "testenv3"
  description = "This sis a long descripion"
  color = "#3b0d98"
  visibility = "Public"
  authorization_issuer = "Stream owner"
  instance = "51be2a6a5eee481198787dc346ab6608"
  owners = "dd84b3ee8e4341fbb58704b18c10ec5c"
  properties = {
    "segment.ms"="60002"
  }

}