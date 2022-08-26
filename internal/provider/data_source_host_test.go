package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestDataSourceHost(t *testing.T) {
	host_ip := os.Getenv("TF_VAR_HOST_IP")
	rnd := generateRandomResourceName()
	datasource := "data.cloudtower_host." + rnd
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceHostPrecheck(t)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testDataSourceHostConfig(rnd, host_ip),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(datasource, "hosts.0.management_ip", host_ip),
					resource.TestCheckResourceAttrSet(datasource, "hosts.0.id"),
					resource.TestCheckResourceAttr(datasource, "hosts.#", "1"),
				),
			},
		},
	})
}

func testAccDataSourceHostPrecheck(t *testing.T) {
	if os.Getenv("TF_VAR_HOST_IP") == "" {
		t.Skip("set TF_VAR_HOST_IP to run data source cluster acceptance test")
	}
}

func testDataSourceHostConfig(resource_name string, ip string) string {
	return fmt.Sprintf(`
	data "cloudtower_host" "%[2]s" {
		management_ip = "%[1]s"
	}
	`, ip, resource_name)
}
