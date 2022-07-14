package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/host"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHost() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower host data source.",

		ReadContext: dataSourceHostRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_in"},
				Description:   "filter hosts by name",
			},
			"name_in": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"name"},
				Description:   "filter data source by name as an array",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter hosts by name contain a certain string",
			},
			"management_ip": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"management_ip_in"},
				Description:   "filter hosts by management IP",
			},
			"management_ip_in": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"management_ip"},
				Description:   "filter datacenters by management ip as an array",
			},
			"management_ip_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter hosts by management IP contain a certain string",
			},
			"data_ip": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"data_ip_in"},
				Description:   "filter hosts by data IP",
			},
			"data_ip_in": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"data_ip"},
				Description:   "filter datacenters by data ip as an array",
			},
			"data_ip_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter hosts by data IP contain a certain string",
			},
			"cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"cluster_id_in"},
				Description:   "filter hosts by cluster id",
			},
			"cluster_id_in": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"cluster_id"},
				Description:   "filter datacenters by cluster id as an array",
			},
			"hosts": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of hosts",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "host's id",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "host's name",
						},
						"management_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "host's management IP",
						},
						"data_ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "host's data IP",
						},
					},
				},
			},
		},
	}
}

func dataSourceHostRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	gp := host.NewGetHostsParams()
	gp.RequestBody = &models.GetHostsRequestBody{
		Where: &models.HostWhereInput{},
	}
	where, err := expandHostWhereInput(d)
	if err != nil {
		return diag.FromErr(err)
	}
	gp.RequestBody.Where = where
	hosts, err := ct.Api.Host.GetHosts(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	for _, d := range hosts.Payload {
		output = append(output, map[string]interface{}{
			"id":            d.ID,
			"name":          d.Name,
			"management_ip": d.ManagementIP,
			"data_ip":       d.DataIP,
		})
	}
	err = d.Set("hosts", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}

func expandHostWhereInput(d *schema.ResourceData) (*models.HostWhereInput, error) {
	where := &models.HostWhereInput{}
	if name := d.Get("name").(string); name != "" {
		where.Name = &name
	} else {
		nameIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("name_in").([]interface{}))
		if err != nil {
			return nil, err
		} else if len(nameIn) > 0 {
			where.NameIn = nameIn
		}

	}
	if nameContains := d.Get("name_contains").(string); nameContains != "" {
		where.NameContains = &nameContains
	}
	if mgtIp := d.Get("management_ip").(string); mgtIp != "" {
		where.ManagementIP = &mgtIp
	} else {
		mgtIpIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("management_ip_in").([]interface{}))
		if err != nil {
			return nil, err
		} else if len(mgtIpIn) > 0 {
			where.ManagementIPIn = mgtIpIn
		}
	}

	if mgtIpContains := d.Get("management_ip_contains").(string); mgtIpContains != "" {
		where.ManagementIPContains = &mgtIpContains
	}
	if dataIp := d.Get("data_ip").(string); dataIp != "" {
		where.DataIP = &dataIp
	} else {
		dataIpIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("data_ip_in").([]interface{}))
		if err != nil {
			return nil, err
		} else if len(dataIpIn) > 0 {
			where.DataIPIn = dataIpIn
		}
	}

	if dataIpContains := d.Get("data_ip_contains").(string); dataIpContains != "" {
		where.DataIPContains = &dataIpContains
	}
	if clusterId := d.Get("cluster_id").(string); clusterId != "" {
		where.Cluster = &models.ClusterWhereInput{
			ID: &clusterId,
		}
	} else {
		clusterIdIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("cluster_id_in").([]interface{}))
		if err != nil {
			return nil, err
		} else if len(clusterIdIn) > 0 {
			where.Cluster = &models.ClusterWhereInput{
				IDIn: clusterIdIn,
			}
		}
	}
	return where, nil
}
