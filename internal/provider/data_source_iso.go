package provider

import (
	"context"
	"github.com/Yuyz0112/cloudtower-go-sdk/client/elf_image"
	"github.com/Yuyz0112/cloudtower-go-sdk/models"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceIso() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower iso data source.",

		ReadContext: dataSourceIsoRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter ISOs by name",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter ISOs by name contain a certain string",
			},
			"cluster_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter ISOs by cluster id",
			},
			"isos": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of ISOs",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ISO's id",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ISO's name",
						},
					},
				},
			},
		},
	}
}

func dataSourceIsoRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	gp := elf_image.NewGetElfImagesParams()
	gp.RequestBody = &models.GetElfImagesRequestBody{
		Where: &models.ElfImageWhereInput{},
	}
	where, err := expandIsoWhereInput(d)
	if err != nil {
		return diag.FromErr(err)
	}
	gp.RequestBody.Where = where
	isos, err := ct.Api.ElfImage.GetElfImages(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	for _, d := range isos.Payload {
		output = append(output, map[string]interface{}{
			"id":   d.ID,
			"name": d.Name,
		})
	}
	err = d.Set("isos", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}

func expandIsoWhereInput(d *schema.ResourceData) (*models.ElfImageWhereInput, error) {
	where := &models.ElfImageWhereInput{}
	if name := d.Get("name").(string); name != "" {
		where.Name = &name
	}
	if nameContains := d.Get("name_contains").(string); nameContains != "" {
		where.NameContains = &nameContains
	}
	if clusterId := d.Get("cluster_id").(string); clusterId != "" {
		where.Cluster = &models.ClusterWhereInput{
			ID: &clusterId,
		}
	}
	return where, nil
}
