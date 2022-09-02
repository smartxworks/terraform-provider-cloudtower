package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceCluster(t *testing.T) {
	cluster_name := os.Getenv("TF_VAR_CLUSTER_NAME")
	rnd := generateRandomResourceName()
	datasource := "data.cloudtower_cluster." + rnd
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceClusterPrecheck(t)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testDataSourceClusterConfig(rnd, cluster_name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(datasource, "clusters.0.name", cluster_name),
					resource.TestCheckResourceAttrSet(datasource, "clusters.0.id"),
					resource.TestCheckResourceAttr(datasource, "clusters.#", "1"),
				),
			},
		},
	})
}

func testAccDataSourceClusterPrecheck(t *testing.T) {
	if os.Getenv("TF_VAR_CLUSTER_NAME") == "" {
		t.Skip("set TF_VAR_CLUSTER_NAME to run data source cluster acceptance test")
	}
}

func testDataSourceClusterConfig(resource_name string, cluster_name string) string {
	return fmt.Sprintf(`
	data "cloudtower_cluster" "%[2]s" {
		name = "%[1]s"
	}
	`, cluster_name, resource_name)
}
