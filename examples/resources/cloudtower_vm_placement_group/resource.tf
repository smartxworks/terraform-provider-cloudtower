resource "cloudtower_vm_placement_group" "database" {
  name        = "database"
  description = "Keep database VMs on dedicated hosts and spread them apart"
  cluster_id  = "cluster-id"

  required_host_rule {
    enabled   = true
    placement = "ON_SELECTED_HOSTS"
    host_ids = [
      "host-id-1",
      "host-id-2",
    ]
  }

  preferred_host_rule {
    enabled = false
  }

  vm_affinity {
    enabled     = true
    enforcement = "REQUIRED"
    placement   = "DIFFERENT_HOSTS"
  }
}
