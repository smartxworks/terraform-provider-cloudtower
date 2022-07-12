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
  username          = "root"
  password          = "111111"
  cloudtower_server = var.config["tower_server"]
}

data "cloudtower_cluster" "query_clusters" {
  name = var.config["cluster"]
}

data "cloudtower_host" "query_hosts" {
  management_ip = var.config["host"]
}

data "cloudtower_vm_template" "query_templates" {
  cluster_id = data.cloudtower_cluster.query_clusters.clusters[0].id
  name       = var.config["template_name"]
}

data "cloudtower_vlan" "query_vlans" {
  name = var.config["portroup"]
}


resource "cloudtower_vm" "vms_create_from_template" {
  count  = length(var.config["vm_ip"])
  name   = var.config["vm_name"][count.index]
  memory = var.config["memory"][count.index] * local.MB
  vcpu   = var.config["vcpu"][count.index]
  # need to wait tower 2.1.0 release to remove previous two line=
  cpu_cores   = 1
  cpu_sockets = var.config["vcpu"][count.index]
  create_effect {
    is_full_copy        = false
    clone_from_template = data.cloudtower_vm_template.query_templates.vm_templates[0].id
    cloud_init {
      hostname              = var.config["host_name"][count.index]
      nameservers           = var.config["dns"][count.index]
      default_user_password = 111111
      networks {
        type       = "IPV4"
        nic_index  = 0
        ip_address = var.config["vm_ip"][count.index]
        netmask    = cidrnetmask("${var.config["vm_ip"][count.index]}/${var.config["cidr"]}")
        routes {
          gateway = var.config["default_gateway"]
          netmask = "0.0.0.0"
          network = "0.0.0.0"
        }
      }
    }
  }
  dynamic "disk" {
    for_each = data.cloudtower_vm_template.query_templates.vm_templates[0].disks
    content {
      boot = disk.value.boot
      bus  = disk.value.bus
      vm_volume {
        storage_policy = disk.value.storage_policy
        name           = "${var.config["vm_name"][count.index]}-${disk.key + 1}"
        size           = disk.value.size
        origin_path    = disk.value.path
      }
    }
  }
  dynamic "cd_rom" {
    for_each = data.cloudtower_vm_template.query_templates.vm_templates[0].cd_roms
    content {
      boot   = cd_rom.value.boot
      iso_id = lookup(cd_rom.value, "elf_image_id", "")
    }
  }
  dynamic "disk" {
    for_each = var.config["extra_disks"][count.index]
    content {
      boot = length(data.cloudtower_vm_template.query_templates.vm_templates[0].disks) + length(data.cloudtower_vm_template.query_templates.vm_templates[0].cd_roms) + disk.key
      bus  = disk.value.bus
      vm_volume {
        storage_policy = disk.value.storage_policy
        name           = disk.value.name
        size           = disk.value.size * local.GB
      }
    }
  }
  nic {
    vlan_id = data.cloudtower_vlan.query_vlans.vlans[0].id
  }
}