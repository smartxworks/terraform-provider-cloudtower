terraform {
  required_providers {
    cloudtower = {
      version = "~> 0.1.4"
      source  = "registry.terraform.io/smartx/cloudtower"
    }
  }
}

provider "cloudtower" {
  username          = var.tower_config["user"]
  user_source       = var.tower_config["source"]
  cloudtower_server = var.tower_config["server"]
}


resource "cloudtower_cluster" "sample_cluster" {
  ip       = var.cluster_config["ip"]
  username = var.cluster_config["user"]
  password = var.cluster_config["password"]
}

data "cloudtower_vm_template" "tf_test_template" {
  cluster_id = cloudtower_cluster.sample_cluster.id
  name = var.vm_template_with_cloud_init
}

data "cloudtower_vlan" "vm_vlan" {
  name       = "default"
  type       = "VM"
  cluster_id = cloudtower_cluster.sample_cluster.id
}


resource "cloudtower_vm" "tf_test_cloned_vm" {
  name       = "tf-test-cloned-vm-from-template"
  is_full_copy = false
  create_effect {
    clone_from_template = data.cloudtower_vm_template.tf_test_template.vm_templates[0].id
  }
}

resource "cloudtower_vm" "tf_test_cloned_vm_modify_nics" {
  name       = "tf-test-cloned-vm-from-template-modify-nics"
  is_full_copy = false
  nic {
    vlan_id = data.cloudtower_vlan.vm_vlan.vlans[0].id
    enabled = true
  }
  create_effect {
    clone_from_template = data.cloudtower_vm_template.tf_test_template.vm_templates[0].id
  }
}

resource "cloudtower_vm" "tf_test_cloned_vm_cloud_init" {
  name = "tf-test-cloned-vm-from-template-modify-cloud-init"
  is_full_copy = false
  cloud_init  {
    hostname = "tf-test-vm-hostname"
    default_user_password = 111111
    networks {
      type = "IPV4"
      nic_index = 0
      ip_address = "192.168.11.1"
      netmask = "0.0.0.0"
    }
  }
  create_effect {
    clone_from_template = data.cloudtower_vm_template.tf_test_template.vm_templates[0].id
  }
}



//FIXME: not work
# resource "cloudtower_vm" "tf_test_cloned_vm_modify_disks" {
#   name = "tf-test-cloned-vm-from-template-modify-disks"
#   is_full_copy = false
#   cd_rom {
#     boot   = 2
#     iso_id = ""
#   }
#   disk {
#     boot = 1
#     bus  = "VIRTIO"
#     vm_volume {
#       storage_policy = "REPLICA_2_THIN_PROVISION"
#       name           = "d1"
#       size           = 20 * 1024 * 1024 * 1024
#     }
#   }
#   create_effect {
#     clone_from_template = "cl3kix921w74l0921sci4fxth"
#   }
# }
