variable "config" {
  type = object({
    tower_server    = string,
    cluster         = string,
    host            = string,
    portroup        = string,
    default_gateway = string,
    template_name   = string,
    host_name       = list(string),
    vm_name         = list(string),
    memory          = list(number),
    vcpu            = list(number),
    cidr            = number,
    vm_ip           = list(string),
    dns             = list(list(string)),
    extra_disks = list(list(object({
      name           = string,
      size           = number,
      bus            = string,
      storage_policy = string
    })))
  })
  validation {
    condition     = length(var.config.host_name) == length(var.config.vm_name) && length(var.config.host_name) == length(var.config.memory) && length(var.config.host_name) == length(var.config.vcpu) && length(var.config.host_name) == length(var.config.vm_ip) && length(var.config.host_name) == length(var.config.dns) && length(var.config.host_name) == length(var.config.extra_disks)
    error_message = "Host_name, vm_name, memory, vcpu, vm_ip, dns, extra_disks must have the same length."
  }
}
