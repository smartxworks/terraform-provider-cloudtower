---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtower_vm Resource - cloudtower-terraform-provider"
subcategory: ""
description: |-
  CloudTower vm resource.
---

# cloudtower_vm (Resource)

CloudTower vm resource.

<!-- schema generated by tfplugindocs -->

## Schema

### Required

- `name` (String) VM's name

### Optional

- `cd_rom` (Block List) VM's CD-ROM (see [below for nested schema](#nestedblock--cd_rom))
- `cluster_id` (String) VM's cluster id
- `cpu_cores` (Number) VM's cpu cores
- `cpu_sockets` (Number) VM's cpu sockets
- `create_effect` (Block List, Max: 1) (see [below for nested schema](#nestedblock--create_effect))
- `description` (String) VM's description
- `disk` (Block List) VM's virtual disks (see [below for nested schema](#nestedblock--disk))
- `firmware` (String) VM's firmware, forcenew as it isn't able to modify after create, must be one of 'BIOS', 'UEFI'
- `folder_id` (String) VM's folder id
- `force_status_change` (Boolean) force VM's status change, will apply when power off or restart
- `guest_os_type` (String) VM's guest OS type
- `ha` (Boolean) whether VM is HA or not
- `host_id` (String) VM's host id
- `memory` (Number) VM's memory, in the unit of byte, must be a multiple of 512MB, long value, ignore the decimal point
- `nic` (Block List) VM's virtual nic (see [below for nested schema](#nestedblock--nic))
- `rollback_to` (String) Vm is going to rollback to target snapshot
- `status` (String) VM's status
- `vcpu` (Number) VM's vcpu

### Read-Only

- `id` (String) VM's id

<a id="nestedblock--cd_rom"></a>

### Nested Schema for `cd_rom`

Required:

- `boot` (Number) VM CD-ROM's boot order
- `iso_id` (String) mount an ISO to a VM CD-ROM by specific it's id

Read-Only:

- `id` (String) VM CD-ROM's id

<a id="nestedblock--create_effect"></a>

### Nested Schema for `create_effect`

Optional:

- `clone_from_content_library_template` (String) Id of source content library VM template to be cloned
- `clone_from_template` (String) Id of source VM template to be cloned
- `clone_from_vm` (String) Id of source vm from created vm to be cloned from
- `cloud_init` (Block List, Max: 1) Set up cloud-init config when create vm from template (see [below for nested schema](#nestedblock--create_effect--cloud_init))
- `is_full_copy` (Boolean) If the vm is full copy from template or not
- `rebuild_from_snapshot` (String) Id of snapshot for created vm to be rebuilt from

<a id="nestedblock--create_effect--cloud_init"></a>

### Nested Schema for `create_effect.cloud_init`

Optional:

- `default_user_password` (String) Password of default user
- `hostname` (String) hostname
- `nameservers` (List of String) Name server address list. At most 3 name servers are allowed.
- `networks` (Block List, Max: 1) Network configuration list. (see [below for nested schema](#nestedblock--create_effect--cloud_init--networks))
- `public_keys` (List of String) Add a list of public keys for the cloud-init default user.At most 10 public keys can be added to the list.
- `user_data` (String) User-provided cloud-init user-data field. Base64 encoding is not supported. Size limit: 32KiB.

<a id="nestedblock--create_effect--cloud_init--networks"></a>

### Nested Schema for `create_effect.cloud_init.networks`

Required:

- `nic_index` (Number) Index of VM NICs. The index starts at 0, which refers to the first NIC.At most 16 NICs are supported, so the index range is [0, 15].
- `type` (String) Network type. Allowed enum values are ipv4, ipv4_dhcp.

Optional:

- `ip_address` (String) IPv4 address. This field is only used when type is not set to ipv4_dhcp.
- `netmask` (String) Netmask. This field is only used when type is not set to ipv4_dhcp.
- `routes` (Block List, Max: 1) Static route list (see [below for nested schema](#nestedblock--create_effect--cloud_init--networks--routes))

<a id="nestedblock--create_effect--cloud_init--networks--routes"></a>

### Nested Schema for `create_effect.cloud_init.networks.routes`

Optional:

- `gateway` (String) Gateway to access the static route address.
- `netmask` (String) Netmask of the network
- `network` (String) Static route network address. If set to 0.0.0.0, then first use the user settings to configure the default route.

<a id="nestedblock--disk"></a>

### Nested Schema for `disk`

Required:

- `boot` (Number) VM disk's boot order
- `bus` (String) VM disk's bus

Optional:

- `vm_volume` (Block List, Max: 1) create a new VM volume and use it as a VM disk (see [below for nested schema](#nestedblock--disk--vm_volume))
- `vm_volume_id` (String) use an existing VM volume as a VM disk, by specific it's id

Read-Only:

- `id` (String) the VM disk's id

<a id="nestedblock--disk--vm_volume"></a>

### Nested Schema for `disk.vm_volume`

Required:

- `name` (String) the new VM volume's name
- `size` (Number) the new VM volume's size, in the unit of byte
- `storage_policy` (String) the new VM volume's storage policy

Optional:

- `origin_path` (String) the VM volume will create base on the path

Read-Only:

- `id` (String) the VM volume's id
- `path` (String) the VM volume's iscsi LUN path

<a id="nestedblock--nic"></a>

### Nested Schema for `nic`

Required:

- `vlan_id` (String) specific the vlan's id the VM nic will use

Optional:

- `enabled` (Boolean) whether the VM nic is enabled
- `gateway` (String) VM nic's gateway
- `ip_address` (String) VM nic's IP address
- `mac_address` (String) VM nic's mac address
- `mirror` (Boolean) whether the VM nic use mirror mode
- `model` (String) VM nic's model
- `subnet_mask` (String) VM nic's subnet mask

Read-Only:

- `id` (String) VM nic's id
- `idx` (Number) VM nic's index

## Usage

### create a blank vm

This will create a blank vm on speicifed cluster.

```hcl
datasource "cloudtower_cluster" "sample_cluster" {
  name       = "sample_cluster"
}
resource "cloudtower_vm" "sample_vm" {
  name = "sample_vm"
  cluster_id = datasource.cloudtower_cluster.sample_cluster.clusters[0].id
  cpu_cores = 1
  cpu_sockets = 1
  memory = 1073741824
  vcpu = 1
  nic {
    vlan_id = "vlan_id"
  }
  disk {
    boot = 1
    bus = "scsi"
    vm_volume {
      name = "sample_vm_volume"
      size = 1073741824
      storage_policy = "REPLICA_2_THICK_PROVISION"
    }
  }
}
```

### clone from another vm

use create_effect.clone_from_vm can clone a new vm from another vm

```hcl
resource "cloudtower_vm" "tf_test_cloned-vm" {
  name       = "tf-test-cloned-vm"
  cluster_id = cloudtower_cluster.sample_cluster.id
  create_effect {
    clone_from_vm = cloudtower_vm.tf_test.id
  }
}
```

### rebuild from snapshot

use create_effect.rebuild_from_snapshot can rebuild a vm from a snapshot

```hcl
datasource "cloudtower_cluster" "sample_cluster" {
  name       = "sample_cluster"
}

datasource "cloudtower_vm_snapshot" "sample_snapshot" {
  name  = "sample-snapshot"
}

resource "cloudtower_vm" "sample_vm_from_snapshot" {
  name       = "sample-vm-from_snapshot"
  cluster_id = data.cloudtower_cluster.sample_cluster.clusters[0].id
  create_effect {
    rebuild_from_snapshot = data.cloudtower_vm_snapshot.sample_snapshot.snapshots[0].id
  }
}
```

### clone from content library template

Create a vm from content library template, which is support to be used between different cluster.

Use effect.clone_from_content_library_template to clone a vm from content library template.

```hcl
datasource "cloudtower_cluster" "sample_cluster" {
  name       = "sample_cluster"
}

data "cloudtower_content_library_vm_template" "sample_template" {
  name       = "sample_template"
}

resource "cloudtower_vm" "sample_content_library_vm" {
  name = "sample-content-library-template-vm"
  create_effect {
    is_full_copy        = false
    clone_from_content_library_template = data.cloudtower_content_library_vm_template.sample_template.vm_templates[0].id
  }
}
```

### clone from content libarary template with modification

Clone from content libarary template support modificiation from the origin template.

#### computing resource

cpu_cores, cpu_sockets, memory, vcpu can be modified from template.

```hcl
datasource "cloudtower_cluster" "sample_cluster" {
  name       = "sample_cluster"
}

data "cloudtower_content_library_vm_template" "sample_template" {
  name       = "sample_template"
}

resource "cloudtower_vm" "sample_content_library_vm" {
  name = "sample-content-library-template-vm"
  create_effect {
    is_full_copy        = false
    clone_from_content_library_template = data.cloudtower_content_library_vm_template.sample_template.vm_templates[0].id
  }
  vcpu = 12
  cpu_cores = 3
  cpu_sockets = 4
  memory = 1073741824
}
```

#### disk

As use terraform to manage vm, if you want to modify disk from template, you should declare all disk in disk block, otherwise it will leave part of disk under management while others not, which may cause many side effect.

If you want fully control the disks, declared any disk in disk block, it will override all disk from template, so you should declare all disk in disk block.

To use disk from template with disk modification, you should configure origin_path of disk to make disk create base on template disk.

```hcl
datasource "cloudtower_cluster" "sample_cluster" {
  name       = "sample_cluster"
}

data "cloudtower_content_library_vm_template" "sample_template" {
  name       = "sample_template"
}

resource "cloudtower_vm" "sample_content_library_vm" {
  cluster_id = data.cloudtower_cluster.sample_cluster.clusters[0].id
  name       = "sample-content-library-template-vm"
  create_effect {
    is_full_copy                        = false
    clone_from_content_library_template = data.cloudtower_content_library_vm_template.sample_template.vm_templates[0].id
  }
  dynamic "disk" {
    for_each = data.cloudtower_content_library_vm_template.sample_template.vm_templates[0].disks
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
  disk {
    boot = length(data.cloudtower_content_library_vm_template.query_templates.content_library_vm_templates[0].vm_templates[0].disks) + disk.key
    bus  = "VIRTIO"
    vm_volume {
      storage_policy = "REPLICA_2_THIN_PROVISION"
      name           = "new-volume"
      size           = disk.value * local.GB
    }
  }

  cd_rom {
    boot   = length(data.cloudtower_content_library_vm_template.query_templates.content_library_vm_templates[0].vm_templates[0].disks) + 2
    iso_id = ""
  }
}
```

#### nic

If nic is not configured, it will use default vlan of cluster for every nic.

The same to disk, if you want to modify nic from template, you should declare all nic in nic block.

```
datasource "cloudtower_cluster" "sample_cluster" {
  name       = "sample_cluster"
}

data "cloudtower_content_library_vm_template" "sample_template" {
  name       = "sample_template"
}

data "cloudtower_vlan" "sample_vlan" {
  name       = "default"
  type       = "VM"
  cluster_id = data.data.cloudtower_cluster.sample_cluster.clusters[0].clusters[0].id
}

resource "cloudtower_vm" "sample_content_library_vm" {
  cluster_id = data.cloudtower_cluster.sample_cluster.clusters[0].id
  name       = "sample-content-library-template-vm"
  create_effect {
    is_full_copy                        = false
    clone_from_content_library_template = data.cloudtower_content_library_vm_template.sample_template.vm_templates[0].id
  }

  cd_rom {
    boot   = 0
    iso_id = ""
  }

  nic {
    vlan_id = data.cloudtower_vlan.sample_vlan.vlans[0].id
  }
}
```

### clone from content library template with cloud init

If template support cloud-init, pass cloudinit config can set up vm's starting ip, hostname, default user password, etc.

Configure them in create_effect.cloud_init block.

```hcl
datasource "cloudtower_cluster" "sample_cluster" {
  name       = "sample_cluster"
}
resource "cloudtower_vm" "sample_cloud_init_vm" {
  name = "tf-test-cloned-vm-from-template-modify-cloud-init"
  create_effect {
    is_full_copy        = false
    clone_from_template = data.cloudtower_vm_template.tf_test_template.vm_templates[0].id
    cloud_init {
      hostname              = "tf-test-vm-hostname"
      default_user_password = 111111
      networks {
        type       = "IPV4"
        nic_index  = 0
        ip_address = "192.168.11.2"
        netmask    = "255.255.255.0",
        gateway    = "192.168.11.1"
      }
    }
  }
}
```

### clone from template

Deprecated, content library template is preferred, but vm template is also supported, usage is close to content library template.

use create_effect.clone_from_template can clone a new vm from template.

```hcl
datasource "cloudtower_cluster" "sample_cluster" {
  name       = "sample_cluster"
}

data "cloudtower_vm_template" "sample_template" {
  cluster_id = data.cloudtower_cluster.sample_cluster.clusters[0].id
  name       = var.vm_template_with_cloud_init
}

resource "cloudtower_vm" "tf_test_cloned_vm" {
  name = "tf-test-cloned-vm-from-template"
  create_effect {
    is_full_copy        = false
    clone_from_template = data.cloudtower_vm_template.sample_template.vm_templates[0].id
  }
}
```
