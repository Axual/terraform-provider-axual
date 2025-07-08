package ApplicationDeploymentResource

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"testing"
)

func TestAccApplicationDeploymentResourceImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + applicationDeploymentResourceConfig(),
			},
			{
				ResourceName:      "axual_application_deployment.test",
				ImportState:       true,
				ImportStateId:     "gP99J4llpZ34O7a4/Joleb20X494aV2o1",
				ImportStateVerify: true,
			},
		},
	})
}

func applicationDeploymentResourceConfig() string {
	return fmt.Sprintf(`
		resource "axual_application_deployment" "test" {
			application = "gP99J4llpZ34O7a4"
			environment = "Joleb20X494aV2o1"
			configs = {
				"connector.class" = "io.axual.connect.plugins.oracle.GoldenGateSourceConnector"
			}
		}
	`)
}
