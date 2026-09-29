# The same environment and application; Terraform forgets the deployment and the principal without
# destroying them, so the next steps can import them back.

resource "axual_environment" "imp" {
  name                 = "tf-ds-import"
  short_name           = "tfdsimport"
  description          = "axual_application_deployment_state import test environment"
  color                = "#19b9be"
  visibility           = "Public"
  authorization_issuer = "Stream owner"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_application" "imp" {
  name              = "tf-test-app-ds-import"
  application_type  = "Connector"
  application_class = "io.axual.connect.plugins.http.HttpSinkConnector"
  short_name        = "tfdsimportapp"
  application_id    = "tf.test.ds.import"
  owners            = data.axual_group.test_group.id
  type              = "SINK"
  visibility        = "Public"
  description       = "axual_application_deployment_state import test application"
}

removed {
  from = axual_application_principal.imp
  lifecycle {
    destroy = false
  }
}

removed {
  from = axual_application_deployment.imp
  lifecycle {
    destroy = false
  }
}
