package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/svt_image"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSvtImage() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vmtools iso data source.",

		ReadContext: dataSourceSvtImageRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_in"},
				Description:   "filter svt ISOs by name",
			},
			"name_in": {
				Type:          schema.TypeList,
				Optional:      true,
				Description:   "filter svt ISOs by name in",
				ConflictsWith: []string{"name"},
				Elem:          &schema.Schema{Type: schema.TypeString},
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter svt ISOs by name contain a certain string",
			},
			"cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"cluster_id_in"},
				Description:   "filter svt ISOs by cluster id",
			},
			"cluster_id_in": {
				Type:          schema.TypeList,
				Optional:      true,
				Description:   "filter svt ISOs by cluster id in",
				ConflictsWith: []string{"cluster_id"},
				Elem:          &schema.Schema{Type: schema.TypeString},
			},
			"version": {
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"version_in"},
				Description:   "filter svt ISOs by version",
			},
			"version_in": {
				Type:          schema.TypeList,
				Optional:      true,
				Description:   "filter svt ISOs by version in",
				ConflictsWith: []string{"version"},
				Elem:          &schema.Schema{Type: schema.TypeInt},
			},
			"version_gte": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "filter svt ISOs by version greater than or equal to",
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
						"version": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "ISO's version",
						},
					},
				},
			},
		},
	}
}

func dataSourceSvtImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	gp := svt_image.NewGetSvtImagesParams()
	gp.RequestBody = &models.GetSvtImagesRequestBody{
		Where: &models.SvtImageWhereInput{},
	}
	where, err := expandSvtImageWhereInput(d)
	if err != nil {
		return diag.FromErr(err)
	}
	gp.RequestBody.Where = where
	isos, err := ct.Api.SvtImage.GetSvtImages(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	for _, d := range isos.Payload {
		output = append(output, map[string]interface{}{
			"id":      d.ID,
			"name":    d.Name,
			"version": d.Version,
		})
	}
	err = d.Set("isos", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}

func expandSvtImageWhereInput(d *schema.ResourceData) (*models.SvtImageWhereInput, error) {
	where := &models.SvtImageWhereInput{}
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
	if version, ok := d.GetOkExists("version"); ok {
		version := int32(version.(int))
		where.Version = &version
	} else {
		rawVersionIn, err := helper.SliceInterfacesToTypeSlice[int](d.Get("version_in").([]interface{}))
		if err != nil {
			return nil, err
		} else if len(rawVersionIn) > 0 {
			versionIn := make([]int32, len(rawVersionIn))
			for i, v := range rawVersionIn {
				versionIn[i] = int32(v)
			}
			where.VersionIn = versionIn
		}
	}
	if versionGte, ok := d.GetOkExists("version_gte"); ok {
		versionGte := int32(versionGte.(int))
		where.VersionGte = &versionGte
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
