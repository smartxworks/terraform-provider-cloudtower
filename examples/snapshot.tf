terraform {
  required_providers {
    cloudtower = {
      version = "~> 0.1.4"
      source  = "registry.terraform.io/smartx/cloudtower"
    }
  }
}

provider "cloudtower" {
  username          = "root"
  user_source       = "LOCAL"
  cloudtower_server = "yinsw-terraform.dev-cloudtower.smartx.com"
}

# data "cloudtower_datacenter" "sample_dc" {}

resource "cloudtower_cluster" "sample_cluster" {
  ip            = "192.168.31.156"
  username      = "root"
  password      = "111111"
#  datacenter_id = data.cloudtower_datacenter.sample_dc.datacenters[0].id
}

data "cloudtower_vlan" "vm_vlan" {
  name       = "default"
  type       = "VM"
  cluster_id = cloudtower_cluster.sample_cluster.id
}

# data "cloudtower_iso" "ubuntu" {
#   name_contains = "ubuntu"
#   cluster_id    = cloudtower_cluster.sample_cluster.id
# }

data "cloudtower_host" "target_host" {
  management_ip_contains = "31.156"
  cluster_id             = cloudtower_cluster.sample_cluster.id
}

resource "cloudtower_vm" "tf_test" {
  name                = "tf-test"
  description         = "managed by terraform"
  cluster_id          = cloudtower_cluster.sample_cluster.id
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

resource "cloudtower_vm_snapshot" "tf_test_snapshot" {
  name        = "tf-test-snapshot"
  vm_id       = cloudtower_vm.tf_test.id
}

resource "cloudtower_vm" "tf_test_vm_from_snapshot" {
  name         = "tf-test-vm-from-snapshot"
  cluster_id   = cloudtower_cluster.sample_cluster.id
  rebuild_from = cloudtower_vm_snapshot.tf_test_snapshot.id
  ha           = false
  memory       = 2 * 1024 * 1024 * 1024
}