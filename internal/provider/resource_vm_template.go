package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
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
		return diag.FromErr(err)
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
