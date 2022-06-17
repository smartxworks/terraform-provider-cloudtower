package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_template"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVmTemplate() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower VM template resource",

		CreateContext: resourceVmTemplateCreate,
		ReadContext:   resourceVmTemplateRead,
		DeleteContext: resourceVmTemplateDelete,
		UpdateContext: resourceVmTemplateUpdate,

		Schema: map[string]*schema.Schema{
			// "create_effect": {
			// 	Type:     schema.TypeList,
			// 	MaxItems: 1,
			// 	Required: true,
			// 	Elem: &schema.Resource{
			// 		Schema: map[string]*schema.Schema{
			// 			"clone_from_vm": {
			// 				Type:        schema.TypeString,
			// 				Optional:    true,
			// 				ForceNew:    true,
			// 				Description: "Id of source vm from created vm to be cloned from",
			// 			},
			// 			// "convert_from_vm": {
			// 			// 	Type:        schema.TypeString,
			// 			// 	Optional:    true,
			// 			// 	ForceNew:    true,
			// 			// 	Description: "Id of source vm from created vm to be converted from",
			// 			// },
			// 		},
			// 	},
			// },
			"src_vm_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Id of source vm from created vm to be cloned from",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of VM template",
			},
			"cloud_init_supported": {
				Type:        schema.TypeBool,
				Required:    true,
				ForceNew:    true,
				Description: "If the cloud-init is installed or not",
			},
			"description": {
				Type:        schema.TypeString,
				Description: "VM template's description",
				Optional:    true,
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "VM template's id",
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

func resourceVmTemplateCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	// convertedFrom := d.Get("create_effect.0.convert_from_vm").(string)
	cloneFrom := d.Get("src_vm_id").(string)
	var templates []*models.WithTaskVMTemplate
	// if convertedFrom != "" && cloneFrom != "" {
	// 	return diag.FromErr(fmt.Errorf("convert_from_vm and clone_from_vm can not be set at the same time"))
	// } else if convertedFrom != "" {
	// 	// we maybe need to remove this later
	// 	gvp := vm.NewGetVmsParams()
	// 	gvp.RequestBody = &models.GetVmsRequestBody{
	// 		Where: &models.VMWhereInput{
	// 			ID: &convertedFrom,
	// 		},
	// 	}
	// 	vms, err := ct.Api.VM.GetVms(gvp)
	// 	if err != nil {
	// 		return diag.FromErr(err)
	// 	}
	// 	cvtfv := vm_template.NewConvertVMTemplateFromVMParams()
	// 	name := d.Get("name").(string)
	// 	description := d.Get("description").(string)
	// 	cloudInitSupported := d.Get("cloud_init_supported").(bool)
	// 	cvtfv.RequestBody = []*models.VMTemplateCreationParams{
	// 		{
	// 			VMID:               &convertedFrom,
	// 			Name:               &name,
	// 			Description:        &description,
	// 			CloudInitSupported: &cloudInitSupported,
	// 			ClusterID:          vms.Payload[0].Cluster.ID,
	// 		},
	// 	}
	// 	response, err := ct.Api.VMTemplate.ConvertVMTemplateFromVM(cvtfv)
	// 	if err != nil {
	// 		return diag.FromErr(err)
	// 	}
	// 	templates = response.Payload
	// } else
	if cloneFrom != "" {
		cvtfv := vm_template.NewCloneVMTemplateFromVMParams()
		name := d.Get("name").(string)
		description := d.Get("description").(string)
		cloudInitSupported := d.Get("cloud_init_supported").(bool)
		cvtfv.RequestBody = []*models.VMTemplateCreationParams{
			{
				VMID:               &cloneFrom,
				Name:               &name,
				Description:        &description,
				CloudInitSupported: &cloudInitSupported,
			},
		}
		response, err := ct.Api.VMTemplate.CloneVMTemplateFromVM(cvtfv)
		if err != nil {
			return diag.FromErr(err)
		}
		templates = response.Payload
	} else {
		return diag.FromErr((fmt.Errorf("must set src_vm_id")))
	}
	d.SetId(*templates[0].Data.ID)
	_, err := ct.WaitTasksFinish([]string{*templates[0].TaskID})
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceVmTemplateRead(ctx, d, meta)
}

func resourceVmTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)

	id := d.Id()
	gvtp := vm_template.NewGetVMTemplatesParams()
	gvtp.RequestBody = &models.GetVMTemplatesRequestBody{
		Where: &models.VMTemplateWhereInput{
			ID: &id,
		},
	}
	vmTemplates, err := ct.Api.VMTemplate.GetVMTemplates(gvtp)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(vmTemplates.Payload) < 1 {
		d.SetId("")
		return diags
	}
	template := vmTemplates.Payload[0]
	if err = d.Set("name", template.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	var disks []map[string]interface{} = make([]map[string]interface{}, 0)
	var cdroms []map[string]interface{} = make([]map[string]interface{}, 0)
	for _, disk := range template.VMDisks {
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
	for _, nic := range template.VMNics {
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
	if diags.HasError() {
		return diags
	}
	return diags
}

func resourceVmTemplateDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)
	dvtp := vm_template.NewDeleteVMTemplateParams()
	id := d.Id()
	dvtp.RequestBody = &models.VMTemplateDeletionParams{
		Where: &models.VMTemplateWhereInput{
			ID: &id,
		},
	}
	templates, err := ct.Api.VMTemplate.DeleteVMTemplate(dvtp)
	if err != nil {
		return diag.FromErr(err)
	}
	taskIds := make([]string, 0)
	for _, c := range templates.Payload {
		if c.TaskID != nil {
			taskIds = append(taskIds, *c.TaskID)
		}
	}
	_, err = ct.WaitTasksFinish(taskIds)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return diags
}

func resourceVmTemplateUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	uvtp := vm_template.NewUpdateVMTemplateParams()
	name := d.Get("name").(string)
	description := d.Get("description").(string)
	cloudInitSupported := d.Get("cloud_init_supported").(bool)
	id := d.Id()
	uvtp.RequestBody = &models.VMTemplateUpdationParams{
		Where: &models.VMTemplateWhereInput{
			ID: &id,
		},
		Data: &models.VMTemplateUpdationParamsData{
			Name:               &name,
			Description:        &description,
			CloudInitSupported: &cloudInitSupported,
		},
	}
	_, err := ct.Api.VMTemplate.UpdateVMTemplate(uvtp)
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceVmTemplateRead(ctx, d, meta)
}
