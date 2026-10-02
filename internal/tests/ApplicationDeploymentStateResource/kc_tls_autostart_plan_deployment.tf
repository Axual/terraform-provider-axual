# autostart = true (the default) on a Kafka Connect target, planned against an existing application.
resource "axual_application_deployment" "tls_plan" {
  application    = axual_application.tls_plan.id
  environment    = axual_environment.tls_plan.id
  target_id      = local.connect_cluster_mtls_id
  target_version = "1.0.0"
  configs = {
    "topics"   = "tf-ds-tls-plan-topic",
    "endpoint" = "http://http-sink-target.kafka.svc.cluster.local:8080/",
  }
}
