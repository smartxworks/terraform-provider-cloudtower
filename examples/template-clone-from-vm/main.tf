terraform {
  required_providers {
    cloudtower = {
      version = "~> 0.1.6"
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
}

data "cloudtower_vlan" "vm_vlan" {
  name       = "default"
  type       = "VM"
  cluster_id = data.cloudtower_cluster.sample_cluster.clusters[0].id
}

data "cloudtower_host" "target_host" {
  management_ip_contains = "31.16"
  cluster_id             = data.cloudtower_cluster.sample_cluster.clusters[0].id
}

resource "cloudtower_vm" "tf_test" {
  name                = "tf-test-to-be-cloned-by-vm"
  description         = "managed by terraform"
  cluster_id          = data.cloudtower_cluster.sample_cluster.clusters[0].id
  host_id             = data.cloudtower_host.target_host.hosts[0].id
  vcpu                = 4
  memory              = 4 * 1024 * 1024 * 1024
  ha                  = false
  firmware            = "BIOS"
  status              = "STOPPED"
  force_status_change = true

  cd_rom {
    boot   = 2
    iso_id = ""
  }

  disk {
    boot = 1
    bus  = "VIRTIO"
    vm_volume {
      storage_policy = "REPLICA_2_THIN_PROVISION"
      name           = "d1"
      size           = 20 * 1024 * 1024 * 1024
    }
  }

  nic {
    vlan_id = data.cloudtower_vlan.vm_vlan.vlans[0].id
  }
}

resource "cloudtower_vm_template" "tf_test_template_clone_from_vm" {
  name                 = "tf-test-template-by-cloned-from-vm-1"
  cloud_init_supported = false
  description          = "first tf template"
  src_vm_id            = cloudtower_vm.tf_test.id
}