resource "axual_application" "tf_test_app_type_source" {
  name              = "tf-test app type source"
  application_type  = "Connector"
  short_name        = "tf_test_app_type_source"
  application_id    = "tf.test.app.type.source"
  owners            = data.axual_group.test_group.id
  type              = "SOURCE"
  application_class = "io.confluent.connect.jdbc.JdbcSourceConnector"
  visibility        = "Public"
  description       = "Test Application for SOURCE connector type"
}
