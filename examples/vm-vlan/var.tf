variable "cluster_name" {
  description = "cluster name"
  type        = string
}

variable "vds_name" {
  description = "vds name"
  type        = string
}

variable "vlans" {
  description = "vlan map"
  type = map(object({
    name        = string
    mode_type   = string
    network_ids = list(string)
  }))
}
