terraform {
  required_providers {
    cloudtower = {
      version = "~> 0.1.7"
      source  = "registry.terraform.io/smartxworks/cloudtower"
    }
  }
}

provider "cloudtower" {
  username          = var.tower_config["user"]
  user_source       = var.tower_config["source"]
  cloudtower_server = var.tower_config["server"]
}


data "cloudtower_cluster" "sample_cluster" {
  name = var.cluster_config["name"]
  # ip       = var.cluster_config["ip"]
  # username = var.cluster_config["user"]
  # password = var.cluster_config["password"]
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
  name = "tf-test-cloned-vm-from-template"
  create_effect {
    is_full_copy        = false
    clone_from_template = data.cloudtower_vm_template.tf_test_template.vm_templates[0].id
  }
}

resource "cloudtower_vm" "tf_test_cloned_vm_modify_nics" {
  name = "tf-test-cloned-vm-from-template-modify-nics"
  nic {
    vlan_id = data.cloudtower_vlan.vm_vlan.vlans[0].id
    enabled = true
  }
  create_effect {
    is_full_copy = false

    clone_from_template = data.cloudtower_vm_template.tf_test_template.vm_templates[0].id
  }
}

resource "cloudtower_vm" "tf_test_cloned_vm_cloud_init" {
  name = "tf-test-cloned-vm-from-template-modify-cloud-init"
  create_effect {
    is_full_copy        = false
    clone_from_template = data.cloudtower_vm_template.tf_test_template.vm_templates[0].id
    cloud_init {
      hostname              = "tf-test-vm-hostname"
      default_user_password = 111111
      networks {
        type       = "IPV4"
        nic_index  = 0
        ip_address = "192.168.11.1"
        netmask    = "0.0.0.0"
      }
    }
  }
}

resource "cloudtower_vm" "tf_test_cloned_vm_modify_disks" {
  name = "tf-test-cloned-vm-from-template-modify-disks"
  cd_rom {
    boot   = 1
    iso_id = ""
  }
  disk {
    boot = 2
    bus  = "VIRTIO"
    vm_volume {
      storage_policy = "REPLICA_2_THIN_PROVISION"
      name           = "d1"
      size           = 20 * 1024 * 1024 * 1024
    }
  }
  create_effect {
    is_full_copy        = false
    clone_from_template = data.cloudtower_vm_template.tf_test_template.vm_templates[0].id
  }
}
