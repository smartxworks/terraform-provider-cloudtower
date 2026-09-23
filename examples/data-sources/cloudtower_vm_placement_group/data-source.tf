data "cloudtower_vm_placement_group" "database" {
  cluster_id    = "cluster-id"
  name_contains = "database"
  enabled       = true
}
