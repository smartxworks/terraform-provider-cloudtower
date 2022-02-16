package provider

import (
	"context"
	"encoding/json"
	"github.com/Yuyz0112/cloudtower-go-sdk/client/datacenter"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDatacenter() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "CloudTower datacenter data source.",

		ReadContext: dataSourceDatacenterRead,

		Schema: map[string]*schema.Schema{
			"datacenters": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"failure_data_space": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"cluster_num": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"total_cpu_hz": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"total_data_capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"total_memory_bytes": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"used_cpu_hz": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"used_data_space": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"used_memory_bytes": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"organization": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
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

	gdp := datacenter.NewGetDatacentersParams()
	datacenters, err := ct.Api.Datacenter.GetDatacenters(gdp)
	if err != nil {
		return diag.FromErr(err)
	}
	str, err := json.Marshal(datacenters.Payload)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	err = json.Unmarshal(str, &output)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("datacenters", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}
