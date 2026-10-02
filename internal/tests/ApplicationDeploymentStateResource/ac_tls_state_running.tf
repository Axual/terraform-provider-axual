resource "axual_application_deployment_state" "ac_tls" {
  application_deployment = axual_application_deployment.ac_tls.id
  state                  = "RUNNING"
  depends_on             = [axual_application_access_grant_approval.ac_tls, axual_application_principal.ac_tls]
}
