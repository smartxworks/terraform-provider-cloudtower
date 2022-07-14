package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vds"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vlan"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVlan() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vlan data source.",

		ReadContext: dataSourceVlanRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_in"},
				Description:   "filter vlans by name",
			},
			"name_in": {
				Type:          schema.TypeList,
				Optional:      true,
				Description:   "filter vlans by name",
				ConflictsWith: []string{"name"},
				Elem:          &schema.Schema{Type: schema.TypeString},
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter vlans by name contain a certain string",
			},
			"type": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "filter vlans by type",
				ConflictsWith: []string{"type_in"},
				ValidateFunc:  validation.StringInSlice([]string{"ACCESS", "MANAGEMENT", "MIGRATION", "STORAGE", "VM"}, false),
			},
			"type_in": {
				Type:          schema.TypeList,
				Optional:      true,
				Description:   "filter vlans by type as array",
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"type"},
				// ValidateFunc:  local_validation.ListStringInSlice([]string{"ACCESS", "MANAGEMENT", "MIGRATION", "STORAGE", "VM"}, false),
			},
			"cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"cluster_id_in"},
				Description:   "filter vlans by cluster id",
			},
			"cluster_id_in": {
				Type:          schema.TypeList,
				Optional:      true,
				Description:   "filter vlans by cluster id as array",
				ConflictsWith: []string{"cluster_id"},
				Elem:          &schema.Schema{Type: schema.TypeString},
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
						"cluster_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "vlan's cluster id",
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
	vdsVlanMap := make(map[string][]*models.Vlan, 0)
	vdsIdList := make([]string, 0)
	for _, d := range vlans.Payload {
		vdsIdList = append(vdsIdList, *d.Vds.ID)
		if _, ok := vdsVlanMap[*d.Vds.ID]; !ok {
			vdsVlanMap[*d.Vds.ID] = make([]*models.Vlan, 0)
		}
		vdsVlanMap[*d.Vds.ID] = append(vdsVlanMap[*d.Vds.ID], d)
	}

	getVdsParams := vds.NewGetVdsesParams()
	getVdsParams.RequestBody = &models.GetVdsesRequestBody{
		Where: &models.VdsWhereInput{
			IDIn: vdsIdList,
		},
	}
	vdses, err := ct.Api.Vds.GetVdses(getVdsParams)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, vds := range vdses.Payload {
		for _, vlan := range vdsVlanMap[*vds.ID] {
			output = append(output, map[string]interface{}{
				"id":         vlan.ID,
				"name":       vlan.Name,
				"type":       vlan.Type,
				"cluster_id": vds.Cluster.ID,
			})
		}
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
	} else {
		rawTypeIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("type_in").([]interface{}))
		if err != nil {
			return nil, err
		} else if len(rawTypeIn) > 0 {
			typeIn := make([]models.NetworkType, len(rawTypeIn))
			for _, t := range rawTypeIn {
				switch t {
				case "ACCESS":
					typeIn = append(typeIn, models.NetworkTypeACCESS)
				case "MANAGEMENT":
					typeIn = append(typeIn, models.NetworkTypeMANAGEMENT)
				case "MIGRATION":
					typeIn = append(typeIn, models.NetworkTypeMIGRATION)
				case "STORAGE":
					typeIn = append(typeIn, models.NetworkTypeSTORAGE)
				case "VM":
					typeIn = append(typeIn, models.NetworkTypeVM)
				}
			}
			where.TypeIn = typeIn
		}
	}

	if clusterId := d.Get("cluster_id").(string); clusterId != "" {
		where.Vds = &models.VdsWhereInput{
			Cluster: &models.ClusterWhereInput{
				ID: &clusterId,
			},
		}
	} else {
		clusterIdIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("cluster_id_in").([]interface{}))
		if err != nil {
			return nil, err
		} else if len(clusterIdIn) > 0 {
			where.Vds = &models.VdsWhereInput{
				Cluster: &models.ClusterWhereInput{
					IDIn: clusterIdIn,
				},
			}
		}
	}
	return where, nil
}
