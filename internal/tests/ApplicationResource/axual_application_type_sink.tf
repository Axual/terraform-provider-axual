resource "axual_application" "tf_test_app_type_sink" {
  name              = "tf-test app type sink"
  application_type  = "Connector"
  short_name        = "tf_test_app_type_sink"
  application_id    = "tf.test.app.type.sink"
  owners            = data.axual_group.test_group.id
  type              = "SINK"
  application_class = "io.confluent.connect.jdbc.JdbcSinkConnector"
  visibility        = "Public"
  description       = "Test Application for SINK connector type"
}
