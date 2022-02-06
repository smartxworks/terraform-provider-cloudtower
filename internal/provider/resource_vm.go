package provider

import (
	"context"
	"encoding/json"
	"github.com/Yuyz0112/cloudtower-go-sdk/client/operations"
	"github.com/Yuyz0112/cloudtower-go-sdk/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
				Type:     schema.TypeString,
				Required: true,
			},
			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"vcpu": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"memory": {
				Type:     schema.TypeFloat,
				Required: true,
			},
			"ha": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"firmware": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"BIOS", "UEFI"}, false),
			},
			"status": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"RUNNING", "STOPPED", "SUSPENDED"}, false),
			},
			"force_status_change": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"disk": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"boot": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"bus": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"IDE", "SCSI", "VIRTIO"}, false),
						},
						"vm_volume_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"vm_volume": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"storage_policy": {
										Type:     schema.TypeString,
										Required: true,
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
										Type:     schema.TypeString,
										Required: true,
									},
									"size": {
										Type:     schema.TypeFloat,
										Required: true,
									},
									"path": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			"cd_rom": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"boot": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"elf_image_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"nic": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vlan_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"mirror": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"model": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"E1000", "SRIOV", "VIRTIO"}, false),
						},
						"mac_address": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ip_address": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"subnet_mask": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"gateway": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"host_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"folder_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"guest_os_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"LINUX", "WINDOWS", "UNKNOWN"}, false),
			},
			"cpu_cores": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"cpu_sockets": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

type VmDisk struct {
	Boot       int                `json:"boot"`
	Bus        models.Bus         `json:"bus"`
	VmVolumeId *string            `json:"vm_volume_id"`
	VmVolume   *[]*VmDiskVmVolume `json:"vm_volume"`
}

type VmDiskVmVolume struct {
	StoragePolicy models.VMVolumeElfStoragePolicyType `json:"storage_policy"`
	Name          string                              `json:"name"`
	Size          float64                             `json:"size"`
	Path          *string                             `json:"path"`
}

type VmNic struct {
	models.VMNicParamsItems0
	VlanId string `json:"vlan_id"`
}

func resourceVmCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	cvp := operations.NewCreateVMParams()
	basic, err := expandVmBasicConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}
	clusterId := d.Get("cluster_id").(string)
	var firmware models.VMFirmware
	switch d.Get("firmware").(string) {
	case "BIOS":
		firmware = models.VMFirmwareBIOS
	case "UEFI":
		firmware = models.VMFirmwareUEFI
	}
	status, err := expandVmStatusConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}
	var guestOsType models.VMGuestsOperationSystem
	switch d.Get("guest_os_type").(string) {
	case "LINUX":
		guestOsType = models.VMGuestsOperationSystemLINUX
	case "WINDOWS":
		guestOsType = models.VMGuestsOperationSystemWINDOWS
	case "UNKNOWN":
		guestOsType = models.VMGuestsOperationSystemUNKNOWN
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
	var cdRoms []*models.VMCdRomParamsItems0
	bytes, err = json.Marshal(d.Get("cd_rom"))
	if err != nil {
		return diag.FromErr(err)
	}
	err = json.Unmarshal(bytes, &cdRoms)
	for _, cdRom := range cdRoms {
		cdRom.Index = cdRom.Boot
	}
	if err != nil {
		return diag.FromErr(err)
	}
	var nics []*VmNic
	bytes, err = json.Marshal(d.Get("nic"))
	err = json.Unmarshal(bytes, &nics)
	if err != nil {
		return diag.FromErr(err)
	}
	var vmNics []*models.VMNicParamsItems0
	for _, nic := range nics {
		vmNics = append(vmNics, &models.VMNicParamsItems0{
			ConnectVlanID: &nic.VlanId,
			Enabled:       nic.Enabled,
			Gateway:       nic.Gateway,
			IPAddress:     nic.IPAddress,
			MacAddress:    nic.MacAddress,
			Mirror:        nic.Mirror,
			Model:         nic.Model,
			SubnetMask:    nic.SubnetMask,
		})
	}
	var mountDisks []*models.MountDisksParamsItems0
	var mountNewCreateDisks []*models.MountNewCreateDisksParamsItems0
	for _, disk := range disks {
		fBoot := float64(disk.Boot)
		if *disk.VmVolumeId != "" {
			mountDisks = append(mountDisks, &models.MountDisksParamsItems0{
				Boot:       &fBoot,
				Bus:        &disk.Bus,
				VMVolumeID: disk.VmVolumeId,
				Index:      &fBoot,
			})
		} else if disk.VmVolume != nil && len(*disk.VmVolume) == 1 {
			volume := *disk.VmVolume
			mountNewCreateDisks = append(mountNewCreateDisks, &models.MountNewCreateDisksParamsItems0{
				Boot: &fBoot,
				Bus:  &disk.Bus,
				VMVolume: &models.MountNewCreateDisksParamsItems0VMVolume{
					ElfStoragePolicy: &volume[0].StoragePolicy,
					Name:             &volume[0].Name,
					Size:             &volume[0].Size,
					Path:             *volume[0].Path,
				},
				Index: fBoot,
			})
		}
	}
	cvp.RequestBody = []*models.VMCreationParams{{
		Name:        &basic.Name,
		ClusterID:   &clusterId,
		Vcpu:        &basic.Vcpu,
		Memory:      &basic.Memory,
		Ha:          &basic.Ha,
		Firmware:    &firmware,
		Status:      &status.Status,
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
	vms, err := ct.Api.Operations.CreateVM(cvp)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(*vms.Payload[0].Data.ID)
	err = waitVmTasksFinish(ct, vms.Payload)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceVmRead(ctx, d, meta)
}

func resourceVmRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)

	id := d.Id()
	gcp := operations.NewGetVmsParams()
	gcp.RequestBody = &models.GetVmsRequestBody{
		Where: &models.VMWhereInput{
			ID: &id,
		},
	}
	vms, err := ct.Api.Operations.GetVms(gcp)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(vms.Payload) < 1 {
		d.SetId("")
		return diags
	}
	if err = d.Set("name", vms.Payload[0].Name); err != nil {
		return diag.FromErr(err)
	}
	if err = d.Set("host_id", vms.Payload[0].Host.ID); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceVmUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	id := d.Id()

	if d.HasChanges("name", "vcpu", "memory", "description", "ha", "cpu_cores", "cpu_sockets") {
		basic, err := expandVmBasicConfig(d)
		if err != nil {
			return diag.FromErr(err)
		}
		uvp := operations.NewUpdateVMParams()
		uvp.RequestBody = &models.VMUpdateParams{
			Where: &models.VMWhereInput{
				ID: &id,
			},
			Data: &models.VMUpdateParamsData{
				Name:        basic.Name,
				Vcpu:        basic.Vcpu,
				Memory:      basic.Memory,
				Description: basic.Description,
				Ha:          basic.Ha,
				CPUCores:    *basic.CpuCores,
				CPUSockets:  *basic.CpuSockets,
			},
		}
		vms, err := ct.Api.Operations.UpdateVM(uvp)
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
		switch status.Status {
		case models.VMStatusRUNNING:
			uvp := operations.NewStartVMParams()
			uvp.RequestBody = &models.VMStartParams{
				Where: &models.VMWhereInput{
					ID: &id,
				},
				Data: &models.VMStartParamsData{
					HostID: &basic.HostId,
				},
			}
			vms, err := ct.Api.Operations.StartVM(uvp)
			if err != nil {
				return diag.FromErr(err)
			}
			err = waitVmTasksFinish(ct, vms.Payload)
			if err != nil {
				return diag.FromErr(err)
			}
		case models.VMStatusSTOPPED:
			if status.Force {
				uvp := operations.NewForceShutDownVMParams()
				uvp.RequestBody = &models.VMOperateParams{
					Where: &models.VMWhereInput{
						ID: &id,
					},
				}
				vms, err := ct.Api.Operations.ForceShutDownVM(uvp)
				if err != nil {
					return diag.FromErr(err)
				}
				err = waitVmTasksFinish(ct, vms.Payload)
				if err != nil {
					return diag.FromErr(err)
				}
			} else {
				uvp := operations.NewShutDownVMParams()
				uvp.RequestBody = &models.VMOperateParams{
					Where: &models.VMWhereInput{
						ID: &id,
					},
				}
				vms, err := ct.Api.Operations.ShutDownVM(uvp)
				if err != nil {
					return diag.FromErr(err)
				}
				err = waitVmTasksFinish(ct, vms.Payload)
				if err != nil {
					return diag.FromErr(err)
				}
			}
		case models.VMStatusSUSPENDED:
			uvp := operations.NewSuspendVMParams()
			uvp.RequestBody = &models.VMOperateParams{
				Where: &models.VMWhereInput{
					ID: &id,
				},
			}
			vms, err := ct.Api.Operations.SuspendVM(uvp)
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
	dvp := operations.NewDeleteVMParams()
	id := d.Id()
	dvp.RequestBody = &models.VMOperateParams{
		Where: &models.VMWhereInput{
			ID: &id,
		},
	}
	vms, err := ct.Api.Operations.DeleteVM(dvp)
	if err != nil {
		return diag.FromErr(err)
	}
	taskIds := make([]string, 0)
	for _, vm := range vms.Payload {
		if vm.TaskID != nil {
			taskIds = append(taskIds, *vm.TaskID)
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
	for _, vm := range vms {
		if vm.TaskID != nil {
			taskIds = append(taskIds, *vm.TaskID)
		}
	}
	_, err := ct.WaitTasksFinish(taskIds)
	return err
}

type VmBasicConfig struct {
	Name        string
	Vcpu        float64
	Memory      float64
	Ha          bool
	HostId      string
	FolderId    string
	Description string
	CpuCores    *float64
	CpuSockets  *float64
}

func expandVmBasicConfig(d *schema.ResourceData) (*VmBasicConfig, error) {
	name := d.Get("name").(string)
	_vcpu := d.Get("vcpu").(int)
	vcpu := float64(_vcpu)
	memory := d.Get("memory").(float64)
	ha := d.Get("ha").(bool)
	hostId := d.Get("host_id").(string)
	folderId := d.Get("folder_id").(string)
	description := d.Get("description").(string)
	var cpuCores *float64
	if _cpuCores := d.Get("cpu_cores").(int); _cpuCores != 0 {
		f := float64(_cpuCores)
		cpuCores = &f
	} else {
		f := float64(1)
		cpuCores = &f
	}
	var cpuSockets *float64
	if _cpuSockets := d.Get("cpu_sockets").(int); _cpuSockets != 0 {
		f := float64(_cpuSockets)
		cpuSockets = &f
	} else {
		f := vcpu / *cpuCores
		cpuSockets = &f
	}

	return &VmBasicConfig{
		Name:        name,
		Vcpu:        vcpu,
		Memory:      memory,
		Ha:          ha,
		HostId:      hostId,
		FolderId:    folderId,
		Description: description,
		CpuCores:    cpuCores,
		CpuSockets:  cpuSockets,
	}, nil
}

type VmStatusConfig struct {
	Status models.VMStatus
	Force  bool
}

func expandVmStatusConfig(d *schema.ResourceData) (*VmStatusConfig, error) {
	var status models.VMStatus
	switch d.Get("status").(string) {
	case "RUNNING":
		status = models.VMStatusRUNNING
	case "STOPPED":
		status = models.VMStatusSTOPPED
	case "SUSPENDED":
		status = models.VMStatusSUSPENDED
	}
	force := d.Get("force_status_change").(bool)
	return &VmStatusConfig{
		Status: status,
		Force:  force,
	}, nil
}
