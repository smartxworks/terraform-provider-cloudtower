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
  user_source       = "LOCAL"
  cloudtower_server = var.config["tower_server"]
}

data "cloudtower_cluster" "query_clusters" {
  name_in = var.config["cluster"]
}

data "cloudtower_host" "query_hosts" {
  management_ip_in = var.config["host"]
}

data "cloudtower_content_library_vm_template" "query_templates" {
  name = var.config["template_name"]
}

data "cloudtower_vlan" "query_vlans" {
  name_in = var.config["portgroup"]
}

locals {
  cluster_name_map = {
    for cluster in data.cloudtower_cluster.query_clusters.clusters :
    cluster.name => {
      for k, v in cluster : k => v
    }
  }
  host_ip_map = {
    for host in data.cloudtower_host.query_hosts.hosts :
    host.management_ip => {
      for k, v in host : k => v
    }
  }
  vlan_name_map = { for n, vlans in {
    for vlan in data.cloudtower_vlan.query_vlans.vlans :
    vlan.name => vlan...
    } :
    n => {
      for vlan in vlans :
      vlan.cluster_id => {
        for k, v in vlan : k => v
      }
    }
  }
}

# output "cluster_name_map" {
#   value = local.vlan_name_map
# }

# output "host_ip_map" {
#   value = local.host_ip_map
# }

# output "vlan_name_map" {
#   value = local.vlan_name_map
# }

resource "cloudtower_vm" "vms_create_from_template" {
  count      = length(var.config["vm_ip"])
  cluster_id = local.cluster_name_map[var.config["cluster"][count.index]].id #data.cloudtower_cluster.query_clusters.clusters[0].id
  host_id    = local.host_ip_map[var.config["host"][count.index]].id         #data.cloudtower_host.query_hosts.hosts[0].id
  name       = var.config["vm_name"][count.index]
  memory     = var.config["memory"][count.index] * local.MB
  vcpu       = var.config["vcpu"][count.index]
  create_effect {
    is_full_copy                        = false
    clone_from_content_library_template = data.cloudtower_content_library_vm_template.query_templates.content_library_vm_templates[0].id
    cloud_init {
      hostname              = var.config["host_name"][count.index]
      nameservers           = var.config["dns"][count.index]
      default_user_password = 111111
      networks {
        type       = "IPV4"
        nic_index  = 0
        ip_address = var.config["vm_ip"][count.index]
        netmask    = cidrnetmask("${var.config["vm_ip"][count.index]}/${var.config["cidr"][count.index]}")
        routes {
          gateway = var.config["default_gw"][count.index]
          netmask = "0.0.0.0"
          network = "0.0.0.0"
        }
      }
    }
  }
  dynamic "disk" {
    for_each = data.cloudtower_content_library_vm_template.query_templates.content_library_vm_templates[0].vm_templates[0].disks
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
  dynamic "disk" {
    for_each = var.config["extra_disks"][count.index]
    content {
      boot = length(data.cloudtower_content_library_vm_template.query_templates.content_library_vm_templates[0].vm_templates[0].disks) + disk.key
      bus  = "VIRTIO"
      vm_volume {
        storage_policy = "REPLICA_2_THIN_PROVISION"
        name           = "${var.config["vm_name"][count.index]}-${length(data.cloudtower_content_library_vm_template.query_templates.content_library_vm_templates[0].vm_templates[0].disks) + disk.key + 1}"
        size           = disk.value * local.GB
      }
    }
  }

  cd_rom {
    boot = length(data.cloudtower_content_library_vm_template.query_templates.content_library_vm_templates[0].vm_templates[0].disks) + length(var.config["extra_disks"][count.index]) + 1
    iso_id = ""
  }
  nic {
    vlan_id = local.vlan_name_map[var.config["portgroup"][count.index]][local.cluster_name_map[var.config["cluster"][count.index]].id].id #data.cloudtower_vlan.query_vlans.vlans[0].id
  }
}