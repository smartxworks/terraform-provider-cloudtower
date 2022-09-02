package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVmTemplate(t *testing.T) {
	template_name := os.Getenv("TF_VAR_TEMPLATE_NAME")
	rnd_suffix := generateRandomResourceName()
	datasource := "data.cloudtower_vm_template." + rnd_suffix
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVmTemplatePrecheck(t)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testDataSourceVmTemplateConfig(template_name, rnd_suffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(datasource, "vm_templates.0.name", template_name),
					resource.TestCheckResourceAttrSet(datasource, "vm_templates.0.id"),
					resource.TestCheckResourceAttr(datasource, "vm_templates.#", "1"),
				),
			},
		},
	})
}

func testAccDataSourceVmTemplatePrecheck(t *testing.T) {
	if os.Getenv("TF_VAR_TEMPLATE_NAME") == "" {
		t.Skip("set TF_VAR_TEMPLATE_NAME to run data source vm template acceptance test")
	}
}

func testDataSourceVmTemplateConfig(resource_name string, template_name string) string {
	return fmt.Sprintf(`
	data "cloudtower_vm_template" "%[2]s" {
		name = "%[1]s"
	}
	`, template_name, resource_name)
}
