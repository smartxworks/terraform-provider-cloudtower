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


