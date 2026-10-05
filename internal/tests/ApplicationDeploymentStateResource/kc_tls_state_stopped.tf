resource "axual_application_deployment_state" "kc_tls" {
  application_deployment = axual_application_deployment.kc_tls.id
  state                  = "STOPPED"
  depends_on             = [axual_application_access_grant_approval.kc_tls, axual_application_principal.kc_tls]
}
