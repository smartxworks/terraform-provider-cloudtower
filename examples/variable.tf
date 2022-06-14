variable "tower_config" {
  type = object({
    server   = string
    source   = string
    user     = string
    password = string
  })
}

variable "cluster_config" {
  type = object({
    name = string
  })
}

variable "vm_template_with_cloud_init" {
  type        = string
  description = "VM_TEMPLATE_WITH_CLOUD_INIT"
}

variable "vm_config" {
  type = list(object({
    vm_name      = string
    vcpu         = number
    memory       = number
    power_status = string
    portroup     = string
    cloud_init = object({
      hostname = string
      password = string
      ip       = string
      netmask  = string
      gateway  = string
      dns      = list(string)
    })
  }))
}
