variable "tower_config" {
  description = "config for tower"
  type = object({
    user   = string
    server = string
    source = string
  })
}

variable "cluster_config" {
  description = "config for cluster"
  type = object({
    user     = string
    password = string
    ip       = string
  })
}

variable "cloudinit_config" {
  type = list(object({
    ip      = string
    netmask = string
    gateway = string
    dns     = list(string)
  }))
}