package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccVmCreateBlank(t *testing.T) {
	rnd_suffix := generateRandomResourceName()
	name := "cloudtower_vm." + rnd_suffix
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVmBasicPrecheck(t)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVmBasicConfig(rnd_suffix, os.Getenv("TF_VAR_CLUSTER_NAME"), os.Getenv("TF_VAR_HOST_IP")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "terraform-test-vm-"+rnd_suffix),
					resource.TestCheckResourceAttr(name, "cpu_cores", "1"),
					resource.TestCheckResourceAttr(name, "cpu_sockets", "4"),
				),
			},
		},
	})
}

func TestAccVmCreateFromTemplate(t *testing.T) {
	rnd_suffix := generateRandomResourceName()
	name := "cloudtower_vm." + rnd_suffix
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVmBasicPrecheck(t)
			testAccResourceVmCreateFromTemplatePrecheck(t)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVmCreateFromTemplateConfig(rnd_suffix, os.Getenv("TF_VAR_CLUSTER_NAME"), os.Getenv("TF_VAR_CLUSTER_NAME"), os.Getenv("TF_VAR_HOST_IP")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "", ""),
				),
			},
		},
	},
	)
}

func TestAccVmClone(t *testing.T) {
	rnd_suffix := generateRandomResourceName()
	source_rnd_suffix := generateRandomResourceName()
	name := "cloudtower_vm." + rnd_suffix
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVmBasicPrecheck(t)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVmCloneConfig(rnd_suffix, source_rnd_suffix, os.Getenv("TF_VAR_CLUSTER_NAME"), os.Getenv("TF_VAR_HOST_IP")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "terraform-test-vm-clone-"+rnd_suffix),
					resource.TestCheckResourceAttr(name, "cpu_cores", "1"),
					resource.TestCheckResourceAttr(name, "cpu_sockets", "4")),
			},
		},
	},
	)
}

func testAccResourceVmBasicPrecheck(t *testing.T) {
	if os.Getenv("TF_VAR_CLUSTER_NAME") == "" {
		t.Skip("set TF_VAR_CLUSTER_NAME to run vm resource acceptance tests")
	}
	if os.Getenv("TF_VAR_HOST_IP") == "" {
		t.Skip("set TF_VAR_HOST_IP to run vm resource acceptance tests")
	}
}

func testAccResourceVmCreateFromTemplatePrecheck(t *testing.T) {
	if os.Getenv("TF_VAR_TEMPLATE_NAME") == "" {
		t.Skip("set TF_VAR_TEMPLATE_NAME to run vm resource create from template acceptance tests")
	}
}

func testAccResourceVmCreateFromTemplateCloudInitPrecheck(t *testing.T) {
	if os.Getenv("TF_VAR_TEMPLATE_CLOUD_INIT_NAME") == "" {
		t.Skip("set TF_VAR_TEMPLATE_CLOUD_INIT_NAME to run vm resource create from template with cloud init acceptance tests")
	}
	if os.Getenv("TF_VAR_CLOUD_INIT_IP") == "" {
		t.Skip("set TF_VAR_CLOUD_INIT_IP to run vm resource create from template with cloud init acceptance tests")
	}
	if os.Getenv("TF_VAR_CLOUD_INIT_GATEWAY") == "" {
		t.Skip("set TF_VAR_CLOUD_INIT_GATEWAY to run vm resource create from template with cloud init acceptance tests")
	}
	if os.Getenv("TF_VAR_CLOUD_INIT_SUBNET") == "" {
		t.Skip("set TF_VAR_CLOUD_INIT_SUBNET to run vm resource create from template with cloud init acceptance tests")
	}
}

func testAccVmBasicConfig(vm_name string, cluster_name string, host_ip string) string {
	data_source_cluster_rnd := generateRandomResourceName()
	data_source_host_rnd := generateRandomResourceName()
	return fmt.Sprintf(`
	%[1]s
	%[2]s
	resource "cloudtower_vm" "%[3]s" {
		name                = "terraform-test-vm-%[3]s"
		description         = "managed by terraform"
		cluster_id          = data.cloudtower_cluster.%[4]s.clusters[0].id
		host_id             = data.cloudtower_host.%[5]s.hosts[0].id
		vcpu                = 4
		memory              = 8 * 1024 * 1024 * 1024
		ha                  = true
		firmware            = "BIOS"
		status              = "STOPPED"
		force_status_change = true
	
		cd_rom {
			boot   = 1
			iso_id = ""
		}
	}
	`, testDataSourceClusterConfig(data_source_cluster_rnd, cluster_name), testDataSourceHostConfig(data_source_host_rnd, host_ip), vm_name, data_source_cluster_rnd, data_source_host_rnd)
}

func testAccVmCreateFromTemplateConfig(vm_name string, template_name string, cluster_name string, host_ip string) string {
	data_source_template_rnd := generateRandomResourceName()
	return fmt.Sprintf(`
	%[1]s
	resource "cloudtower_vm" "%[2]s" {
		name = "%[2]s"
		create_effect {
			is_full_copy        = false
			clone_from_template = data.cloudtower_vm_template.%[3]s.vm_templates[0].id
		}
	}
	`, testDataSourceVmTemplateConfig(data_source_template_rnd, template_name), vm_name, data_source_template_rnd)
}

func testAccVmCloneConfig(vm_name string, source_vm_name string, cluster_name string, host_ip string) string {
	return fmt.Sprintf(`
	%[1]s
	resource "cloudtower_vm" "%[2]s" {
		name = "terraform-test-vm-clone-%[2]s"
		cluster_id = cloudtower_vm.%[3]s.cluster_id
		create_effect {
			clone_from_vm = cloudtower_vm.%[3]s.id
		}
	}
	`, testAccVmBasicConfig(source_vm_name, cluster_name, host_ip), vm_name, source_vm_name, cluster_name, host_ip)
}
