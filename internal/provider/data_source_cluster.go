package provider

import (
	"context"
	"github.com/Yuyz0112/cloudtower-go-sdk/client/cluster"
	"github.com/Yuyz0112/cloudtower-go-sdk/models"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCluster() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower cluster data source.",

		ReadContext: dataSourceClusterRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter clusters by name",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter clusters by name contain a certain string",
			},
			"clusters": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of clusters",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "cluster's id",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "cluster's name",
						},
					},
				},
			},
		},
	}
}

func dataSourceClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	gp := cluster.NewGetClustersParams()
	gp.RequestBody = &models.GetClustersRequestBody{
		Where: &models.ClusterWhereInput{},
	}
	if name := d.Get("name").(string); name != "" {
		gp.RequestBody.Where.Name = &name
	}
	if nameContains := d.Get("name_contains").(string); nameContains != "" {
		gp.RequestBody.Where.NameContains = &nameContains
	}
	clusters, err := ct.Api.Cluster.GetClusters(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	for _, d := range clusters.Payload {
		output = append(output, map[string]interface{}{
			"id":   d.ID,
			"name": d.Name,
		})
	}
	err = d.Set("clusters", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}
