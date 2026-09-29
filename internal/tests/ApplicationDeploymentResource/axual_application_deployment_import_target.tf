# Declares the resource address for an import-only test step (TestApplicationDeploymentImportNotFound).
# The values here are never used to create anything - the step only imports into this address.
resource "axual_application_deployment" "import_target" {
  environment = "dummy"
  application = "dummy"
}
