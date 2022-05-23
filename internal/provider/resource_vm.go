package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_disk"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_nic"
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
				Description: "VM's cluster id",
			},
			"vcpu": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "VM's vcpu",
			},
			"memory": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Description: "VM's memory, in the unit of byte",
			},
			"ha": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "whether VM is HA or not",
			},
			"firmware": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "VM's firmware",
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
										Optional:    true,
										Computed:    true,
										Description: "the VM volume's iscsi LUN path",
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
							Required:    true,
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
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"clone_from_vm": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "Id of source vm from created vm to be cloned from",
						},
						"rebuild_from_snapshot": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "Id of snapshot for created vm to be rebuilt from",
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

func resourceVmCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)

	basic, err := expandVmBasicConfig(d)
	if err != nil {
		return diag.FromErr(err)
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
		return diag.FromErr(err)
	}
	var guestOsType *models.VMGuestsOperationSystem = nil
	switch d.Get("guest_os_type").(string) {
	case "LINUX":
		guestOsType = models.VMGuestsOperationSystemLINUX.Pointer()
	case "WINDOWS":
		guestOsType = models.VMGuestsOperationSystemWINDOWS.Pointer()
	case "UNKNOWN":
		guestOsType = models.VMGuestsOperationSystemUNKNOWN.Pointer()
	}
	var disks []*VmDisk
	if rawDisk, ok := d.GetOk("disk"); ok && rawDisk != "" {
		bytes, err := json.Marshal(rawDisk)
		if err != nil {
			return diag.FromErr(err)
		}
		err = json.Unmarshal(bytes, &disks)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	var cdRoms []*models.VMCdRomParams
	var _cdRoms []*CdRom
	if rawCdRom, ok := d.GetOk("cd_rom"); ok && rawCdRom != "" {
		bytes, err := json.Marshal(rawCdRom)
		if err != nil {
			return diag.FromErr(err)
		}
		err = json.Unmarshal(bytes, &_cdRoms)
		for _, cdRom := range _cdRoms {
			params := &models.VMCdRomParams{
				Boot:  &cdRom.Boot,
				Index: &cdRom.Boot,
			}
			if cdRom.IsoId != "" {
				params.ElfImageID = &cdRom.IsoId
			}
			cdRoms = append(cdRoms, params)
		}
		if err != nil {
			return diag.FromErr(err)
		}
	}

	var nics []*VmNic
	var vmNics []*models.VMNicParams
	if rawNic, ok := d.GetOk("nic"); ok && rawNic != "" {
		bytes, err := json.Marshal(d.Get("nic"))
		if err != nil {
			return diag.FromErr(err)
		}
		err = json.Unmarshal(bytes, &nics)
		if err != nil {
			return diag.FromErr(err)
		}
		for _, nic := range nics {
			params := &models.VMNicParams{
				ConnectVlanID: &nic.VlanId,
				//FIXME: not to set 0 value to boolean
				Enabled: nic.Enabled,
				Mirror:  nic.Mirror,
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
			mountDisks = append(mountDisks, &models.MountDisksParams{
				Boot:       &boot,
				Bus:        &disk.Bus,
				VMVolumeID: &disk.VmVolumeId,
				Index:      &boot,
			})
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
				Index: &boot,
			}
			if volume.Path != "" {
				params.VMVolume.Path = &volume.Path
			}
			mountNewCreateDisks = append(mountNewCreateDisks, params)
		}
	}
	var vms []*models.WithTaskVM
	rebuildFrom := d.Get("create_effect.0.rebuild_from_snapshot").(string)
	cloneFrom := d.Get("create_effect.0.clone_from_vm").(string)
	if rebuildFrom != "" && cloneFrom != "" {
		return diag.FromErr(fmt.Errorf("rebuild_from_snapshot and clone_from_vm can not be set at the same time"))
	} else if rebuildFrom != "" {
		// rebuild from target snapshot
		rp := vm.NewRebuildVMParams()
		rp.RequestBody = []*models.VMRebuildParams{
			{
				Name:        &basic.Name,
				ClusterID:   clusterId,
				Vcpu:        basic.Vcpu,
				Memory:      basic.Memory,
				Ha:          basic.Ha,
				Firmware:    firmware,
				Status:      status.Status,
				HostID:      basic.HostId,
				FolderID:    basic.FolderId,
				Description: basic.Description,
				GuestOsType: guestOsType,
				CPUCores:    basic.CpuCores,
				CPUSockets:  basic.CpuSockets,
				VMDisks: &models.VMDiskParams{
					MountCdRoms:         cdRoms,
					MountDisks:          mountDisks,
					MountNewCreateDisks: mountNewCreateDisks,
				},
				VMNics:                vmNics,
				RebuildFromSnapshotID: &rebuildFrom,
			},
		}
		response, err := ct.Api.VM.RebuildVM(rp)
		if err != nil {
			return diag.FromErr(err)
		}
		vms = response.Payload
	} else if cloneFrom != "" {
		cp := vm.NewCloneVMParams()
		var diskParams *models.VMDiskParams = nil
		if len(cdRoms)+len(mountDisks)+len(mountNewCreateDisks) > 0 {
			diskParams = &models.VMDiskParams{
				MountCdRoms:         cdRoms,
				MountDisks:          mountDisks,
				MountNewCreateDisks: mountNewCreateDisks,
			}
		}
		cp.RequestBody = []*models.VMCloneParams{
			{
				Name:        &basic.Name,
				ClusterID:   clusterId,
				Vcpu:        basic.Vcpu,
				Memory:      basic.Memory,
				Ha:          basic.Ha,
				Firmware:    firmware,
				Status:      status.Status,
				HostID:      basic.HostId,
				FolderID:    basic.FolderId,
				Description: basic.Description,
				GuestOsType: guestOsType,
				CPUCores:    basic.CpuCores,
				CPUSockets:  basic.CpuSockets,
				VMDisks:     diskParams,
				VMNics:      vmNics,
				SrcVMID:     &cloneFrom,
			},
		}
		response, err := ct.Api.VM.CloneVM(cp)
		if err != nil {
			return diag.FromErr(err)
		}
		vms = response.Payload
	} else {
		// normal create
		cvp := vm.NewCreateVMParams()
		// check status
		missingFields := make([]string, 0)
		if basic.Vcpu == nil {
			missingFields = append(missingFields, "vcpu")
		}
		if basic.Ha == nil {
			missingFields = append(missingFields, "ha")
		}
		if basic.Memory == nil {
			missingFields = append(missingFields, "memory")
		}
		if basic.HostId == nil {
			missingFields = append(missingFields, "host_id")
		}
		if clusterId == nil {
			missingFields = append(missingFields, "cluster_id")
		}
		if status.Status == nil {
			missingFields = append(missingFields, "status")
		}
		if firmware == nil {
			missingFields = append(missingFields, "firmware")
		}
		if len(missingFields) > 0 {
			return diag.Errorf("Simple create vm need more config, missing fields: %v", missingFields)
		}
		if basic.CpuCores == nil {
			var core int32 = 1
			basic.CpuCores = &core
		}
		if basic.CpuSockets == nil {
			socket := *basic.Vcpu / *basic.CpuCores
			basic.CpuSockets = &socket
		}
		cvp.RequestBody = []*models.VMCreationParams{{
			Name:        &basic.Name,
			ClusterID:   clusterId,
			Vcpu:        basic.Vcpu,
			Memory:      basic.Memory,
			Ha:          basic.Ha,
			Firmware:    firmware,
			Status:      status.Status,
			HostID:      basic.HostId,
			FolderID:    basic.FolderId,
			Description: basic.Description,
			GuestOsType: guestOsType,
			CPUCores:    basic.CpuCores,
			CPUSockets:  basic.CpuSockets,
			VMDisks: &models.VMDiskParams{
				MountCdRoms:         cdRoms,
				MountDisks:          mountDisks,
				MountNewCreateDisks: mountNewCreateDisks,
			},
			VMNics: vmNics,
		}}
		response, err := ct.Api.VM.CreateVM(cvp)
		if err != nil {
			return diag.FromErr(err)
		}
		vms = response.Payload
	}

	d.SetId(*vms[0].Data.ID)
	err = waitVmTasksFinish(ct, vms)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceVmRead(ctx, d, meta)
}

func resourceVmRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)

	v, diags := readVm(ctx, d, ct)
	if diags != nil {
		return diags
	}

	// if err := d.Set("name", v.Name); err != nil {
	// 	return diag.FromErr(err)
	// }
	if err := d.Set("host_id", v.Host.ID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", v.Status); err != nil {
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
	for idx, d := range vmDisks {
		vmVolume := vmVolumes[idx]
		vmVolumeData := map[string]interface{}{
			"id": d.VMVolume.ID,
		}
		if vmVolume != nil {
			vmVolumeData["name"] = vmVolume.Name
			vmVolumeData["size"] = vmVolume.Size
			vmVolumeData["path"] = vmVolume.Path
			vmVolumeData["storage_policy"] = vmVolume.ElfStoragePolicy
		}
		disks = append(disks, map[string]interface{}{
			"id":   d.ID,
			"boot": d.Boot,
			"bus":  d.Bus,
			"vm_volume": []map[string]interface{}{
				vmVolumeData,
			},
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

	if d.HasChanges("name", "vcpu", "memory", "description", "ha", "cpu_cores", "cpu_sockets") {
		basic, err := expandVmBasicConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		uvp := vm.NewUpdateVMParams()
		uvp.RequestBody = &models.VMUpdateParams{
			Where: &models.VMWhereInput{
				ID: &id,
			},
			Data: &models.VMUpdateParamsData{
				Name:        &basic.Name,
				Vcpu:        basic.Vcpu,
				Memory:      basic.Memory,
				Description: basic.Description,
				Ha:          basic.Ha,
				CPUCores:    basic.CpuCores,
				CPUSockets:  basic.CpuSockets,
			},
		}
		vms, err := ct.Api.VM.UpdateVM(uvp)
		if err != nil {
			return diag.FromErr(err)
		}
		err = waitVmTasksFinish(ct, vms.Payload)
		if err != nil {
			return diag.FromErr(err)
		}
	}

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

	if d.HasChange("host_id") {
		hostId := d.Get("host_id").(string)
		mvp := vm.NewMigRateVMParams()
		mvp.RequestBody = &models.VMMigrateParams{
			Where: &models.VMWhereInput{
				ID: &id,
			},
			Data: &models.VMMigrateParamsData{
				HostID: &hostId,
			},
		}
		vms, err := ct.Api.VM.MigRateVM(mvp)
		if err != nil {
			return diag.FromErr(err)
		}
		err = waitVmTasksFinish(ct, vms.Payload)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("nic") {
		vmNics, diags := readVmNics(ctx, d, ct)
		if diags != nil {
			return diags
		}
		curNicMap := make(map[string]*int, 0)
		for idx, n := range vmNics {
			_idx := idx
			curNicMap[*n.ID] = &_idx
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
		var adds []*models.VMNicParams
		var removes []int32
		enabled := false
		for _, n := range nics {
			if n.Id == "" {
				// new nic
				adds = append(adds, &models.VMNicParams{
					ConnectVlanID: &n.VlanId,
					Enabled:       n.Enabled,
					Gateway:       n.Gateway,
					IPAddress:     n.IPAddress,
					MacAddress:    n.MacAddress,
					Mirror:        n.Mirror,
					Model:         n.Model,
					SubnetMask:    n.SubnetMask,
				})
			} else if curNicMap[n.Id] != nil {
				srcN := vmNics[*curNicMap[n.Id]]
				// mark consumed
				delete(curNicMap, n.Id)
				if n.VlanId == derefAny(srcN.Vlan.ID, "") &&
					n.Enabled == derefAny(srcN.Enabled, false) &&
					n.Mirror == derefAny(srcN.Mirror, false) &&
					n.Model == derefAny(*srcN.Model, "") {
					continue
				}
				// update nic
				idx := int32(*curNicMap[n.Id])
				p := vm.NewUpdateVMNicParams()
				p.RequestBody = &models.VMUpdateNicParams{
					Where: &models.VMWhereInput{
						ID: &id,
					},
					Data: &models.VMUpdateNicParamsData{
						ConnectVlanID: &n.VlanId,
						Enabled:       &enabled,
						Gateway:       n.Gateway,
						IPAddress:     n.IPAddress,
						MacAddress:    n.MacAddress,
						Mirror:        n.Mirror,
						Model:         n.Model,
						SubnetMask:    n.SubnetMask,
						NicID:         &n.Id,
						NicIndex:      &idx,
					},
				}
				vms, err := ct.Api.VM.UpdateVMNic(p)
				if err != nil {
					return diag.FromErr(err)
				}
				err = waitVmTasksFinish(ct, vms.Payload)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		}
		for _, v := range curNicMap {
			removeIdx := int32(*v)
			removes = append(removes, removeIdx)
		}
		if len(removes) > 0 {
			p := vm.NewRemoveVMNicParams()
			p.RequestBody = &models.VMRemoveNicParams{
				Where: &models.VMWhereInput{
					ID: &id,
				},
				Data: &models.VMRemoveNicParamsData{
					NicIndex: removes,
				},
			}
			vms, err := ct.Api.VM.RemoveVMNic(p)
			if err != nil {
				return diag.FromErr(err)
			}
			err = waitVmTasksFinish(ct, vms.Payload)
			if err != nil {
				return diag.FromErr(err)
			}
		}
		if len(adds) > 0 {
			p := vm.NewAddVMNicParams()
			p.RequestBody = &models.VMAddNicParams{
				Where: &models.VMWhereInput{
					ID: &id,
				},
				Data: &models.VMAddNicParamsData{
					VMNics: adds,
				},
			}
			vms, err := ct.Api.VM.AddVMNic(p)
			if err != nil {
				return diag.FromErr(err)
			}
			err = waitVmTasksFinish(ct, vms.Payload)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("cd_rom") {
		cdRoms, diags := readCdRoms(ctx, d, ct)
		if diags != nil {
			return diags
		}
		curMap := make(map[string]*int, 0)
		for idx, v := range cdRoms {
			_idx := idx
			curMap[*v.ID] = &_idx
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

		var adds []*models.VMCdRomParams
		var removeIds []string
		for _, v := range cdRomsData {
			if v.Id == "" {
				// new cd-rom
				adds = append(adds, &models.VMCdRomParams{
					Boot:       &v.Boot,
					ElfImageID: &v.IsoId,
					Index:      &v.Boot,
				})
			} else if curMap[v.Id] != nil {
				srcV := cdRoms[*curMap[v.Id]]
				// mark consumed
				delete(curMap, v.Id)
				var srcIsoId interface{}
				if srcV.ElfImage == nil {
					srcIsoId = ""
				} else {
					srcIsoId = derefAny(srcV.ElfImage.ID, "")
				}
				if v.IsoId == srcIsoId {
					continue
				}
				// update cd-rom
				p := vm.NewUpdateVMDiskParams()
				var elfImageId *string
				if v.IsoId != "" {
					elfImageId = &v.IsoId
				}
				p.RequestBody = &models.VMUpdateDiskParams{
					Where: &models.VMWhereInput{
						ID: &id,
					},
					Data: &models.VMUpdateDiskParamsData{
						VMDiskID:   &v.Id,
						ElfImageID: elfImageId,
					},
				}
				vms, err := ct.Api.VM.UpdateVMDisk(p)
				if err != nil {
					return diag.FromErr(err)
				}
				err = waitVmTasksFinish(ct, vms.Payload)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		}
		for k := range curMap {
			removeIds = append(removeIds, k)
		}
		if len(removeIds) > 0 {
			p := vm.NewRemoveVMCdRomParams()
			p.RequestBody = &models.VMRemoveCdRomParams{
				Where: &models.VMWhereInput{
					ID: &id,
				},
				Data: &models.VMRemoveCdRomParamsData{
					CdRomIds: removeIds,
				},
			}
			vms, err := ct.Api.VM.RemoveVMCdRom(p)
			if err != nil {
				return diag.FromErr(err)
			}
			err = waitVmTasksFinish(ct, vms.Payload)
			if err != nil {
				return diag.FromErr(err)
			}
		}
		if len(adds) > 0 {
			p := vm.NewAddVMCdRomParams()
			p.RequestBody = &models.VMAddCdRomParams{
				Where: &models.VMWhereInput{
					ID: &id,
				},
				Data: &models.VMAddCdRomParamsData{
					VMCdRoms: adds,
				},
			}
			vms, err := ct.Api.VM.AddVMCdRom(p)
			if err != nil {
				return diag.FromErr(err)
			}
			err = waitVmTasksFinish(ct, vms.Payload)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChange("disk") {
		vmDisks, _, diags := readVmDisks(ctx, d, ct)
		if diags != nil {
			return diags
		}
		curMap := make(map[string]*int, 0)
		for idx, v := range vmDisks {
			_idx := idx
			curMap[*v.ID] = &_idx
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

		mountDisks := make([]*models.MountDisksParams, 0)
		mountNewCreateDisks := make([]*models.MountNewCreateDisksParams, 0)
		var removeIds []string
		for _, v := range disks {
			if v.Id == "" {
				// new disk
				// TODO: reuse with create context
				boot := int32(v.Boot)
				if v.VmVolumeId != "" {
					mountDisks = append(mountDisks, &models.MountDisksParams{
						Boot:       &boot,
						Bus:        &v.Bus,
						VMVolumeID: &v.VmVolumeId,
						Index:      &boot,
					})
				} else if v.VmVolume != nil && len(v.VmVolume) == 1 {
					volume := v.VmVolume
					mountNewCreateDisks = append(mountNewCreateDisks, &models.MountNewCreateDisksParams{
						Boot: &boot,
						Bus:  &v.Bus,
						VMVolume: &models.MountNewCreateDisksParamsVMVolume{
							ElfStoragePolicy: &volume[0].StoragePolicy,
							Name:             &volume[0].Name,
							Size:             &volume[0].Size,
							Path:             &volume[0].Path,
						},
						Index: &boot,
					})
				}
			} else if curMap[v.Id] != nil {
				// TODO: support update vm disk
				delete(curMap, v.Id)
			}
		}
		for k := range curMap {
			removeIds = append(removeIds, k)
		}
		if len(removeIds) > 0 {
			p := vm.NewRemoveVMDiskParams()
			p.RequestBody = &models.VMRemoveDiskParams{
				Where: &models.VMWhereInput{
					ID: &id,
				},
				Data: &models.VMRemoveDiskParamsData{
					DiskIds: removeIds,
				},
			}
			vms, err := ct.Api.VM.RemoveVMDisk(p)
			if err != nil {
				return diag.FromErr(err)
			}
			err = waitVmTasksFinish(ct, vms.Payload)
			if err != nil {
				return diag.FromErr(err)
			}
		}
		if len(mountDisks)+len(mountNewCreateDisks) > 0 {
			p := vm.NewAddVMDiskParams()
			p.RequestBody = &models.VMAddDiskParams{
				Where: &models.VMWhereInput{
					ID: &id,
				},
				Data: &models.VMAddDiskParamsData{
					VMDisks: &models.VMAddDiskParamsDataVMDisks{
						MountDisks:          mountDisks,
						MountNewCreateDisks: mountNewCreateDisks,
					},
				},
			}
			vms, err := ct.Api.VM.AddVMDisk(p)
			if err != nil {
				return diag.FromErr(err)
			}
			err = waitVmTasksFinish(ct, vms.Payload)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return resourceVmRead(ctx, d, meta)
}

func resourceVmDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)
	dvp := vm.NewDeleteVMParams()
	id := d.Id()
	dvp.RequestBody = &models.VMOperateParams{
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
		basicConfig.HostId = &folderId
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
		basicConfig.CpuCores = &cpuSockets
	}
	return basicConfig, nil
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
		},
	}
	vmDisks, err := ct.Api.VMDisk.GetVMDisks(gp)
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
	vmVolumeMap := make(map[string]*models.VMVolume, 0)
	for _, v := range vmVolumes.Payload {
		vmVolumeMap[*v.ID] = v
	}
	vmVolumesSlice := make([]*models.VMVolume, len(vmDisks.Payload))
	for idx, v := range vmDisks.Payload {
		vmVolume := vmVolumeMap[*v.VMVolume.ID]
		vmVolumesSlice[idx] = vmVolume
	}

	if err != nil {
		return nil, nil, diag.FromErr(err)
	}
	return vmDisks.Payload, vmVolumesSlice, nil
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
