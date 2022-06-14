terraform {
  required_providers {
    cloudtower = {
      version = "~> 0.1.7"
      source  = "registry.terraform.io/smartxworks/cloudtower"
    }
  }
}

locals {
  GB = 1024 * local.MB
  MB = 1024 * local.KB
  KB = 1024
}

provider "cloudtower" {
  username          = var.tower_config["user"]
  user_source       = var.tower_config["source"]
  cloudtower_server = var.tower_config["server"]
  password          = var.tower_config["password"]
}

data "cloudtower_cluster" "sample_cluster" {
  name = var.cluster_config["name"]
}

data "cloudtower_vm_template" "tf_test_template" {
  cluster_id = data.cloudtower_cluster.sample_cluster.clusters[0].id
  name       = var.vm_template_with_cloud_init
}

data "cloudtower_vlan" "vm_vlan" {
  name       = "default"
  type       = "VM"
  cluster_id = data.cloudtower_cluster.sample_cluster.clusters[0].id
}

resource "cloudtower_vm" "tf_test_cloned_vm" {
  name = "yinsw-tf-test-modify-disk"
  create_effect {
    is_full_copy        = false
    clone_from_template = data.cloudtower_vm_template.tf_test_template.vm_templates[0].id
  }
  disk {
    boot = 0
    bus  = data.cloudtower_vm_template.tf_test_template.vm_templates[0].disks[0].bus
    vm_volume {
      storage_policy = data.cloudtower_vm_template.tf_test_template.vm_templates[0].disks[0].storage_policy
      name           = "tf-test-cloned-vm-from-template-1"
      size           = data.cloudtower_vm_template.tf_test_template.vm_templates[0].disks[0].size
      origin_path    = data.cloudtower_vm_template.tf_test_template.vm_templates[0].disks[0].path
    }
  }
  cd_rom {
    boot   = 1
    iso_id = data.cloudtower_vm_template.tf_test_template.vm_templates[0].cd_roms[0].elf_image_id
  }
}