package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_template"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVmTemplate() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vm template data source.",

		ReadContext: dataSourceVmTemplateRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_in"},
				Description:   "filter vm template by its name",
			},
			"name_in": {
				Type:          schema.TypeList,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"name"},
				Description:   "filter vm template by its name as an array",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter vm template by its name contains characters",
			},
			"cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"cluster_id_in"},
				Description:   "filter vm template by cluster's id of the template",
			},
			"cluster_id_in": {
				Type:          schema.TypeList,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"cluster_id"},
				Description:   "filter vm template by cluster's id of the template as an array",
			},
			"vm_templates": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of queried vm templates",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "template's id",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "template's name",
						},
						"create_time": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "template's create_time",
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
						"cpu_cores": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "template's cpu cores",
						},
						"cpu_sockets": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "template's cpu sockets",
						},
						"memory": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "template's in the unit of byte, must be a multiple of 512MB, long value, ignore the decimal point",
						},
						"firmware": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "template's firmware, forcenew as it isn't able to modify after create, must be one of 'BIOS', 'UEFI'",
						},
						"clock_offset": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "template's clock offset, must be one of 'LOCALTIME', 'UTC'",
						},
						"win_opt": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "template's win_opt",
						},
					},
				},
			},
		},
	}
}

func dataSourceVmTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	gp := vm_template.NewGetVMTemplatesParams()
	gp.RequestBody = &models.GetVMTemplatesRequestBody{
		Where:   &models.VMTemplateWhereInput{},
		OrderBy: models.VMTemplateOrderByInputLocalCreatedAtASC.Pointer(),
	}
	if name := d.Get("name").(string); name != "" {
		gp.RequestBody.Where.Name = &name
	} else {
		nameIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("name_in").([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		} else if len(nameIn) > 0 {
			gp.RequestBody.Where.NameIn = nameIn
		}
	}
	if nameContains := d.Get("name_contains").(string); nameContains != "" {
		gp.RequestBody.Where.NameContains = &nameContains
	}
	if cluster_id := d.Get("cluster_id").(string); cluster_id != "" {
		gp.RequestBody.Where.Cluster = &models.ClusterWhereInput{
			ID: &cluster_id,
		}
	} else {
		clusterIdIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("cluster_id_in").([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		} else if len(clusterIdIn) > 0 {
			gp.RequestBody.Where.Cluster = &models.ClusterWhereInput{
				IDIn: clusterIdIn,
			}
		}
	}
	vm_templates, err := ct.Api.VMTemplate.GetVMTemplates(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	for _, d := range vm_templates.Payload {
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
			"id":           d.ID,
			"name":         d.Name,
			"create_time":  d.LocalCreatedAt,
			"disks":        disks,
			"cd_roms":      cdroms,
			"nics":         nics,
			"cpu_cores":    d.CPU.Cores,
			"cpu_sockets":  d.CPU.Sockets,
			"memory":       d.Memory,
			"firmware":     d.Firmware,
			"clock_offset": d.ClockOffset,
			"win_opt":      d.WinOpt,
		})
	}
	if diags.HasError() {
		return diags
	}
	err = d.Set("vm_templates", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}
