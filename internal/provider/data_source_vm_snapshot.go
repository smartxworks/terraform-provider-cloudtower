package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_snapshot"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVmSnapshot() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vm snapshot data source.",

		ReadContext: dataSourceVmSnapshotRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_in"},
				Description:   "vm snapshot's name",
			},
			"name_in": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"name"},
				Description:   "vm snapshot's name as an array",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter vm snapshot by its name contains characters",
			},
			"vm_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"vm_id_in"},
				Description:   "vm's id of the snapshot",
			},
			"vm_id_in": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"vm_id"},
				Description:   "vm's id of the snapshot as an array",
			},
			"vm_snapshots": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of queried vm snapshots",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "snapshots's id",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "snapshot's name",
						},
						"create_time": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "snapshot's create_time",
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
							Description: "snapshot's disks",
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
							Description: "snapshot's cd_rom",
						},
						"nics": {
							Type: schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"vlan_id": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "specific the vlan's id the snapshot nic is using",
									},
									"enabled": {
										Type:        schema.TypeBool,
										Description: "whether the snapshot nic is enabled",
										Computed:    true,
									},
									"mirror": {
										Type:        schema.TypeBool,
										Description: "whether the snapshot nic use mirror mode",
										Computed:    true,
									},
									"model": {
										Type:        schema.TypeString,
										Description: "snapshot nic's model",
										Computed:    true,
									},
									"idx": {
										Type:        schema.TypeInt,
										Description: "snapshot nic's index",
										Computed:    true,
									},
								},
							},
							Computed:    true,
							Description: "snapshot's nics",
						},
					},
				},
			},
		},
	}
}

func dataSourceVmSnapshotRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	gp := vm_snapshot.NewGetVMSnapshotsParams()
	gp.RequestBody = &models.GetVMSnapshotsRequestBody{
		Where:   &models.VMSnapshotWhereInput{},
		OrderBy: models.VMSnapshotOrderByInputLocalCreatedAtASC.Pointer(),
	}
	if name := d.Get("name").(string); name != "" {
		gp.RequestBody.Where.Name = &name
	} else {
		nameIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("name_in").([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		} else {
			gp.RequestBody.Where.NameIn = nameIn
		}
	}
	if nameContains := d.Get("name_contains").(string); nameContains != "" {
		gp.RequestBody.Where.NameContains = &nameContains
	}
	if vmId := d.Get("vm_id").(string); vmId != "" {
		gp.RequestBody.Where.VM.ID = &vmId
	} else {
		vmIdIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("vm_id_in").([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		} else {
			gp.RequestBody.Where.VM.IDIn = vmIdIn
		}
	}
	snapshots, err := ct.Api.VMSnapshot.GetVMSnapshots(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	for _, d := range snapshots.Payload {
		var disks []map[string]interface{} = make([]map[string]interface{}, 0)
		var cdroms []map[string]interface{} = make([]map[string]interface{}, 0)
		for _, disk := range d.VMDisks {
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
		var nics []map[string]interface{} = make([]map[string]interface{}, 0)
		for _, nic := range d.VMNics {
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
		output = append(output, map[string]interface{}{
			"id":          d.ID,
			"name":        d.Name,
			"create_time": d.LocalCreatedAt,
			"disks":       disks,
			"cd_roms":     cdroms,
			"nics":        nics,
		})
	}
	for _, d := range snapshots.Payload {
		output = append(output, map[string]interface{}{
			"id":          d.ID,
			"name":        d.Name,
			"create_time": d.LocalCreatedAt,
		})
	}
	err = d.Set("vm_snapshots", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}
