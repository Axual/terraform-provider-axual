package ApplicationDeploymentResource

//func TestApplicationDeploymentResource(t *testing.T) {
//	resource.Test(t, resource.TestCase{
//		ProtoV6ProviderFactories: GetProviderConfig(t).ProtoV6ProviderFactories,
//		ExternalProviders:        GetProviderConfig(t).ExternalProviders,
//
//		Steps: []resource.TestStep{
//			{
//				Config: GetProvider() + GetFile("axual_application_deployment_initial.tf"),
//				Check: resource.ComposeTestCheckFunc(
//					resource.TestCheckResourceAttr("axual_application_deployment.connector_axual_application_deployment", "name", "tf-test-app"),
//					resource.TestCheckResourceAttrPair("axual_application_deployment.connector_axual_application_deployment", "environment", "axual_environment.tf-test-env", "id"),
//				),
//			},
//			//{
//			//	Config: GetProvider() + GetFile("axual_application_updated.tf"),
//			//	Check: resource.ComposeTestCheckFunc(
//			//		resource.TestCheckResourceAttr("axual_application.tf-test-app", "name", "tf-test-app1"),
//			//	),
//			//},
//			//{
//			//	ResourceName:      "axual_application.tf-test-app",
//			//	ImportState:       true,
//			//	ImportStateVerify: true,
//			//},
//			{
//				// To ensure cleanup if one of the test cases had an error
//				Destroy: true,
//				Config:  GetProvider() + GetFile("axual_application_deployment_initial.tf"),
//			},
//		},
//	})
//}
