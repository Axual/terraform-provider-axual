resource "axual_application" "tf_test_app_type_sap" {
  name             = "tf-test app type sap"
  application_type = "Custom"
  short_name       = "tf_test_app_type_sap"
  application_id   = "tf.test.app.type.sap"
  owners           = data.axual_group.test_group.id
  type             = "SAP"
  visibility       = "Public"
  description      = "Test Application for SAP type"
}
