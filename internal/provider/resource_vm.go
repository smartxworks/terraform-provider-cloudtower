package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hasura/go-graphql-client"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/content_library_vm_template"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_disk"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_nic"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_snapshot"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_template"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_volume"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

func resourceVm() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "CloudTower vm resource.",

		CreateContext: resourceVmCreate,
		ReadContext:   resourceVmRead,
		UpdateContext: resourceVmUpdate,
		DeleteContext: resourceVmDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "VM's name",
			},
			"cluster_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "VM's cluster id",
			},
			"vcpu": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "VM's vcpu",
			},
			"memory": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Computed:    true,
				Description: "VM's memory, in the unit of byte, must be a multiple of 512MB, long value, ignore the decimal point",
				ValidateDiagFunc: func(v interface{}, _ cty.Path) diag.Diagnostics {
					var diags diag.Diagnostics
					val, ok := v.(float64)
					if !ok {
						return append(diags, diag.Diagnostic{
							Severity: diag.Error,
							Summary:  "Wrong type",
							Detail:   "Memory should be a number",
						})
					}
					intVal := int64(val)
					if intVal%512*1024 != 0 {
						return append(diags, diag.Diagnostic{
							Severity: diag.Error,
							Summary:  "Wrong type",
							Detail:   "Memory should be a multiple of 512MB",
						})
					}
					return diags
				},
			},
			"ha": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "whether VM is HA or not",
			},
			"firmware": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				Description:  "VM's firmware, forcenew as it isn't able to modify after create, must be one of 'BIOS', 'UEFI'",
				ValidateFunc: validation.StringInSlice([]string{"BIOS", "UEFI"}, false),
			},
			"status": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "VM's status",
				ValidateFunc: validation.StringInSlice([]string{"RUNNING", "STOPPED", "SUSPENDED"}, false),
			},
			"force_status_change": {
				Type:        schema.TypeBool,
				Description: "force VM's status change, will apply when power off or restart",
				Optional:    true,
			},
			"disk": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "VM's virtual disks",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"boot": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "VM disk's boot order",
						},
						"bus": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "VM disk's bus",
							ValidateFunc: validation.StringInSlice([]string{"IDE", "SCSI", "VIRTIO"}, false),
						},
						"vm_volume_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "use an existing VM volume as a VM disk, by specific it's id",
						},
						"vm_volume": {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: "create a new VM volume and use it as a VM disk",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"storage_policy": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "the new VM volume's storage policy",
										ValidateFunc: validation.StringInSlice(
											[]string{
												"REPLICA_2_THIN_PROVISION",
												"REPLICA_2_THICK_PROVISION",
												"REPLICA_3_THIN_PROVISION",
												"REPLICA_3_THICK_PROVISION",
											}, false,
										),
									},
									"name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "the new VM volume's name",
									},
									"size": {
										Type:        schema.TypeFloat,
										Required:    true,
										Description: "the new VM volume's size, in the unit of byte",
									},
									"path": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "the VM volume's iscsi LUN path",
									},
									"origin_path": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "the VM volume will create base on the path",
									},
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "the VM volume's id",
									},
								},
							},
						},
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the VM disk's id",
						},
					},
				},
			},
			"cd_rom": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "VM's CD-ROM",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"boot": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "VM CD-ROM's boot order",
						},
						"iso_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "mount an ISO to a VM CD-ROM by specific it's id",
						},
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VM CD-ROM's id",
						},
					},
				},
			},
			"nic": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "VM's virtual nic",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vlan_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "specific the vlan's id the VM nic will use",
						},
						"enabled": {
							Type:        schema.TypeBool,
							Description: "whether the VM nic is enabled",
							Optional:    true,
							Computed:    true,
						},
						"mirror": {
							Type:        schema.TypeBool,
							Description: "whether the VM nic use mirror mode",
							Optional:    true,
							Computed:    true,
						},
						"model": {
							Type:         schema.TypeString,
							Description:  "VM nic's model",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.StringInSlice([]string{"E1000", "SRIOV", "VIRTIO"}, false),
						},
						"mac_address": {
							Type:        schema.TypeString,
							Description: "VM nic's mac address",
							Optional:    true,
							Computed:    true,
						},
						"ip_address": {
							Type:        schema.TypeString,
							Description: "VM nic's IP address",
							Optional:    true,
							Computed:    true,
						},
						"subnet_mask": {
							Type:        schema.TypeString,
							Description: "VM nic's subnet mask",
							Optional:    true,
							Computed:    true,
						},
						"gateway": {
							Type:        schema.TypeString,
							Description: "VM nic's gateway",
							Optional:    true,
							Computed:    true,
						},
						"id": {
							Type:        schema.TypeString,
							Description: "VM nic's id",
							Computed:    true,
						},
						"idx": {
							Type:        schema.TypeInt,
							Description: "VM nic's index",
							Computed:    true,
						},
					},
				},
			},
			// create vm from another resource, template, snapshot or source vm
			"create_effect": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"clone_from_vm": {
							Type:          schema.TypeString,
							Optional:      true,
							ForceNew:      true,
							ConflictsWith: []string{"create_effect.0.clone_from_template", "create_effect.0.rebuild_from_snapshot", "create_effect.0.clone_from_content_library_template"},
							Description:   "Id of source vm from created vm to be cloned from",
						},
						"clone_from_template": {
							Type:          schema.TypeString,
							Optional:      true,
							ForceNew:      true,
							ConflictsWith: []string{"create_effect.0.clone_from_vm", "create_effect.0.rebuild_from_snapshot", "create_effect.0.clone_from_content_library_template"},
							Description:   "Id of source VM template to be cloned",
						},
						"clone_from_content_library_template": {
							Type:          schema.TypeString,
							Optional:      true,
							ForceNew:      true,
							ConflictsWith: []string{"create_effect.0.clone_from_template", "create_effect.0.rebuild_from_snapshot", "create_effect.0.clone_from_vm"},
							Description:   "Id of source content library VM template to be cloned",
						},
						"is_full_copy": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "If the vm is full copy from template or not",
						},
						"rebuild_from_snapshot": {
							Type:          schema.TypeString,
							Optional:      true,
							ForceNew:      true,
							ConflictsWith: []string{"create_effect.0.clone_from_template", "create_effect.0.clone_from_vm", "create_effect.0.clone_from_vm"},
							Description:   "Id of snapshot for created vm to be rebuilt from",
						},
						"cloud_init": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Set up cloud-init config when create vm from template",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"default_user_password": {
										Type:        schema.TypeString,
										Optional:    true,
										ForceNew:    true,
										Description: "Password of default user",
									},
									"nameservers": {
										Type: schema.TypeList,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
										Optional:    true,
										ForceNew:    true,
										MaxItems:    3,
										Description: "Name server address list. At most 3 name servers are allowed.",
									},
									"public_keys": {
										Type: schema.TypeList,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
										Optional:    true,
										ForceNew:    true,
										Description: "Add a list of public keys for the cloud-init default user.At most 10 public keys can be added to the list.",
									},
									"networks": {
										Type:     schema.TypeList,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"ip_address": {
													Type:        schema.TypeString,
													Optional:    true,
													ForceNew:    true,
													Description: "IPv4 address. This field is only used when type is not set to ipv4_dhcp.",
												},
												"netmask": {
													Type:        schema.TypeString,
													Optional:    true,
													ForceNew:    true,
													Description: "Netmask. This field is only used when type is not set to ipv4_dhcp.",
												},
												"nic_index": {
													Type:        schema.TypeInt,
													Required:    true,
													ForceNew:    true,
													Description: "Index of VM NICs. The index starts at 0, which refers to the first NIC.At most 16 NICs are supported, so the index range is [0, 15].",
												},
												"routes": {
													Type:        schema.TypeList,
													Optional:    true,
													ForceNew:    true,
													MaxItems:    1,
													Description: "Static route list",
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"gateway": {
																Type:        schema.TypeString,
																Optional:    true,
																ForceNew:    true,
																Description: "Gateway to access the static route address.",
															},
															"netmask": {
																Type:        schema.TypeString,
																Optional:    true,
																ForceNew:    true,
																Description: "Netmask of the network",
															},
															"network": {
																Type:        schema.TypeString,
																Optional:    true,
																ForceNew:    true,
																Description: "Static route network address. If set to 0.0.0.0, then first use the user settings to configure the default route.",
															},
														},
													},
												},
												"type": {
													Type:        schema.TypeString,
													Required:    true,
													ForceNew:    true,
													Description: "Network type. Allowed enum values are ipv4, ipv4_dhcp.",
													ValidateDiagFunc: func(v interface{}, _ cty.Path) diag.Diagnostics {
														var diags diag.Diagnostics
														val, ok := v.(string)
														if !ok {
															return append(diags, diag.Diagnostic{
																Severity: diag.Error,
																Summary:  "Wrong type",
																Detail:   "type should be a string",
															})
														} else if val != string(models.CloudInitNetworkTypeEnumIPV4) && val != string(models.CloudInitNetworkTypeEnumIPV4DHCP) {
															return append(diags, diag.Diagnostic{
																Severity: diag.Error,
																Summary:  "Invalid type",
																Detail: fmt.Sprintf("type should be one of %v, but get %s",
																	[]string{string(models.CloudInitNetworkTypeEnumIPV4), string(models.CloudInitNetworkTypeEnumIPV4DHCP)},
																	val,
																),
															})
														}
														return diags
													},
												},
											},
										},
										Optional:    true,
										ForceNew:    true,
										Description: "Network configuration list.",
									},
									"hostname": {
										Type:        schema.TypeString,
										Optional:    true,
										ForceNew:    true,
										Description: "hostname",
									},
									"user_data": {
										Type:        schema.TypeString,
										Optional:    true,
										ForceNew:    true,
										Description: "User-provided cloud-init user-data field. Base64 encoding is not supported. Size limit: 32KiB.",
									},
								},
							},
						},
					},
				},
			},
			"rollback_to": {
				Type:        schema.TypeString,
				Description: "Vm is going to rollback to target snapshot",
				Optional:    true,
			},
			// computed
			"host_id": {
				Type:        schema.TypeString,
				Description: "VM's host id",
				Optional:    true,
				Computed:    true,
			},
			"folder_id": {
				Type:        schema.TypeString,
				Description: "VM's folder id",
				Optional:    true,
				Computed:    true,
			},
			"description": {
				Type:        schema.TypeString,
				Description: "VM's description",
				Optional:    true,
				Computed:    true,
			},
			"guest_os_type": {
				Type:         schema.TypeString,
				Description:  "VM's guest OS type",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"LINUX", "WINDOWS", "UNKNOWN"}, false),
			},
			"cpu_cores": {
				Type:        schema.TypeInt,
				Description: "VM's cpu cores",
				Optional:    true,
				Computed:    true,
			},
			"cpu_sockets": {
				Type:        schema.TypeInt,
				Description: "VM's cpu sockets",
				Optional:    true,
				Computed:    true,
			},
			"id": {
				Type:        schema.TypeString,
				Description: "VM's id",
				Computed:    true,
			},
		},
	}
}

type VmDisk struct {
	Id         string           `json:"id"`
	Boot       int              `json:"boot"`
	Bus        models.Bus       `json:"bus"`
	VmVolumeId string           `json:"vm_volume_id"`
	VmVolume   []VmDiskVmVolume `json:"vm_volume"`
}

type VmDiskVmVolume struct {
	Id            string                              `json:"id"`
	StoragePolicy models.VMVolumeElfStoragePolicyType `json:"storage_policy"`
	Name          string                              `json:"name"`
	Size          int64                               `json:"size"`
	Path          string                              `json:"path"`
	OriginPath    string                              `json:"origin_path"`
}

type CdRom struct {
	Id    string `json:"id"`
	Boot  int32  `json:"boot"`
	IsoId string `json:"iso_id"`
}

type VmNic struct {
	models.VMNicParams
	VlanId string `json:"vlan_id"`
	Id     string `json:"id"`
	Idx    int    `json:"idx"`
}

type VmUpdateInput struct {
	Name        string        `json:"name,omitempty"`
	Memory      int64         `json:"memory,omitempty"`
	Ha          bool          `json:"ha,omitempty"`
	Description string        `json:"description,omitempty"`
	VCpu        int32         `json:"vcpu,omitempty"`
	Cpu         *CpuStruct    `json:"cpu,omitempty"`
	VmNics      *VmNicStruct  `json:"vm_nics,omitempty"`
	VmDisks     *VmDiskStruct `json:"vm_disks,omitempty"`
}

type CpuStruct struct {
	Sockets int32 `json:"sockets,omitempty"`
	Cores   int32 `json:"cores,omitempty"`
}
type VmNicStruct struct {
	Create []VmNicCreate `json:"create,omitempty"`
	Delete []VmNicDelete `json:"delete,omitempty"`
}
type VmNicCreate struct {
	Enabled    bool              `json:"enabled,omitempty"`
	Gateway    string            `json:"gateway,omitempty"`
	IPAddress  string            `json:"ip_address,omitempty"`
	MacAddress string            `json:"mac_address,omitempty"`
	SubnetMask string            `json:"subnet_mask,omitempty"`
	LocalId    string            `json:"local_id"`
	Model      models.VMNicModel `json:"model,omitempty"`
	Nic        *ConnectStruct    `json:"nic,omitempty"`
	Vlan       *ConnectStruct    `json:"vlan,omitempty"`
}

type VmNicDelete struct {
	Id string `json:"id"`
}
type VmDiskStruct struct {
	Create []VmDiskCreate `json:"create,omitempty"`
	Delete []VmDiskDelete `json:"delete,omitempty"`
	Update []VmDiskUpdate `json:"update,omitempty"`
}

type VmDiskCreate struct {
	Boot     int                          `json:"boot,omitempty"`
	Bus      models.Bus                   `json:"bus,omitempty"`
	Type     models.VMDiskType            `json:"type,omitempty"`
	ElfImage *ConnectStruct               `json:"elf_image,omitempty"`
	VmVolume *VmVolumeCreateConnectStruct `json:"vm_volume,omitempty"`
}

type VmDiskDelete struct {
	Id string `json:"id"`
}

type VmDiskUpdate struct {
	Where VmDiskWhereInput `json:"where"`
	Data  VmDiskUpdateData `json:"data"`
}

type VmDiskWhereInput struct {
	Id string `json:"id"`
}

type VmDiskUpdateData struct {
	Boot     int                          `json:"boot,omitempty"`
	Bus      models.Bus                   `json:"bus,omitempty"`
	Key      int32                        `json:"key,omitempty"`
	Type     models.VMDiskType            `json:"type,omitempty"`
	Disabled bool                         `json:"disabled,omitempty"`
	ElfImage *ConnectStruct               `json:"elf_image,omitempty"`
	VmVolume *VmVolumeCreateConnectStruct `json:"vm_volume,omitempty"`
}

type VmVolumeCreateConnectStruct struct {
	Connect    *ConnectConnect      `json:"connect,omitempty"`
	Disconnect *bool                `json:"disconnect,omitempty"`
	Create     *VmVolumeCreateInput `json:"create,omitempty"`
}

type VmVolumeCreateInput struct {
	Name             string                              `json:"name"`
	Path             string                              `json:"path"`
	Size             int64                               `json:"size"`
	ElfStoragePolicy models.VMVolumeElfStoragePolicyType `json:"elf_storage_policy"`
	LocalCreatedAt   string                              `json:"local_created_at"`
	LocalId          string                              `json:"local_id"`
	Mounting         bool                                `json:"mounting"`
	Sharing          bool                                `json:"sharing"`
	Cluster          *ConnectStruct                      `json:"cluster,omitempty"`
}

type ConnectStruct struct {
	Connect    *ConnectConnect `json:"connect,omitempty"`
	Disconnect *bool           `json:"disconnect,omitempty"`
}

type ConnectConnect struct {
	Id string `json:"id"`
}

type UpdateVmEffect map[string]interface{}
type VmWhereUniqueInput map[string]interface{}

var updateVm struct {
	UpdateVm struct {
		Id graphql.String
	} `graphql:"updateVm(data: $data, effect: $effect,where:$where)"`
}

func resourceVmCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	var vms []*models.WithTaskVM
	var diags diag.Diagnostics
	rebuildFrom := d.Get("create_effect.0.rebuild_from_snapshot").(string)
	cloneFrom := d.Get("create_effect.0.clone_from_vm").(string)
	cloneFromTemplate := d.Get("create_effect.0.clone_from_template").(string)
	cloneFromContentLibraryTemplate := d.Get("create_effect.0.clone_from_content_library_template").(string)
	effects := d.Get("create_effect.0").(map[string]interface{})
	count := 0
	for k := range effects {
		if effects[k] != "" {
			if k == "rebuild_from_snapshot" || k == "clone_from_vm" || k == "clone_from_template" || k == "clone_from_content_library_template" {
				count = count + 1
			}
		}
	}
	if count >= 2 {
		return diag.FromErr(fmt.Errorf("can only set one create effect"))
	} else if rebuildFrom != "" {
		vms, diags = rebuildVmFromSnapshot(rebuildFrom, ctx, d, ct)
	} else if cloneFrom != "" {
		vms, diags = cloneVmFromSourceVm(cloneFrom, ctx, d, ct)
	} else if cloneFromTemplate != "" {
		vms, diags = cloneVmFromVmTemplate(cloneFromTemplate, ctx, d, ct)
	} else if cloneFromContentLibraryTemplate != "" {
		vms, diags = cloneVmFromContentLibraryVmTemplate(cloneFromContentLibraryTemplate, ctx, d, ct)
	} else {
		vms, diags = createBlankVm(ctx, d, ct)
	}
	if vms == nil {
		return diags
	}
	err := waitVmTasksFinish(ct, vms)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(*vms[0].Data.ID)
	return resourceVmRead(ctx, d, meta)
}

func resourceVmRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)

	v, diags := readVm(ctx, d, ct)
	if diags != nil || v == nil {
		return diags
	}

	// set computed variables
	if err := d.Set("cluster_id", v.Cluster.ID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("vcpu", v.Vcpu); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("memory", v.Memory); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("ha", v.Ha); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("firmware", v.Firmware); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", v.Status); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("host_id", v.Host.ID); err != nil {
		return diag.FromErr(err)
	}
	if v.Folder != nil {
		if err := d.Set("folder_id", v.Folder.ID); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("description", v.Description); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("guest_os_type", v.GuestOsType); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cpu_cores", v.CPU.Cores); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cpu_sockets", v.CPU.Sockets); err != nil {
		return diag.FromErr(err)
	}
	vmNics, diags := readVmNics(ctx, d, ct)
	if diags != nil {
		return diags
	}
	var nics []map[string]interface{}
	for idx, n := range vmNics {
		nics = append(nics, map[string]interface{}{
			"vlan_id":     n.Vlan.ID,
			"enabled":     n.Enabled,
			"mirror":      n.Mirror,
			"model":       n.Model,
			"mac_address": n.MacAddress,
			"ip_address":  n.IPAddress,
			"subnet_mask": n.SubnetMask,
			"gateway":     n.Gateway,
			"id":          n.ID,
			"idx":         idx,
		})
	}
	if err := d.Set("nic", nics); err != nil {
		return diag.FromErr(err)
	}

	vmDisks, vmVolumes, diags := readVmDisks(ctx, d, ct)
	if diags != nil {
		return diags
	}
	var disks []map[string]interface{}
	for idx, disk := range vmDisks {
		vmVolume := vmVolumes[idx]
		vmVolumeData := map[string]interface{}{
			"id": disk.VMVolume.ID,
		}
		if vmVolume != nil {
			vmVolumeData["name"] = vmVolume.Name
			vmVolumeData["size"] = vmVolume.Size
			vmVolumeData["path"] = vmVolume.Path
			vmVolumeData["origin_path"] = d.Get(fmt.Sprintf("disk.%d.vm_volume.0.origin_path", idx))
			vmVolumeData["storage_policy"] = vmVolume.ElfStoragePolicy
		}
		disks = append(disks, map[string]interface{}{
			"id":   disk.ID,
			"boot": disk.Boot,
			"bus":  disk.Bus,
			"vm_volume": []map[string]interface{}{
				vmVolumeData,
			},
			"vm_volume_id": disk.VMVolume.ID,
		})
	}
	if err := d.Set("disk", disks); err != nil {
		return diag.FromErr(err)
	}

	cdRomsData, diags := readCdRoms(ctx, d, ct)
	if diags != nil {
		return diags
	}
	var cdRoms []map[string]interface{}
	for _, c := range cdRomsData {
		cdRom := map[string]interface{}{
			"id":   c.ID,
			"boot": c.Boot,
		}
		if c.ElfImage != nil {
			cdRom["iso_id"] = c.ElfImage.ID
		}
		cdRoms = append(cdRoms, cdRom)
	}
	if err := d.Set("cd_rom", cdRoms); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceVmUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	id := d.Id()
	originalVm, _ := readVm(ctx, d, ct)
	var updateParams VmUpdateInput = VmUpdateInput{}
	// handle update data first, if there is error in data, will not execute other mutation
	if d.HasChanges("name", "vcpu", "memory", "description", "ha", "cpu_cores", "cpu_sockets") {
		basic, err := expandVmBasicConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		updateParams.Name = basic.Name
		if basic.Memory != nil {
			updateParams.Memory = *basic.Memory
		}

		if basic.Ha != nil {
			updateParams.Ha = *basic.Ha
		}
		if basic.Description != nil {
			updateParams.Description = *basic.Description
		}
		if basic.CpuCores == nil || basic.CpuSockets == nil || basic.Vcpu == nil {
			if basic.CpuCores == nil {
				if basic.CpuSockets == nil {
					// do nothing if nothing of cpu config has changed
					if basic.Vcpu != nil {
						updateParams.VCpu = *basic.Vcpu
						if *basic.Vcpu%*originalVm.CPU.Sockets == 0 {
							newCore := *basic.Vcpu / *originalVm.CPU.Sockets
							updateParams.Cpu = &CpuStruct{
								Cores:   newCore,
								Sockets: *originalVm.CPU.Sockets,
							}
						} else if *basic.Vcpu%*originalVm.CPU.Cores == 0 {
							newSockets := *basic.Vcpu / *originalVm.CPU.Cores
							updateParams.Cpu = &CpuStruct{
								Cores:   *originalVm.CPU.Cores,
								Sockets: newSockets,
							}
						} else {
							updateParams.Cpu = &CpuStruct{
								Cores:   *basic.Vcpu,
								Sockets: 1,
							}
						}
					}
				} else {
					if basic.Vcpu != nil {
						if *basic.Vcpu%*basic.CpuSockets == 0 {
							newCore := *basic.Vcpu / *basic.CpuSockets
							updateParams.Cpu = &CpuStruct{
								Cores:   newCore,
								Sockets: *basic.CpuSockets,
							}
						} else {
							return diag.Errorf("vcpu must be divisible by number of cpu sockets")
						}
					} else {
						updateParams.VCpu = *basic.CpuSockets * *originalVm.CPU.Cores
						updateParams.Cpu = &CpuStruct{
							Cores:   *originalVm.CPU.Cores,
							Sockets: *basic.CpuSockets,
						}
					}
				}
			} else if basic.CpuSockets == nil {
				if basic.Vcpu == nil {
					updateParams.VCpu = (*originalVm.CPU.Sockets) * (*basic.CpuCores)
					updateParams.Cpu = &CpuStruct{
						Cores:   *basic.CpuCores,
						Sockets: *originalVm.CPU.Sockets,
					}
				} else {
					updateParams.VCpu = *basic.Vcpu
					if *basic.Vcpu%*basic.CpuCores == 0 {
						newSocket := *basic.Vcpu / *basic.CpuCores
						updateParams.Cpu = &CpuStruct{
							Cores:   *basic.CpuCores,
							Sockets: newSocket,
						}
					} else {
						return diag.Errorf("vcpu must be divisible by number of cpu cores")
					}
				}
			} else {
				updateParams.VCpu = (*basic.CpuSockets) * (*basic.CpuCores)
				updateParams.Cpu = &CpuStruct{
					Cores:   *basic.CpuCores,
					Sockets: *basic.CpuSockets,
				}
			}
		} else {
			updateParams.VCpu = *basic.Vcpu
			updateParams.Cpu = &CpuStruct{
				Cores:   *basic.CpuCores,
				Sockets: *basic.CpuSockets,
			}
		}
	}

	if d.HasChange("nic") {
		// delete all previous vm nic
		nicsToDelete := make([]VmNicDelete, 0)
		nicsToCreate := make([]VmNicCreate, 0)
		vmNics, diags := readVmNics(ctx, d, ct)
		if diags != nil {
			return diags
		}
		for _, n := range vmNics {
			nicsToDelete = append(nicsToDelete, VmNicDelete{
				Id: *n.ID,
			})
		}
		var nics []*VmNic
		bytes, err := json.Marshal(d.Get("nic"))
		if err != nil {
			return diag.FromErr(err)
		}
		err = json.Unmarshal(bytes, &nics)
		if err != nil {
			return diag.FromErr(err)
		}
		for _, n := range nics {
			var nicId string = ""
			if n.NicID != nil {
				nicId = *n.ConnectVlanID
			}
			var vlanId string = ""
			if n.ConnectVlanID != nil {
				vlanId = *n.ConnectVlanID
			}
			nicsToCreate = append(nicsToCreate,
				VmNicCreate{
					Enabled:    *n.Enabled,
					Gateway:    *n.Gateway,
					IPAddress:  *n.IPAddress,
					LocalId:    *n.LocalID,
					MacAddress: *n.MacAddress,
					Model:      *n.Model,
					SubnetMask: *n.SubnetMask,
					Nic: &ConnectStruct{
						Connect: &ConnectConnect{
							Id: nicId,
						},
					},
					Vlan: &ConnectStruct{
						Connect: &ConnectConnect{
							Id: vlanId,
						},
					},
				})
		}
		updateParams.VmNics = &VmNicStruct{
			Create: nicsToCreate,
			Delete: nicsToDelete,
		}
	}

	if d.HasChanges("cd_rom", "disk") {
		diskToCreate := make([]VmDiskCreate, 0)
		diskToUpdate := make([]VmDiskUpdate, 0)
		diskToDelete := make([]VmDiskDelete, 0)
		cdRoms, diags := readCdRoms(ctx, d, ct)
		if diags != nil {
			return diags
		}
		vmDisks, vmVolumes, diags := readVmDisks(ctx, d, ct)
		if diags != nil {
			return diags
		}
		curVolumeMap := make(map[string]*models.VMVolume, 0)
		for _, v := range vmVolumes {
			curVolumeMap[*v.ID] = v
		}
		if d.HasChange("cd_rom") {
			curMap := make(map[string]*models.VMDisk, 0)
			for _, v := range cdRoms {
				curMap[*v.ID] = v
			}
			var cdRomsData []*CdRom
			bytes, err := json.Marshal(d.Get("cd_rom"))
			if err != nil {
				return diag.FromErr(err)
			}
			err = json.Unmarshal(bytes, &cdRomsData)
			if err != nil {
				return diag.FromErr(err)
			}
			// push all current cd_rom to update, ignore deleted cd-rom, tower will handle it
			for _, cr := range cdRomsData {
				// for existing cd_rom, update it
				origin := curMap[cr.Id]
				if origin != nil {
					var data = VmDiskUpdateData{}
					data.Boot = int(cr.Boot)
					data.Bus = *origin.Bus
					data.Key = *origin.Key
					data.Disabled = *origin.Disabled
					data.Type = models.VMDiskTypeCDROM
					if origin.ElfImage != nil && cr.IsoId == "" {
						flag := true
						data.ElfImage = &ConnectStruct{
							Disconnect: &flag,
						}
					} else if !(cr.IsoId == "" && origin.ElfImage == nil) &&
						!(origin.ElfImage != nil && cr.IsoId == *origin.ElfImage.ID) {
						data.ElfImage = &ConnectStruct{
							Connect: &ConnectConnect{
								Id: cr.IsoId,
							},
						}
					}
					diskToUpdate = append(diskToUpdate, VmDiskUpdate{
						Where: VmDiskWhereInput{Id: cr.Id},
						Data:  data,
					})
				} else {
					var data = VmDiskCreate{
						Boot: int(cr.Boot),
						Bus:  models.BusIDE,
						Type: models.VMDiskTypeCDROM,
					}
					if cr.IsoId != "" {
						data.ElfImage = &ConnectStruct{
							Connect: &ConnectConnect{
								Id: cr.IsoId,
							},
						}
					}
					diskToCreate = append(diskToCreate, data)
				}
			}
		} else {
			// keep original cd_rom
			for _, v := range cdRoms {
				diskToUpdate = append(diskToUpdate, VmDiskUpdate{
					Where: VmDiskWhereInput{Id: *v.ID},
					Data: VmDiskUpdateData{
						Boot:     int(*v.Boot),
						Bus:      *v.Bus,
						Key:      *v.Key,
						Type:     models.VMDiskTypeCDROM,
						Disabled: *v.Disabled,
					},
				})
			}
		}
		if d.HasChange("disk") {
			curMap := make(map[string]*models.VMDisk, 0)
			for _, v := range vmDisks {
				// use volume id as key
				curMap[*v.VMVolume.ID] = v
			}
			var disks []*VmDisk
			bytes, err := json.Marshal(d.Get("disk"))
			if err != nil {
				return diag.FromErr(err)
			}
			err = json.Unmarshal(bytes, &disks)
			if err != nil {
				return diag.FromErr(err)
			}
			for _, vd := range disks {
				origin := curMap[vd.VmVolumeId]
				originVolume := curVolumeMap[vd.VmVolumeId]
				if vd.VmVolumeId != "" {
					if origin != nil {
						// if given volume is mounted, update it
						var data = VmDiskUpdateData{
							Boot: vd.Boot,
							Bus:  vd.Bus,
							Type: *origin.Type,
						}
						if len(vd.VmVolume) > 0 {
							newVolume := vd.VmVolume[0]
							if newVolume.Name != *originVolume.Name {
								//FIXME: move all invalid change detection to customizeDiffFunc
								return diag.Errorf("mounted disk %s's name can not be changed", *originVolume.Name)
							}
							if newVolume.Size < *originVolume.Size {
								return diag.Errorf("Disk %s's size can not shrink", *originVolume.Name)
							}
							if newVolume.StoragePolicy != *originVolume.ElfStoragePolicy {
								return diag.Errorf("mounted disk %s's storage policy can not be changed", *originVolume.Name)
							}
							data.VmVolume = &VmVolumeCreateConnectStruct{
								Create: &VmVolumeCreateInput{
									Name:             *originVolume.Name,
									Path:             newVolume.Path,
									Size:             newVolume.Size,
									ElfStoragePolicy: *originVolume.ElfStoragePolicy,
									LocalCreatedAt:   "",
									LocalId:          "",
									Mounting:         *originVolume.Mounting,
									Sharing:          *originVolume.Sharing,
									Cluster: &ConnectStruct{
										Connect: &ConnectConnect{
											Id: "",
										},
									},
								},
							}
						} else if originVolume != nil {
							// use original volume
							data.VmVolume = &VmVolumeCreateConnectStruct{
								Create: &VmVolumeCreateInput{
									Name:             *originVolume.Name,
									Path:             *originVolume.Path,
									Size:             *originVolume.Size,
									ElfStoragePolicy: *originVolume.ElfStoragePolicy,
									LocalCreatedAt:   "",
									LocalId:          "",
									Mounting:         *originVolume.Mounting,
									Sharing:          *originVolume.Sharing,
									Cluster: &ConnectStruct{
										Connect: &ConnectConnect{
											Id: "",
										},
									},
								},
							}
						}
						diskToUpdate = append(diskToUpdate, VmDiskUpdate{
							Where: VmDiskWhereInput{
								Id: vd.Id,
							},
							Data: data,
						})
						delete(curMap, vd.VmVolumeId)
					} else {
						// if giving volume is not mounted, mount it
						var data = VmDiskCreate{
							Boot: vd.Boot,
							Bus:  vd.Bus,
							Type: models.VMDiskTypeDISK,
							VmVolume: &VmVolumeCreateConnectStruct{
								Connect: &ConnectConnect{
									Id: vd.VmVolumeId,
								},
							},
						}
						diskToCreate = append(diskToCreate, data)
					}
				} else if len(vd.VmVolume) > 0 {
					// no volume is given, but VmVolume is configured mean we need to create one
					var data = VmDiskCreate{
						Boot: vd.Boot,
						Bus:  vd.Bus,
						Type: models.VMDiskTypeDISK,
						VmVolume: &VmVolumeCreateConnectStruct{
							Create: &VmVolumeCreateInput{
								Name:             vd.VmVolume[0].Name,
								Path:             "",
								Size:             vd.VmVolume[0].Size,
								ElfStoragePolicy: vd.VmVolume[0].StoragePolicy,
								LocalCreatedAt:   "",
								LocalId:          "",
								Mounting:         true,
								Sharing:          false,
								Cluster: &ConnectStruct{
									Connect: &ConnectConnect{
										Id: "",
									},
								},
							},
						},
					}
					diskToCreate = append(diskToCreate, data)
				}
				for _, d := range curMap {
					diskToDelete = append(diskToDelete, VmDiskDelete{
						Id: *d.ID,
					})
				}
			}
		} else {
			// keep original disks
			for _, v := range vmDisks {
				var originVmVolume = curVolumeMap[*v.VMVolume.ID]
				if originVmVolume != nil {
					var data = VmDiskUpdateData{
						Boot: int(*v.Boot),
						Bus:  *v.Bus,
						Type: models.VMDiskTypeDISK,
						VmVolume: &VmVolumeCreateConnectStruct{
							Create: &VmVolumeCreateInput{
								Name:             *originVmVolume.Name,
								Path:             *originVmVolume.Path,
								Size:             *originVmVolume.Size,
								Sharing:          *originVmVolume.Sharing,
								ElfStoragePolicy: *originVmVolume.ElfStoragePolicy,
								LocalCreatedAt:   "",
								LocalId:          "",
								Mounting:         *originVmVolume.Mounting,
								Cluster: &ConnectStruct{
									Connect: &ConnectConnect{
										Id: "",
									},
								},
							},
						},
					}
					diskToUpdate = append(diskToUpdate, VmDiskUpdate{
						Where: VmDiskWhereInput{
							Id: *v.ID,
						},
						Data: data,
					})
				}
			}
		}
		sort.SliceStable(diskToCreate, func(i, j int) bool {
			return diskToCreate[i].Boot < diskToCreate[j].Boot
		})

		sort.SliceStable(diskToUpdate, func(i, j int) bool {
			return diskToUpdate[i].Data.Boot < diskToUpdate[j].Data.Boot
		})

		updateParams.VmDisks = &VmDiskStruct{
			Create: diskToCreate,
			Update: diskToUpdate,
			Delete: diskToDelete,
		}
	}
	// rollback vm to target state first if rollback_to not match rollback from
	if d.HasChange("rollback_to") {
		rawRollbackTo, ok := d.GetOk("rollback_to")
		if ok {
			rollbackTo := rawRollbackTo.(string)
			if rollbackTo != "" {
				// if rollbackTo was set to empty string or undefined, not do anything
				id := d.Id()
				rp := vm.NewRollbackVMParams()
				rp.RequestBody = &models.VMRollbackParams{
					Where: &models.VMWhereInput{
						ID: &id,
					},
					Data: &models.VMRollbackParamsData{
						SnapshotID: &rollbackTo,
					},
				}
				vms, err := ct.Api.VM.RollbackVM(rp)
				if err != nil {
					return diag.FromErr(err)
				}
				waitVmTasksFinish(ct, vms.Payload)
				resourceVmRead(ctx, d, meta)
			}
		}
	}
	// change vm status first as some change could not make in some status
	if d.HasChange("status") {
		basic, err := expandVmBasicConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		status, err := expandVmStatusConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		switch *status.Status {
		case models.VMStatusRUNNING:
			uvp := vm.NewStartVMParams()
			uvp.RequestBody = &models.VMStartParams{
				Where: &models.VMWhereInput{
					ID: &id,
				},
				Data: &models.VMStartParamsData{
					HostID: basic.HostId,
				},
			}
			vms, err := ct.Api.VM.StartVM(uvp)
			if err != nil {
				return diag.FromErr(err)
			}
			err = waitVmTasksFinish(ct, vms.Payload)
			if err != nil {
				return diag.FromErr(err)
			}
		case models.VMStatusSTOPPED:
			if status.Force {
				uvp := vm.NewPoweroffVMParams()
				uvp.RequestBody = &models.VMOperateParams{
					Where: &models.VMWhereInput{
						ID: &id,
					},
				}
				vms, err := ct.Api.VM.PoweroffVM(uvp)
				if err != nil {
					return diag.FromErr(err)
				}
				err = waitVmTasksFinish(ct, vms.Payload)
				if err != nil {
					return diag.FromErr(err)
				}
			} else {
				uvp := vm.NewShutDownVMParams()
				uvp.RequestBody = &models.VMOperateParams{
					Where: &models.VMWhereInput{
						ID: &id,
					},
				}
				vms, err := ct.Api.VM.ShutDownVM(uvp)
				if err != nil {
					return diag.FromErr(err)
				}
				err = waitVmTasksFinish(ct, vms.Payload)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		case models.VMStatusSUSPENDED:
			uvp := vm.NewSuspendVMParams()
			uvp.RequestBody = &models.VMOperateParams{
				Where: &models.VMWhereInput{
					ID: &id,
				},
			}
			vms, err := ct.Api.VM.SuspendVM(uvp)
			if err != nil {
				return diag.FromErr(err)
			}
			err = waitVmTasksFinish(ct, vms.Payload)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	// then migrate the vm if needed
	if d.HasChange("host_id") {
		hostId := d.Get("host_id").(string)
		var mvpd *models.VMMigrateParamsData = nil
		// if set to AUTO_SCHEDULE, try to auto migrate to the proper host
		if hostId != "" && hostId != "AUTO_SCHEDULE" {
			mvpd = &models.VMMigrateParamsData{
				HostID: &hostId,
			}
		}
		mvp := vm.NewMigrateVMParams()
		mvp.RequestBody = &models.VMMigrateParams{
			Where: &models.VMWhereInput{
				ID: &id,
			},
			Data: mvpd,
		}
		vms, err := ct.Api.VM.MigrateVM(mvp)
		if err != nil {
			return diag.FromErr(err)
		}
		err = waitVmTasksFinish(ct, vms.Payload)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// execute vm updation in the last
	err := ct.GraphqlApi.Mutate(context.Background(), &updateVm, map[string]interface{}{
		"data":   updateParams,
		"effect": UpdateVmEffect{},
		"where": VmWhereUniqueInput{
			"id": d.Id(),
		},
	}, graphql.OperationName("updateVm"))
	if err != nil {
		return diag.FromErr(err)
	}
	ct.WaitTaskForResource(d.Id(), "updateVm")
	return resourceVmRead(ctx, d, meta)
}

func resourceVmDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)
	dvp := vm.NewDeleteVMParams()
	id := d.Id()
	dvp.RequestBody = &models.VMDeleteParams{
		Where: &models.VMWhereInput{
			ID: &id,
		},
	}
	vms, err := ct.Api.VM.DeleteVM(dvp)
	if err != nil {
		return diag.FromErr(err)
	}
	taskIds := make([]string, 0)
	for _, v := range vms.Payload {
		if v.TaskID != nil {
			taskIds = append(taskIds, *v.TaskID)
		}
	}
	_, err = ct.WaitTasksFinish(taskIds)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return diags
}

func waitVmTasksFinish(ct *cloudtower.Client, vms []*models.WithTaskVM) error {
	taskIds := make([]string, 0)
	for _, v := range vms {
		if v.TaskID != nil {
			taskIds = append(taskIds, *v.TaskID)
		}
	}
	_, err := ct.WaitTasksFinish(taskIds)
	return err
}

type VmBasicConfig struct {
	Name        string
	Vcpu        *int32
	Memory      *int64
	Ha          *bool
	HostId      *string
	FolderId    *string
	Description *string
	CpuCores    *int32
	CpuSockets  *int32
}

func expandVmBasicConfig(d *schema.ResourceData) (*VmBasicConfig, error) {
	basicConfig := &VmBasicConfig{}
	basicConfig.Name = d.Get("name").(string)
	vcpu, ok := d.GetOk("vcpu")
	if ok {
		vcpu := int32(vcpu.(int))
		basicConfig.Vcpu = &vcpu
	}
	memory, ok := d.GetOk("memory")
	if ok {
		memory := int64(memory.(float64))
		basicConfig.Memory = &memory
	}
	ha, ok := d.GetOkExists("ha")
	if ok {
		ha := ha.(bool)
		basicConfig.Ha = &ha
	}
	hostId, ok := d.GetOk("host_id")
	if ok {
		hostId := hostId.(string)
		basicConfig.HostId = &hostId
	}
	folderId, ok := d.GetOk("folder_id")
	if ok {
		folderId := folderId.(string)
		basicConfig.FolderId = &folderId
	}
	description, ok := d.GetOk("description")
	if ok {
		description := description.(string)
		basicConfig.Description = &description
	}
	cpuCores, ok := d.GetOk("cpu_cores")
	if ok {
		cpuCores := int32(cpuCores.(int))
		basicConfig.CpuCores = &cpuCores
	}
	cpuSockets, ok := d.GetOk("cpu_sockets")
	if ok {
		cpuSockets := int32(cpuSockets.(int))
		basicConfig.CpuSockets = &cpuSockets
	}
	return basicConfig, nil
}

func expandCloudInitConfig(d *schema.ResourceData) (*models.TemplateCloudInit, error) {
	cloudInit := &models.TemplateCloudInit{}
	defaultPassword, ok := d.GetOk("create_effect.0.cloud_init.0.default_user_password")
	if ok {
		defaultPassword := defaultPassword.(string)
		cloudInit.DefaultUserPassword = &defaultPassword
	}
	nameservers, ok := d.GetOk("create_effect.0.cloud_init.0.nameservers")
	if ok {
		bytes, err := json.Marshal(nameservers)
		if err != nil {
			return nil, err
		}
		nameservers := make([]string, 0)
		err = json.Unmarshal(bytes, &nameservers)
		if err != nil {
			return nil, err
		}
		cloudInit.Nameservers = nameservers
	}
	publicKeys, ok := d.GetOk("create_effect.0.cloud_init.0.public_keys")
	if ok {
		bytes, err := json.Marshal(publicKeys)
		if err != nil {
			return nil, err
		}
		publicKeys := make([]string, 0)
		err = json.Unmarshal(bytes, &publicKeys)
		if err != nil {
			return nil, err
		}
		cloudInit.PublicKeys = publicKeys
	}
	hostName, ok := d.GetOk("create_effect.0.cloud_init.0.hostname")
	if ok {
		hostName := hostName.(string)
		cloudInit.Hostname = &hostName
	}
	userData, ok := d.GetOk("create_effect.0.cloud_init.0.user_data")
	if ok {
		userData := userData.(string)
		cloudInit.UserData = &userData
	}
	networks, ok := d.GetOk("create_effect.0.cloud_init.0.networks")
	if ok {
		bytes, err := json.Marshal(networks)
		if err != nil {
			return nil, err
		}
		var networks []*models.CloudInitNetWork
		err = json.Unmarshal(bytes, &networks)
		if err != nil {
			return nil, err
		}
		cloudInit.Networks = networks
	}
	return cloudInit, nil
}

type VmStatusConfig struct {
	Status *models.VMStatus
	Force  bool
}

func expandVmStatusConfig(d *schema.ResourceData) (*VmStatusConfig, error) {
	var status *models.VMStatus
	switch d.Get("status").(string) {
	case "RUNNING":
		status = models.VMStatusRUNNING.Pointer()
	case "STOPPED":
		status = models.VMStatusSTOPPED.Pointer()
	case "SUSPENDED":
		status = models.VMStatusSUSPENDED.Pointer()
	}
	force := d.Get("force_status_change").(bool)
	return &VmStatusConfig{
		Status: status,
		Force:  force,
	}, nil
}

func readVm(ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) (*models.VM, diag.Diagnostics) {
	var diags diag.Diagnostics
	id := d.Id()
	gcp := vm.NewGetVmsParams()
	gcp.RequestBody = &models.GetVmsRequestBody{
		Where: &models.VMWhereInput{
			ID: &id,
		},
	}
	vms, err := ct.Api.VM.GetVms(gcp)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	if len(vms.Payload) < 1 {
		d.SetId("")
		return nil, diags
	}
	v := vms.Payload[0]
	return v, nil
}

func readVmNics(ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) ([]*models.VMNic, diag.Diagnostics) {
	id := d.Id()
	gp := vm_nic.NewGetVMNicsParams()
	gp.RequestBody = &models.GetVMNicsRequestBody{
		Where: &models.VMNicWhereInput{
			VM: &models.VMWhereInput{
				ID: &id,
			},
		},
	}
	vmNics, err := ct.Api.VMNic.GetVMNics(gp)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	return vmNics.Payload, nil
}

func readVmDisks(ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) ([]*models.VMDisk, []*models.VMVolume, diag.Diagnostics) {
	id := d.Id()
	gp := vm_disk.NewGetVMDisksParams()
	typePt := models.VMDiskTypeDISK
	gp.RequestBody = &models.GetVMDisksRequestBody{
		Where: &models.VMDiskWhereInput{
			Type: &typePt,
			VM: &models.VMWhereInput{
				ID: &id,
			},
			CloudInitImagePathNot: nil,
		},
	}
	vmDisks, err := ct.Api.VMDisk.GetVMDisks(gp)
	if err != nil {
		return nil, nil, diag.FromErr(err)
	}
	gp2 := vm_volume.NewGetVMVolumesParams()
	gp2.RequestBody = &models.GetVMVolumesRequestBody{
		Where: &models.VMVolumeWhereInput{
			VMDisksSome: &models.VMDiskWhereInput{
				Type: &typePt,
				VM: &models.VMWhereInput{
					ID: &id,
				},
			},
		},
	}
	vmVolumes, err := ct.Api.VMVolume.GetVMVolumes(gp2)
	if err != nil {
		return nil, nil, diag.FromErr(err)
	}
	vmVolumeMap := make(map[string]*models.VMVolume, 0)
	for _, v := range vmVolumes.Payload {
		vmVolumeMap[*v.ID] = v
	}
	vmVolumesSlice := make([]*models.VMVolume, len(vmDisks.Payload))
	for idx, v := range vmDisks.Payload {
		vmVolume := vmVolumeMap[*v.VMVolume.ID]
		vmVolumesSlice[idx] = vmVolume
	}
	return vmDisks.Payload, vmVolumesSlice, nil
}

func readVmDisksFromTemplate(ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) ([]*models.NestedFrozenDisks, diag.Diagnostics) {
	cloneFromTemplate := d.Get("create_effect.0.clone_from_template").(string)
	gp := vm_template.NewGetVMTemplatesParams()
	gp.RequestBody = &models.GetVMTemplatesRequestBody{
		Where: &models.VMTemplateWhereInput{
			ID: &cloneFromTemplate,
		},
	}
	vmTemplates, err := ct.Api.VMTemplate.GetVMTemplates(gp)
	if err != nil {
		return nil, diag.FromErr(err)
	} else if len(vmTemplates.Payload) == 0 {
		return nil, diag.FromErr(fmt.Errorf("template %s not found", cloneFromTemplate))
	}
	return vmTemplates.Payload[0].VMDisks, nil
}

func readVmDisksFromContentLibraryTemplate(ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) ([]*models.NestedFrozenDisks, diag.Diagnostics) {
	cloneFromTemplate := d.Get("create_effect.0.clone_from_content_library_template").(string)
	gcp := content_library_vm_template.NewGetContentLibraryVMTemplatesParams()
	gcp.RequestBody = &models.GetContentLibraryVMTemplatesRequestBody{
		Where: &models.ContentLibraryVMTemplateWhereInput{
			ID: &cloneFromTemplate,
		},
	}
	contentLibraryVmTemplates, err := ct.Api.ContentLibraryVMTemplate.GetContentLibraryVMTemplates(gcp)
	if err != nil {
		return nil, diag.FromErr(err)
	} else if len(contentLibraryVmTemplates.Payload) == 0 {
		return nil, diag.FromErr(fmt.Errorf("content library template %s not found", cloneFromTemplate))
	}
	gp := vm_template.NewGetVMTemplatesParams()
	gp.RequestBody = &models.GetVMTemplatesRequestBody{
		Where: &models.VMTemplateWhereInput{
			ID: contentLibraryVmTemplates.Payload[0].VMTemplates[0].ID,
		},
	}
	vmTemplates, err := ct.Api.VMTemplate.GetVMTemplates(gp)
	if err != nil {
		return nil, diag.FromErr(err)
	} else if len(vmTemplates.Payload) == 0 {
		return nil, diag.FromErr(fmt.Errorf("template %s not found", cloneFromTemplate))
	}
	return vmTemplates.Payload[0].VMDisks, nil
}

func readVmDisksFromSnapshot(ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) ([]*models.NestedFrozenDisks, diag.Diagnostics) {
	cloneFromTemplate := d.Get("create_effect.0.rebuild_from_snapshot").(string)
	gp := vm_snapshot.NewGetVMSnapshotsParams()
	gp.RequestBody = &models.GetVMSnapshotsRequestBody{
		Where: &models.VMSnapshotWhereInput{
			ID: &cloneFromTemplate,
		},
	}
	snapshots, err := ct.Api.VMSnapshot.GetVMSnapshots(gp)
	if err != nil {
		return nil, diag.FromErr(err)
	} else if len(snapshots.Payload) == 0 {
		return nil, diag.FromErr(fmt.Errorf("snapshot %s not found", cloneFromTemplate))
	}
	return snapshots.Payload[0].VMDisks, nil
}
func readCdRoms(ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) ([]*models.VMDisk, diag.Diagnostics) {
	id := d.Id()
	gp := vm_disk.NewGetVMDisksParams()
	typePt := models.VMDiskTypeCDROM
	gp.RequestBody = &models.GetVMDisksRequestBody{
		Where: &models.VMDiskWhereInput{
			Type: &typePt,
			VM: &models.VMWhereInput{
				ID: &id,
			},
		},
	}
	cdRoms, err := ct.Api.VMDisk.GetVMDisks(gp)

	if err != nil {
		return nil, diag.FromErr(err)
	}
	return cdRoms.Payload, nil
}

func derefAny(v interface{}, fallback interface{}) interface{} {
	ptr := reflect.ValueOf(v)
	if ptr.Kind() == reflect.Ptr {
		if ptr.IsNil() {
			return fallback
		}
		val := ptr.Elem().Interface()
		return val
	}
	return v
}

type VmCreateCommon struct {
	basic               *VmBasicConfig
	clusterId           *string
	firmware            *models.VMFirmware
	status              *VmStatusConfig
	guestOsType         *models.VMGuestsOperationSystem
	cdRoms              []*models.VMCdRomParams
	mountDisks          []*models.MountDisksParams
	mountNewCreateDisks []*models.MountNewCreateDisksParams
	vmNics              []*models.VMNicParams
}

// preprocess common create params for vm create from schema
// including vm basic, cluster, status, guest_os_type, disks and nics
func preprocessVmCreateCommon(ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) (*VmCreateCommon, diag.Diagnostics) {
	basic, err := expandVmBasicConfig(d)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	var clusterId *string = nil
	_clusterId, ok := d.GetOk("cluster_id")
	if ok {
		_clusterId := _clusterId.(string)
		clusterId = &_clusterId
	}
	var firmware *models.VMFirmware = nil
	switch d.Get("firmware").(string) {
	case "BIOS":
		firmware = models.VMFirmwareBIOS.Pointer()
	case "UEFI":
		firmware = models.VMFirmwareUEFI.Pointer()
	}
	status, err := expandVmStatusConfig(d)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	var guestOsType *models.VMGuestsOperationSystem = nil
	switch d.Get("guest_os_type").(string) {
	case "LINUX":
		guestOsType = models.VMGuestsOperationSystemLINUX.Pointer()
	case "WINDOWS":
		guestOsType = models.VMGuestsOperationSystemWINDOWS.Pointer()
	default:
		guestOsType = models.VMGuestsOperationSystemUNKNOWN.Pointer()
	}
	var disks []*VmDisk
	if rawDisk, ok := d.GetOk("disk"); ok && rawDisk != "" {
		bytes, err := json.Marshal(rawDisk)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		err = json.Unmarshal(bytes, &disks)
		if err != nil {
			return nil, diag.FromErr(err)
		}
	}

	var cdRoms []*models.VMCdRomParams
	var _cdRoms []*CdRom
	if rawCdRom, ok := d.GetOk("cd_rom"); ok && rawCdRom != "" {
		bytes, err := json.Marshal(rawCdRom)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		err = json.Unmarshal(bytes, &_cdRoms)
		for _, cdRom := range _cdRoms {
			params := &models.VMCdRomParams{
				Boot: &cdRom.Boot,
				// Index: &cdRom.Boot,
			}
			if cdRom.IsoId != "" {
				params.ElfImageID = &cdRom.IsoId
			}
			cdRoms = append(cdRoms, params)
		}
		if err != nil {
			return nil, diag.FromErr(err)
		}
	}

	var nics []*VmNic
	var vmNics []*models.VMNicParams
	if rawNic, ok := d.GetOk("nic"); ok && rawNic != "" {
		bytes, err := json.Marshal(d.Get("nic"))
		if err != nil {
			return nil, diag.FromErr(err)
		}
		err = json.Unmarshal(bytes, &nics)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		for i, nic := range nics {
			params := &models.VMNicParams{
				ConnectVlanID: &nic.VlanId,
				Mirror:        nic.Mirror,
			}
			if !*nic.Enabled {
				_, ok := d.GetOkExists(fmt.Sprintf("nic.%d.enabled", i))
				if !ok {
					_enabled := true
					nic.Enabled = &_enabled
				} else {
					params.Enabled = nic.Enabled
				}
			} else {
				params.Enabled = nic.Enabled
			}
			if *nic.Model != "" {
				params.Model = nic.Model
			}
			if *nic.Gateway != "" {
				params.Gateway = nic.Gateway
			}
			if *nic.IPAddress != "" {
				params.IPAddress = nic.IPAddress
			}
			if *nic.MacAddress != "" {
				params.MacAddress = nic.MacAddress
			}
			if *nic.SubnetMask != "" {
				params.SubnetMask = nic.SubnetMask
			}
			vmNics = append(vmNics, params)
		}
	}

	mountDisks := make([]*models.MountDisksParams, 0)
	mountNewCreateDisks := make([]*models.MountNewCreateDisksParams, 0)
	for _, disk := range disks {
		boot := int32(disk.Boot)
		if disk.VmVolumeId != "" {
			params := &models.MountDisksParams{
				Boot:       &boot,
				Bus:        &disk.Bus,
				VMVolumeID: &disk.VmVolumeId,
				// Index:      &boot,
			}
			mountDisks = append(mountDisks, params)
		} else if disk.VmVolume != nil && len(disk.VmVolume) == 1 {
			volume := disk.VmVolume[0]
			params := &models.MountNewCreateDisksParams{
				Boot: &boot,
				Bus:  &disk.Bus,
				VMVolume: &models.MountNewCreateDisksParamsVMVolume{
					ElfStoragePolicy: &volume.StoragePolicy,
					Name:             &volume.Name,
					Size:             &volume.Size,
				},
				// Index: &boot,
			}
			if volume.OriginPath != "" {
				params.VMVolume.Path = &volume.OriginPath
			}
			mountNewCreateDisks = append(mountNewCreateDisks, params)
		}
	}
	return &VmCreateCommon{
		basic:               basic,
		clusterId:           clusterId,
		firmware:            firmware,
		status:              status,
		guestOsType:         guestOsType,
		cdRoms:              cdRoms,
		mountDisks:          mountDisks,
		mountNewCreateDisks: mountNewCreateDisks,
		vmNics:              vmNics,
	}, nil
}

func rebuildVmFromSnapshot(rebuildFrom string, ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) ([]*models.WithTaskVM, diag.Diagnostics) {
	common, diags := preprocessVmCreateCommon(ctx, d, ct)
	if diags != nil {
		return nil, diags
	}
	rp := vm.NewRebuildVMParams()
	vmDisks, diags := readVmDisksFromSnapshot(ctx, d, ct)
	if diags != nil {
		return nil, diags
	}
	var indexPathMap = make(map[string]int32)
	for i, nfd := range vmDisks {
		indexPathMap[*nfd.Path] = int32(i)
	}
	for _, mncdp := range common.mountNewCreateDisks {
		// if the path is from an existed volumn, set its index
		if mncdp.VMVolume.Path != nil {
			if index, ok := indexPathMap[*mncdp.VMVolume.Path]; ok {
				mncdp.Index = &index
			}
		}

	}
	var diskParams *models.VMDiskParams = nil
	if len(common.cdRoms)+len(common.mountNewCreateDisks)+len(common.mountDisks) > 0 {
		diskParams = &models.VMDiskParams{
			MountCdRoms:         common.cdRoms,
			MountDisks:          common.mountDisks,
			MountNewCreateDisks: common.mountNewCreateDisks,
		}
	}
	rp.RequestBody = []*models.VMRebuildParams{
		{
			Name:                  &common.basic.Name,
			ClusterID:             common.clusterId,
			Vcpu:                  common.basic.Vcpu,
			Memory:                common.basic.Memory,
			Ha:                    common.basic.Ha,
			Firmware:              common.firmware,
			Status:                common.status.Status,
			HostID:                common.basic.HostId,
			FolderID:              common.basic.FolderId,
			Description:           common.basic.Description,
			GuestOsType:           common.guestOsType,
			CPUCores:              common.basic.CpuCores,
			CPUSockets:            common.basic.CpuSockets,
			VMDisks:               diskParams,
			VMNics:                common.vmNics,
			RebuildFromSnapshotID: &rebuildFrom,
		},
	}
	response, err := ct.Api.VM.RebuildVM(rp)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return response.Payload, nil
}

func cloneVmFromSourceVm(cloneFrom string, ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) ([]*models.WithTaskVM, diag.Diagnostics) {
	common, diags := preprocessVmCreateCommon(ctx, d, ct)
	if diags != nil {
		return nil, diags
	}
	cp := vm.NewCloneVMParams()
	vmDisks, vmVolumes, diags := readVmDisks(ctx, d, ct)
	var pathVolumeMap = make(map[string]*models.VMVolume)
	for _, v := range vmVolumes {
		pathVolumeMap[*v.Path] = v
	}
	if diags != nil {
		return nil, diags
	}
	var indexVolumeIdMap = make(map[string]int32)
	for i, nfd := range vmDisks {
		if *nfd.Type == models.VMDiskTypeDISK {
			indexVolumeIdMap[*nfd.VMVolume.ID] = int32(i)
		}
	}
	for _, mncdp := range common.mountNewCreateDisks {
		// if the path is from an existed volumn, set its index
		if mncdp.VMVolume.Path != nil {
			if volume, ok := pathVolumeMap[*mncdp.VMVolume.Path]; ok {
				if index, ok := indexVolumeIdMap[*volume.ID]; ok {
					mncdp.Index = &index
				}
			}
		}

	}
	var diskParams *models.VMDiskParams = nil
	if len(common.cdRoms)+len(common.mountDisks)+len(common.mountNewCreateDisks) > 0 {
		diskParams = &models.VMDiskParams{
			MountCdRoms:         common.cdRoms,
			MountDisks:          common.mountDisks,
			MountNewCreateDisks: common.mountNewCreateDisks,
		}
	}
	cp.RequestBody = []*models.VMCloneParams{
		{
			Name:        &common.basic.Name,
			ClusterID:   common.clusterId,
			Vcpu:        common.basic.Vcpu,
			Memory:      common.basic.Memory,
			Ha:          common.basic.Ha,
			Firmware:    common.firmware,
			Status:      common.status.Status,
			HostID:      common.basic.HostId,
			FolderID:    common.basic.FolderId,
			Description: common.basic.Description,
			GuestOsType: common.guestOsType,
			CPUCores:    common.basic.CpuCores,
			CPUSockets:  common.basic.CpuSockets,
			VMDisks:     diskParams,
			VMNics:      common.vmNics,
			SrcVMID:     &cloneFrom,
		},
	}
	response, err := ct.Api.VM.CloneVM(cp)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	return response.Payload, nil
}

func cloneVmFromVmTemplate(cloneFromTemplate string, ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) ([]*models.WithTaskVM, diag.Diagnostics) {
	common, diags := preprocessVmCreateCommon(ctx, d, ct)
	if diags != nil {
		return nil, diags
	}
	isFullCopyRes, ok := d.GetOkExists("create_effect.0.is_full_copy")
	if !ok {
		return nil, diag.FromErr(fmt.Errorf("when create from template, please set is_full_copy"))
	}
	isFullCopy := isFullCopyRes.(bool)
	cvft := vm.NewCreateVMFromTemplateParams()
	cloudInit, err := expandCloudInitConfig(d)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	var diskParams *models.VMDiskParams = nil
	removeIndex := make([]int32, 0)
	if len(common.cdRoms)+len(common.mountDisks)+len(common.mountNewCreateDisks) > 0 {
		vmDisks, diags := readVmDisksFromTemplate(ctx, d, ct)
		if diags != nil {
			return nil, diags
		}
		var indexPathMap = make(map[string]int32)
		for i, nfd := range vmDisks {
			indexPathMap[*nfd.Path] = int32(i)
		}
		for _, mncdp := range common.mountNewCreateDisks {
			// if the path is from an existed volumn, set its index
			if mncdp.VMVolume.Path != nil {
				if index, ok := indexPathMap[*mncdp.VMVolume.Path]; ok {
					mncdp.Index = &index
				}
			}
		}
		for idx := range vmDisks {
			removeIndex = append(removeIndex, int32(idx))
		}
		diskParams = &models.VMDiskParams{
			MountCdRoms:         common.cdRoms,
			MountDisks:          common.mountDisks,
			MountNewCreateDisks: common.mountNewCreateDisks,
		}
	}
	cvft.RequestBody = []*models.VMCreateVMFromTemplateParams{
		{
			IsFullCopy:  &isFullCopy,
			Status:      common.status.Status,
			Name:        &common.basic.Name,
			ClusterID:   common.clusterId,
			HostID:      common.basic.HostId,
			Description: common.basic.Description,
			Vcpu:        common.basic.Vcpu,
			FolderID:    common.basic.FolderId,
			GuestOsType: common.guestOsType,
			Memory:      common.basic.Memory,
			CPUCores:    common.basic.CpuCores,
			CPUSockets:  common.basic.CpuSockets,
			Ha:          common.basic.Ha,
			Firmware:    common.firmware,
			TemplateID:  &cloneFromTemplate,
			VMNics:      common.vmNics,
			CloudInit:   cloudInit,
			DiskOperate: &models.VMDiskOperate{
				NewDisks: diskParams,
				RemoveDisks: &models.VMDiskOperateRemoveDisks{
					DiskIndex: removeIndex,
				},
			},
		},
	}
	response, err := ct.Api.VM.CreateVMFromTemplate(cvft)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	return response.Payload, nil
}

func cloneVmFromContentLibraryVmTemplate(cloneFromContentLibraryTemplate string, ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) ([]*models.WithTaskVM, diag.Diagnostics) {
	common, diags := preprocessVmCreateCommon(ctx, d, ct)
	if diags != nil {
		return nil, diags
	}
	isFullCopyRes, ok := d.GetOkExists("create_effect.0.is_full_copy")
	if !ok {
		return nil, diag.FromErr(fmt.Errorf("when create from template, please set is_full_copy"))
	}
	isFullCopy := isFullCopyRes.(bool)
	cvft := vm.NewCreateVMFromContentLibraryTemplateParams()
	cloudInit, err := expandCloudInitConfig(d)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	var diskParams *models.VMDiskParams = nil
	removeIndex := make([]int32, 0)
	if len(common.cdRoms)+len(common.mountDisks)+len(common.mountNewCreateDisks) > 0 {
		vmDisks, diags := readVmDisksFromContentLibraryTemplate(ctx, d, ct)
		if diags != nil {
			return nil, diags
		}
		var indexPathMap = make(map[string]int32)
		for i, nfd := range vmDisks {
			indexPathMap[*nfd.Path] = int32(i)
		}
		for _, mncdp := range common.mountNewCreateDisks {
			// if the path is from an existed volumn, set its index
			if mncdp.VMVolume.Path != nil {
				if index, ok := indexPathMap[*mncdp.VMVolume.Path]; ok {
					mncdp.Index = &index
				}
			}
		}
		for idx := range vmDisks {
			removeIndex = append(removeIndex, int32(idx))
		}
		diskParams = &models.VMDiskParams{
			MountCdRoms:         common.cdRoms,
			MountDisks:          common.mountDisks,
			MountNewCreateDisks: common.mountNewCreateDisks,
		}
	}
	cvft.RequestBody = []*models.VMCreateVMFromContentLibraryTemplateParams{
		{
			IsFullCopy:  &isFullCopy,
			Status:      common.status.Status,
			Name:        &common.basic.Name,
			ClusterID:   common.clusterId,
			HostID:      common.basic.HostId,
			Description: common.basic.Description,
			Vcpu:        common.basic.Vcpu,
			FolderID:    common.basic.FolderId,
			GuestOsType: common.guestOsType,
			Memory:      common.basic.Memory,
			CPUCores:    common.basic.CpuCores,
			CPUSockets:  common.basic.CpuSockets,
			Ha:          common.basic.Ha,
			Firmware:    common.firmware,
			TemplateID:  &cloneFromContentLibraryTemplate,
			VMNics:      common.vmNics,
			CloudInit:   cloudInit,
			DiskOperate: &models.VMDiskOperate{
				NewDisks: diskParams,
				RemoveDisks: &models.VMDiskOperateRemoveDisks{
					DiskIndex: removeIndex,
				},
			},
		},
	}
	response, err := ct.Api.VM.CreateVMFromContentLibraryTemplate(cvft)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	return response.Payload, nil
}

func createBlankVm(ctx context.Context, d *schema.ResourceData, ct *cloudtower.Client) ([]*models.WithTaskVM, diag.Diagnostics) {
	common, diags := preprocessVmCreateCommon(ctx, d, ct)
	if diags != nil {
		return nil, diags
	}
	cvp := vm.NewCreateVMParams()
	// check status
	missingFields := make([]string, 0)
	if common.basic.Vcpu == nil {
		missingFields = append(missingFields, "vcpu")
	}
	if common.basic.Ha == nil {
		missingFields = append(missingFields, "ha")
	}
	if common.basic.Memory == nil {
		missingFields = append(missingFields, "memory")
	}
	if common.clusterId == nil {
		missingFields = append(missingFields, "cluster_id")
	}
	if common.status.Status == nil {
		missingFields = append(missingFields, "status")
	}
	if common.firmware == nil {
		missingFields = append(missingFields, "firmware")
	}
	if len(missingFields) > 0 {
		return nil, diag.Errorf("Simple create vm need more config, missing fields: %v", missingFields)
	}
	if common.basic.CpuCores == nil {
		var core int32 = 1
		common.basic.CpuCores = &core
	}
	if common.basic.CpuSockets == nil {
		socket := *common.basic.Vcpu / *common.basic.CpuCores
		common.basic.CpuSockets = &socket
	}
	cvp.RequestBody = []*models.VMCreationParams{{
		Name:        &common.basic.Name,
		ClusterID:   common.clusterId,
		Vcpu:        common.basic.Vcpu,
		Memory:      common.basic.Memory,
		Ha:          common.basic.Ha,
		Firmware:    common.firmware,
		Status:      common.status.Status,
		HostID:      common.basic.HostId,
		FolderID:    common.basic.FolderId,
		Description: common.basic.Description,
		GuestOsType: common.guestOsType,
		CPUCores:    common.basic.CpuCores,
		CPUSockets:  common.basic.CpuSockets,
		VMDisks: &models.VMDiskParams{
			MountCdRoms:         common.cdRoms,
			MountDisks:          common.mountDisks,
			MountNewCreateDisks: common.mountNewCreateDisks,
		},
		VMNics: common.vmNics,
	}}
	response, err := ct.Api.VM.CreateVM(cvp)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	return response.Payload, nil
}
