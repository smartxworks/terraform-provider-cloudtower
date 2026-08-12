terraform {
  required_providers {
    cloudtower = {
      source = "registry.terraform.io/smartxworks/cloudtower"
    }
  }
}

provider "cloudtower" {
  username          = "terraform"
  user_source       = "LOCAL"
  cloudtower_server = "192.68.1.1"
}

data "cloudtower_cluster" "target_cluster" {
  name = var.cluster_name
}

data "cloudtower_vds" "test" {
  name       = var.vds_name
  cluster_id = data.cloudtower_cluster.target_cluster.clusters[0].id
}

resource "cloudtower_vm_vlan" "multi_vlan_ids" {
  for_each = var.vlans

  name        = each.value.name
  vds_id      = data.cloudtower_vds.test.vds[0].id
  mode_type   = each.value.mode_type
  network_ids = each.value.network_ids
}

resource "cloudtower_vm_vlan" "single_access_vlan" {
  name        = "tf-single-access-vlan"
  vds_id      = data.cloudtower_vds.test.vds[0].id
  mode_type   = "VLAN_ACCESS"
  network_ids = ["100"]
}

resource "cloudtower_vm_vlan" "trunk_vlan" {
  name        = "tf-trunk-vlan"
  vds_id      = data.cloudtower_vds.test.vds[0].id
  mode_type   = "VLAN_TRUNK"
  network_ids = ["100", "200-205"]
}

output "multi_vlan_ids" {
  value = { for k, v in cloudtower_vm_vlan.multi_vlan_ids : k => {
    id          = v.id
    name        = v.name
    local_id    = v.local_id
    mode_type   = v.mode_type
    network_ids = v.network_ids
  } }
}

output "single_access_vlan" {
  value = {
    id          = cloudtower_vm_vlan.single_access_vlan.id
    name        = cloudtower_vm_vlan.single_access_vlan.name
    local_id    = cloudtower_vm_vlan.single_access_vlan.local_id
    mode_type   = cloudtower_vm_vlan.single_access_vlan.mode_type
    network_ids = cloudtower_vm_vlan.single_access_vlan.network_ids
  }
}

output "trunk_vlan" {
  value = {
    id          = cloudtower_vm_vlan.trunk_vlan.id
    name        = cloudtower_vm_vlan.trunk_vlan.name
    local_id    = cloudtower_vm_vlan.trunk_vlan.local_id
    mode_type   = cloudtower_vm_vlan.trunk_vlan.mode_type
    network_ids = cloudtower_vm_vlan.trunk_vlan.network_ids
  }
}
