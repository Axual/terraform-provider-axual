resource "axual_application" "tf_test_app_type_flinksql" {
  name             = "tf-test app type flinksql"
  application_type = "FLINK_SQL"
  short_name       = "tf_test_app_type_flinksql"
  application_id   = "tf.test.app.type.flinksql"
  owners           = data.axual_group.test_group.id
  visibility       = "Public"
  description      = "Test Application for FLINK_SQL type"
}
