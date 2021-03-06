---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtower_vm_template Data Source - cloudtower-terraform-provider"
subcategory: ""
description: |-
  CloudTower vm template data source.
---

# cloudtower_vm_template (Data Source)

CloudTower vm template data source.



<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `cluster_id` (String) cluster's id of the template
- `name` (String) vm template's name
- `name_contains` (String) filter vm template by its name contains characters

### Read-Only

- `id` (String) The ID of this resource.
- `vm_templates` (List of Object) list of queried vm templates (see [below for nested schema](#nestedatt--vm_templates))

<a id="nestedatt--vm_templates"></a>
### Nested Schema for `vm_templates`

Read-Only:

- `cd_roms` (List of Object) (see [below for nested schema](#nestedobjatt--vm_templates--cd_roms))
- `create_time` (String)
- `disks` (List of Object) (see [below for nested schema](#nestedobjatt--vm_templates--disks))
- `id` (String)
- `name` (String)
- `nics` (List of Object) (see [below for nested schema](#nestedobjatt--vm_templates--nics))

<a id="nestedobjatt--vm_templates--cd_roms"></a>
### Nested Schema for `vm_templates.cd_roms`

Read-Only:

- `boot` (Number)
- `elf_image_id` (String)
- `svt_image_id` (String)


<a id="nestedobjatt--vm_templates--disks"></a>
### Nested Schema for `vm_templates.disks`

Read-Only:

- `boot` (Number)
- `bus` (String)
- `name` (String)
- `path` (String)
- `size` (Number)
- `storage_policy` (String)


<a id="nestedobjatt--vm_templates--nics"></a>
### Nested Schema for `vm_templates.nics`

Read-Only:

- `enabled` (Boolean)
- `idx` (Number)
- `mirror` (Boolean)
- `model` (String)
- `vlan_id` (String)


