resource "axual_application_deployment" "kc_tls" {
  application    = axual_application.kc_tls.id
  environment    = axual_environment.kc_tls.id
  target_id      = data.axual_connect_cluster.kc_tls.id
  target_version = "1.0.0"
  autostart      = false
  configs = {
    "topics"          = "tf-ds-kc-tls-topic",
    "endpoint"        = "http://http-sink-target.kafka.svc.cluster.local:8080/",
    "key.converter"   = "org.apache.kafka.connect.storage.StringConverter",
    "value.converter" = "org.apache.kafka.connect.storage.StringConverter",
    "tasks.max"       = "1",
  }
}
