resource "axual_application" "tf_test_app_type_kafka_streams" {
  name             = "tf-test app type kafka streams"
  application_type = "Custom"
  short_name       = "tf_test_app_type_kafka_streams"
  application_id   = "tf.test.app.type.kafka.streams"
  owners           = data.axual_group.test_group.id
  type             = "Kafka Streams"
  visibility       = "Public"
  description      = "Test Application for Kafka Streams type"
}
