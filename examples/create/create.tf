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
}

data "cloudtower_datacenter" "sample_dc" {}

resource "cloudtower_cluster" "sample_cluster" {
  ip       = var.cluster_config["ip"]
  username = var.cluster_config["user"]
  password = var.cluster_config["password"]
}

data "cloudtower_vlan" "vm_vlan" {
  name       = "default"
  type       = "VM"
  cluster_id = cloudtower_cluster.sample_cluster.id
}

data "cloudtower_iso" "ubuntu" {
  name_contains = "ubuntu"
  cluster_id    = cloudtower_cluster.sample_cluster.id
}

data "cloudtower_host" "target_host" {
  management_ip_contains = "17.42"
  cluster_id             = cloudtower_cluster.sample_cluster.id
}

resource "cloudtower_vm" "tf_test" {
  name                = "tf-test"
  description         = "managed by terraform"
  cluster_id          = cloudtower_cluster.sample_cluster.id
  host_id             = data.cloudtower_host.target_host.hosts[0].id
  vcpu                = 4
  memory              = 8 * local.GB
  ha                  = true
  firmware            = "BIOS"
  status              = "STOPPED"
  force_status_change = true

  cd_rom {
    boot   = 1
    iso_id = data.cloudtower_iso.ubuntu.isos[0].id
  }


  disk {
    boot = 2
    bus  = "VIRTIO"
    vm_volume {
      storage_policy = "REPLICA_2_THIN_PROVISION"
      name           = "d1"
      size           = 10 * local.GB
    }
  }

  disk {
    boot = 3
    bus  = "VIRTIO"
    vm_volume {
      storage_policy = "REPLICA_3_THICK_PROVISION"
      name           = "d2"
      size           = 1 * local.GB
    }
  }

  cd_rom {
    boot   = 4
    iso_id = ""
  }

  nic {
    vlan_id = data.cloudtower_vlan.vm_vlan.vlans[0].id
  }
}
