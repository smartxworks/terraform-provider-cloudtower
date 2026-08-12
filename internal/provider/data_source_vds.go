package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vds"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

func dataSourceVds() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vds data source.",

		ReadContext: dataSourceVdsRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_in"},
				Description:   "filter vdses by name.",
			},
			"name_in": {
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"name"},
				Description:   "filter vdses by name in list.",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter vdses by name contains.",
			},
			"cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"cluster_id_in"},
				Description:   "filter vdses by cluster id.",
			},
			"cluster_id_in": {
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"cluster_id"},
				Description:   "filter vdses by cluster id in list.",
			},
			"type": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"type_in"},
				Description:   "filter vdses by type. Must be one of: ACCESS, MANAGEMENT, MIGRATION, STORAGE, VM.",
				ValidateFunc:  validation.StringInSlice([]string{"ACCESS", "MANAGEMENT", "MIGRATION", "STORAGE", "VM"}, false),
			},
			"type_in": {
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"type"},
				Description:   "filter vdses by type in list.",
			},
			"vds": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of vds",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The id of the vds.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the vds.",
						},
						"local_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The local id of the vds.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the vds.",
						},
						"bond_mode": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The bond mode of the vds.",
						},
						"ovsbr_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The OVS bridge name of the vds.",
						},
						"internal": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the vds is internal.",
						},
						"work_mode": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The work mode of the vds.",
						},
						"vlans_num": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of vlans in the vds.",
						},
						"cluster": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The cluster of the vds.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The cluster id of the vds.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The cluster name of the vds.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceVdsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	where, err := expandVdsWhereInput(d)
	if err != nil {
		return diag.FromErr(err)
	}

	gp := vds.NewGetVdsesParams()
	gp.RequestBody = &models.GetVdsesRequestBody{
		Where: where,
	}

	vdses, err := ct.Api.Vds.GetVdses(gp)
	if err != nil {
		return diag.FromErr(err)
	}

	output := make([]map[string]interface{}, 0)
	for _, g := range vdses.Payload {
		output = append(output, flattenVds(g))
	}
	err = d.Set("vds", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}

func expandVdsWhereInput(d *schema.ResourceData) (*models.VdsWhereInput, error) {
	where := &models.VdsWhereInput{}
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
	if t := d.Get("type").(string); t != "" {
		nt := models.NetworkType(t)
		where.Type = &nt
	} else {
		rawTypeIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("type_in").([]interface{}))
		if err != nil {
			return nil, err
		} else if len(rawTypeIn) > 0 {
			typeIn := make([]models.NetworkType, 0, len(rawTypeIn))
			for _, t := range rawTypeIn {
				typeIn = append(typeIn, models.NetworkType(t))
			}
			where.TypeIn = typeIn
		}
	}
	return where, nil
}

func flattenVds(g *models.Vds) map[string]interface{} {
	res := map[string]interface{}{}

	if g.ID != nil {
		res["id"] = *g.ID
	}
	if g.Name != nil {
		res["name"] = *g.Name
	}
	if g.LocalID != nil {
		res["local_id"] = *g.LocalID
	}
	if g.Type != nil {
		res["type"] = string(*g.Type)
	}
	if g.BondMode != nil {
		res["bond_mode"] = *g.BondMode
	}
	if g.OvsbrName != nil {
		res["ovsbr_name"] = *g.OvsbrName
	}
	if g.Internal != nil {
		res["internal"] = *g.Internal
	}
	if g.WorkMode != nil {
		res["work_mode"] = *g.WorkMode
	}
	if g.VlansNum != nil {
		res["vlans_num"] = *g.VlansNum
	}
	if g.Cluster != nil {
		cluster := map[string]interface{}{}
		if g.Cluster.ID != nil {
			cluster["id"] = *g.Cluster.ID
		}
		if g.Cluster.Name != nil {
			cluster["name"] = *g.Cluster.Name
		}
		res["cluster"] = []map[string]interface{}{cluster}
	}

	return res
}
