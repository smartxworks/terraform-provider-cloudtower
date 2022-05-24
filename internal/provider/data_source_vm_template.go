package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_template"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVmTemplate() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vm template data source.",

		ReadContext: dataSourceVmSnapshotRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "vm template's name",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter vm template by its name contains characters",
			},
			"cluster_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "cluster's id of the template",
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
	} else if nameContains := d.Get("name_contains").(string); nameContains != "" {
		gp.RequestBody.Where.NameContains = &nameContains
	}
	if cluster_id := d.Get("cluster_id").(string); cluster_id != "" {
		gp.RequestBody.Where.Cluster.ID = &cluster_id
	}
	vm_templates, err := ct.Api.VMTemplate.GetVMTemplates(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	for _, d := range vm_templates.Payload {
		output = append(output, map[string]interface{}{
			"id":          d.ID,
			"name":        d.Name,
			"create_time": d.LocalCreatedAt,
		})
	}
	err = d.Set("vm_templates", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}
