package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/datacenter"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDatacenter() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower datacenter data source.",

		ReadContext: dataSourceDatacenterRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_in"},
				Description:   "filter datacenters by name",
			},
			"name_in": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"name"},
				Description:   "filter datacenters by name as an array",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter datacenters by name contain a certain string",
			},
			"datacenters": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of datacenters",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "datacenter's id",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "datacenter's name",
						},
					},
				},
			},
		},
	}
}

func dataSourceDatacenterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	gp := datacenter.NewGetDatacentersParams()
	gp.RequestBody = &models.GetDatacentersRequestBody{
		Where: &models.DatacenterWhereInput{},
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

	datacenters, err := ct.Api.Datacenter.GetDatacenters(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	for _, d := range datacenters.Payload {
		output = append(output, map[string]interface{}{
			"id":   d.ID,
			"name": d.Name,
		})
	}
	err = d.Set("datacenters", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}
