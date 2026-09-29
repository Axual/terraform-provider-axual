resource "axual_application_deployment_state" "kc_sasl" {
  application_deployment = axual_application_deployment.kc_sasl.id
  state                  = "STOPPED"
  depends_on             = [axual_application_access_grant_approval.kc_sasl, axual_application_credential.kc_sasl]
}
