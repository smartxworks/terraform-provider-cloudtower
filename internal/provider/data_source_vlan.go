package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/Sczlog/cloudtower-go-sdk/client/vlan"
	"github.com/Sczlog/cloudtower-go-sdk/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVlan() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vlan data source.",

		ReadContext: dataSourceVlanRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter vlans by name",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter vlans by name contain a certain string",
			},
			"type": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "filter vlans by type",
				ValidateFunc: validation.StringInSlice([]string{"ACCESS", "MANAGEMENT", "MIGRATION", "STORAGE", "VM"}, false),
			},
			"cluster_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter vlans by cluster id",
			},
			"vlans": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of vlans",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "vlan's id",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "vlan's name",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "vlan's type",
						},
					},
				},
			},
		},
	}
}

func dataSourceVlanRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	gp := vlan.NewGetVlansParams()
	gp.RequestBody = &models.GetVlansRequestBody{
		Where: &models.VlanWhereInput{},
	}
	where, err := expandVlanWhereInput(d)
	if err != nil {
		return diag.FromErr(err)
	}
	gp.RequestBody.Where = where
	vlans, err := ct.Api.Vlan.GetVlans(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	for _, d := range vlans.Payload {
		output = append(output, map[string]interface{}{
			"id":   d.ID,
			"name": d.Name,
			"type": d.Type,
		})
	}
	err = d.Set("vlans", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}

func expandVlanWhereInput(d *schema.ResourceData) (*models.VlanWhereInput, error) {
	where := &models.VlanWhereInput{}
	if name := d.Get("name").(string); name != "" {
		where.Name = &name
	}
	if nameContains := d.Get("name_contains").(string); nameContains != "" {
		where.NameContains = &nameContains
	}
	if t := d.Get("type").(string); t != "" {
		switch t {
		case "ACCESS":
			pt := models.NetworkTypeACCESS
			where.Type = &pt
		case "MANAGEMENT":
			pt := models.NetworkTypeMANAGEMENT
			where.Type = &pt
		case "MIGRATION":
			pt := models.NetworkTypeMIGRATION
			where.Type = &pt
		case "STORAGE":
			pt := models.NetworkTypeSTORAGE
			where.Type = &pt
		case "VM":
			pt := models.NetworkTypeVM
			where.Type = &pt
		}
	}
	if clusterId := d.Get("cluster_id").(string); clusterId != "" {
		where.Vds = &models.VdsWhereInput{
			Cluster: &models.ClusterWhereInput{
				ID: &clusterId,
			},
		}
	}
	return where, nil
}
