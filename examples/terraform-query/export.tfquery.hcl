# Finds the resources of one team and one environment, so they can be written to .tf files with:
#
#   terraform init
#   terraform query                                   # print what is found
#   terraform query -generate-config-out=generated.tf # write resource and import blocks
#
# Needs Terraform 1.14 or newer. Replace every YOUR_* value below. See the guide "Exporting existing
# resources with terraform query" for what to check in generated.tf before you apply it.

# Resources of the application team.

list "axual_topic" "team" {
  provider = axual
  limit    = 1000
  config {
    owners = "YOUR_GROUP_NAME" # group name or group ID
  }
}

list "axual_topic_config" "team" {
  provider = axual
  limit    = 1000
  config {
    owners      = "YOUR_GROUP_NAME"
    environment = "YOUR_ENVIRONMENT_SHORT_NAME" # short name, name or ID
  }
}

list "axual_application" "team" {
  provider = axual
  limit    = 1000
  config {
    owners = "YOUR_GROUP_NAME"
  }
}

list "axual_application_access_grant" "team" {
  provider = axual
  limit    = 1000
  config {
    owners      = "YOUR_GROUP_NAME" # the team that owns the application
    environment = "YOUR_ENVIRONMENT_SHORT_NAME"
  }
}

# configs is sensitive, so generated.tf has `configs = null # sensitive`. Fill it in before you apply;
# the provider stops the plan with "Missing configs" until you do.
list "axual_application_deployment" "team" {
  provider = axual
  limit    = 1000
  config {
    owners      = "YOUR_GROUP_NAME"
    environment = "YOUR_ENVIRONMENT_SHORT_NAME"
  }
}

# Resources of the topic team: it approves the grants on its topics.

list "axual_application_access_grant_approval" "topic_team" {
  provider = axual
  limit    = 1000
  config {
    owners      = "YOUR_TOPIC_TEAM_GROUP_NAME" # the team that owns the topics
    environment = "YOUR_ENVIRONMENT_SHORT_NAME"
  }
}
