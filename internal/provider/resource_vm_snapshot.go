package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_snapshot"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVmSnapshot() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vm snapshot resource",

		CreateContext: resourceVmSnapshotCreate,
		ReadContext:   resourceVmSnapshotRead,
		DeleteContext: resourceVmSnapshotDelete,

		Schema: map[string]*schema.Schema{
			"vm_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The id of vm to the snapshot belongs to",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of snapshot",
			},
			"consistent_type": {
				Type:        schema.TypeString,
				Default:     string(models.ConsistentTypeCRASHCONSISTENT),
				ForceNew:    true,
				Optional:    true,
				Description: "The consistent type of snapshot",
				ValidateDiagFunc: func(v interface{}, _ cty.Path) diag.Diagnostics {
					var diags diag.Diagnostics
					val, ok := v.(string)
					if !ok {
						return append(diags, diag.Diagnostic{
							Severity: diag.Error,
							Summary:  "Wrong type",
							Detail:   "Consistent type should be a string",
						})
					} else if val != string(models.ConsistentTypeCRASHCONSISTENT) && val != string(models.ConsistentTypeFILESYSTEMCONSISTENT) {
						return append(diags, diag.Diagnostic{
							Severity: diag.Error,
							Summary:  "Invalid consistent type",
							Detail: fmt.Sprintf("Consistent type should be one of %v, but get %s",
								[]string{string(models.ConsistentTypeCRASHCONSISTENT), string(models.ConsistentTypeFILESYSTEMCONSISTENT)},
								val,
							),
						})
					}
					return diags
				},
			},
			"disks": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"boot": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "disk's boot order",
						},
						"bus": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "disk's bus",
						},
						"storage_policy": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "vm volume's storage policy",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "vm volume's name",
						},
						"size": {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "vm volume's size, in the unit of byte",
						},
						"path": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "vm volume's iscsi LUN path",
						},
					},
				},
				Computed:    true,
				Description: "template's disks",
			},
			"cd_roms": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"boot": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "cd-rom's boot order",
						},
						"elf_image_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Optional:    true,
							Description: "cd-rom's elf image id",
						},
						"svt_image_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Optional:    true,
							Description: "cd-rom's svt image id",
						},
					},
				},
				Computed:    true,
				Description: "template's cd_rom",
			},
			"nics": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vlan_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "specific the vlan's id the template nic is using",
						},
						"enabled": {
							Type:        schema.TypeBool,
							Description: "whether the template nic is enabled",
							Computed:    true,
						},
						"mirror": {
							Type:        schema.TypeBool,
							Description: "whether the template nic use mirror mode",
							Computed:    true,
						},
						"model": {
							Type:        schema.TypeString,
							Description: "template nic's model",
							Computed:    true,
						},
						"idx": {
							Type:        schema.TypeInt,
							Description: "template nic's index",
							Computed:    true,
						},
					},
				},
				Computed:    true,
				Description: "template's nics",
			},
		},
	}
}

func resourceVmSnapshotCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	cvsp := vm_snapshot.NewCreateVMSnapshotParams()
	consistent_type := models.ConsistentType(d.Get("consistent_type").(string))
	name := d.Get("name").(string)
	vm_id := d.Get("vm_id").(string)
	cvsp.RequestBody = &models.VMSnapshotCreationParams{
		Data: []*models.VMSnapshotCreationParamsData{
			{
				ConsistentType: &consistent_type,
				Name:           &name,
				VMID:           &vm_id,
			},
		},
	}
	snapshots, err := ct.Api.VMSnapshot.CreateVMSnapshot(cvsp)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(*snapshots.Payload[0].Data.ID)
	_, err = ct.WaitTasksFinish(ctx, []string{*snapshots.Payload[0].TaskID})
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceVmSnapshotRead(ctx, d, meta)
}

func resourceVmSnapshotRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)

	id := d.Id()
	gvsp := vm_snapshot.NewGetVMSnapshotsParams()
	gvsp.RequestBody = &models.GetVMSnapshotsRequestBody{
		Where: &models.VMSnapshotWhereInput{
			ID: &id,
		},
	}
	vmSnapshots, err := ct.Api.VMSnapshot.GetVMSnapshots(gvsp)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(vmSnapshots.Payload) < 1 {
		d.SetId("")
		return diags
	}
	snapshot := vmSnapshots.Payload[0]
	var disks []map[string]interface{} = make([]map[string]interface{}, 0)
	var cdroms []map[string]interface{} = make([]map[string]interface{}, 0)
	for _, disk := range snapshot.VMDisks {
		if *disk.Type == models.VMDiskTypeDISK {
			storagePolicy, err := helper.GetElfStoragePolicyByLocalId(ct.Api, *disk.StoragePolicyUUID)
			if err != nil {
				// return diag.FromErr(err)
				diags = append(diags, diag.FromErr(err)...)
				continue
			}
			disks = append(disks, map[string]interface{}{
				"boot":           disk.Boot,
				"bus":            disk.Bus,
				"storage_policy": storagePolicy,
				"name":           disk.DiskName,
				"size":           disk.Size,
				"path":           disk.Path,
			})
		} else if *disk.Type == models.VMDiskTypeCDROM {
			var elfImageId = ""
			var svtImageId = ""
			if disk.ElfImageLocalID != nil {
				elfImage, _ := helper.GetElfImageFromLocalId(ct.Api, *disk.ElfImageLocalID)
				if elfImage != nil {
					elfImageId = *elfImage.ID
				}
			}
			if disk.SvtImageLocalID != nil {
				svtImage, _ := helper.GetSvtIMageFromLocalId(ct.Api, *disk.SvtImageLocalID)
				if svtImage != nil {
					svtImageId = *svtImage.ID
				}
			}
			cdroms = append(cdroms, map[string]interface{}{
				"boot":         disk.Boot,
				"elf_image_id": elfImageId,
				"svt_image_id": svtImageId,
			})
		}
	}
	if err = d.Set("disks", disks); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err = d.Set("cd_roms", cdroms); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	var nics []map[string]interface{} = make([]map[string]interface{}, 0)
	for _, nic := range snapshot.VMNics {
		nicVlan, err := helper.GetVlanFromLocalId(ct.Api, *nic.Vlan.VlanLocalID)
		if err != nil {
			return diag.FromErr(err)
		}
		nics = append(nics, map[string]interface{}{
			"vlan_id": nicVlan.ID,
			"enabled": nic.Enabled,
			"mirror":  nic.Mirror,
			"model":   nic.Model,
			"idx":     nic.Index,
		})
	}
	if err = d.Set("nics", nics); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("consistent_type", snapshot.ConsistentType); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceVmSnapshotDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)
	dvsp := vm_snapshot.NewDeleteVMSnapshotParams()
	id := d.Id()
	dvsp.RequestBody = &models.VMSnapshotDeletionParams{
		Where: &models.VMSnapshotWhereInput{
			ID: &id,
		},
	}
	snapshots, err := ct.Api.VMSnapshot.DeleteVMSnapshot(dvsp)
	if err != nil {
		return diag.FromErr(err)
	}
	taskIds := make([]string, 0)
	for _, c := range snapshots.Payload {
		if c.TaskID != nil {
			taskIds = append(taskIds, *c.TaskID)
		}
	}
	_, err = ct.WaitTasksFinish(ctx, taskIds)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return diags
}
