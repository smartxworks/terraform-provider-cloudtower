package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/content_library_vm_template"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_template"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceContentLibraryVmTemplate() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower content library VM template resource",

		CreateContext: resourceContentLibraryVmTemplateCreate,
		ReadContext:   resourceContentLibraryVmTemplateRead,
		DeleteContext: resourceContentLibraryVmTemplateDelete,
		UpdateContext: resourceContentLibraryVmTemplateUpdate,

		Schema: map[string]*schema.Schema{
			"src_vm_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Id of source vm from created vm to be cloned from",
			},
			"cluster_id": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Cluster id to distribute vm template to",
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

func resourceContentLibraryVmTemplateCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	cloneFrom := d.Get("src_vm_id").(string)
	var templates []*models.WithTaskContentLibraryVMTemplate
	if cloneFrom != "" {
		cclvtfv := content_library_vm_template.NewCloneContentLibraryVMTemplateFromVMParams()
		name := d.Get("name").(string)
		description := d.Get("description").(string)
		cloudInitSupported := d.Get("cloud_init_supported").(bool)
		cluster_ids, diags := getClusterIds(d)
		if diags != nil {
			return diags
		}
		cclvtfv.RequestBody = []*models.ContentLibraryVMTemplateCreationParams{
			{
				VM: &models.VMWhereUniqueInput{
					ID: &cloneFrom,
				},
				Name:               &name,
				Description:        &description,
				CloudInitSupported: &cloudInitSupported,
				Clusters: &models.ClusterWhereInput{
					IDIn: *cluster_ids,
				},
			},
		}
		response, err := ct.Api.ContentLibraryVMTemplate.CloneContentLibraryVMTemplateFromVM(cclvtfv)
		if err != nil {
			return diag.FromErr(err)
		}
		templates = response.Payload
	} else {
		return diag.FromErr(fmt.Errorf("must set src_vm_id"))
	}
	d.SetId(*templates[0].Data.ID)
	_, err := ct.WaitTasksFinish([]string{*templates[0].TaskID})
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceContentLibraryVmTemplateRead(ctx, d, meta)
}

func resourceContentLibraryVmTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)

	id := d.Id()
	gclvtp := content_library_vm_template.NewGetContentLibraryVMTemplatesParams()
	gclvtp.RequestBody = &models.GetContentLibraryVMTemplatesRequestBody{
		Where: &models.ContentLibraryVMTemplateWhereInput{
			ID: &id,
		},
	}
	vmTemplates, err := ct.Api.ContentLibraryVMTemplate.GetContentLibraryVMTemplates(gclvtp)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}
	if len(vmTemplates.Payload) < 1 {
		d.SetId("")
		diags = append(diags, diag.Errorf("len(vmTemplates.Payload) < 1")...)
		return diags
	}
	template := vmTemplates.Payload[0].VMTemplates[0]
	if err = d.Set("name", template.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	var disks []map[string]interface{} = make([]map[string]interface{}, 0)
	var cdroms []map[string]interface{} = make([]map[string]interface{}, 0)
	gvtp := vm_template.NewGetVMTemplatesParams()
	gvtp.RequestBody = &models.GetVMTemplatesRequestBody{
		Where: &models.VMTemplateWhereInput{
			ID: template.ID,
		},
	}
	resp, err := ct.Api.VMTemplate.GetVMTemplates(gvtp)
	rawTemplate := resp.Payload[0]
	for _, disk := range rawTemplate.VMDisks {
		if *disk.Type == models.VMDiskTypeDISK {
			storagePolicy, err := helper.GetElfStoragePolicyByLocalId(ct.Api, *disk.StoragePolicyUUID)
			if err != nil {
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
	for _, nic := range rawTemplate.VMNics {
		nicVlan, err := helper.GetVlanFromLocalId(ct.Api, *nic.Vlan.VlanLocalID)
		if err != nil {
			diags = append(diags, diag.FromErr(err)...)
			continue
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

func resourceContentLibraryVmTemplateDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)
	dclvtp := content_library_vm_template.NewDeleteContentLibraryVMTemplateParams()
	id := d.Id()
	dclvtp.RequestBody = &models.ContentLibraryVMTemplateDeletionParams{
		Where: &models.ContentLibraryVMTemplateWhereInput{
			ID: &id,
		},
	}
	templates, err := ct.Api.ContentLibraryVMTemplate.DeleteContentLibraryVMTemplate(dclvtp)
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

func resourceContentLibraryVmTemplateUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	uvtp := content_library_vm_template.NewUpdateContentLibraryVMTemplateParams()
	id := d.Id()
	if d.HasChange("name") || d.HasChange("description") || d.HasChange("cloud_init_supported") {
		name := d.Get("name").(string)
		description := d.Get("description").(string)
		cloudInitSupported := d.Get("cloud_init_supported").(bool)
		uvtp.RequestBody = &models.ContentLibraryVMTemplateUpdationParams{
			Where: &models.ContentLibraryVMTemplateWhereInput{
				ID: &id,
			},
			Data: &models.ContentLibraryVMTemplateUpdationParamsData{
				Name:               &name,
				Description:        &description,
				CloudInitSupported: &cloudInitSupported,
			},
		}
		_, err := ct.Api.ContentLibraryVMTemplate.UpdateContentLibraryVMTemplate(uvtp)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange("cluster_id") {
		cluster_ids, diags := getClusterIds(d)
		if diags != nil {
			return diags
		}
		dtclvtp := content_library_vm_template.NewDistributeContentLibraryVmtemplateClustersParams()
		dtclvtp.RequestBody = &models.ContentLibraryVMTemplateUpdationClusterParams{
			Where: &models.ContentLibraryVMTemplateWhereInput{
				ID: &id,
			},
			Data: &models.ContentLibraryVMTemplateUpdationClusterParamsData{
				Clusters: &models.ClusterWhereInput{
					IDIn: *cluster_ids,
				},
			},
		}
	}
	return resourceContentLibraryVmTemplateRead(ctx, d, meta)
}

func getClusterIds(d *schema.ResourceData) (*[]string, diag.Diagnostics) {
	raw_cluster_id, ok := d.GetOk("cluster_id")
	cluster_ids := make([]string, 0)
	if ok {
		bytes, err := json.Marshal(raw_cluster_id)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		err = json.Unmarshal(bytes, &cluster_ids)
		if err != nil {
			return nil, diag.FromErr(err)
		}
	}
	return &cluster_ids, nil
}
