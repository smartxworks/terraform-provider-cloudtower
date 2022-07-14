package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	local_validation "github.com/hashicorp/terraform-provider-cloudtower/internal/validation"
)

func dataSourceVm() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vm data source.",

		ReadContext: dataSourceVmRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_in"},
				Description:   "filter VMs by name",
			},
			"name_in": {
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"name"},
				Description:   "filter VMs by name in an array",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter VMs by name contain a certain string",
			},
			"status": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "filter VMs by status",
				ConflictsWith: []string{"status_in"},
				ValidateFunc:  validation.StringInSlice([]string{"RUNNING", "STOPPED", "SUSPENDED"}, false),
			},
			"status_in": {
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				Description:   "filter VMs by status in an array",
				ConflictsWith: []string{"status"},
				ValidateFunc:  local_validation.ListStringInSlice([]string{"RUNNING", "STOPPED", "SUSPENDED"}, false),
			},
			"cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"cluster_id_in"},
				Description:   "filter VMs by cluster id",
			},
			"cluster_id_in": {
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"cluster_id"},
				Description:   "filter VMs by cluster id in an array",
			},
			"host_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"host_id_in"},
				Description:   "filter VMs by host id",
			},
			"host_id_in": {
				Type:          schema.TypeList,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"host_id"},
				Description:   "filter VMs by host id in an array",
			},
			"vms": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of VMs",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VM's id",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VM's name",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VM's status",
						},
					},
				},
			},
		},
	}
}

func dataSourceVmRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	gp := vm.NewGetVmsParams()
	gp.RequestBody = &models.GetVmsRequestBody{
		Where: &models.VMWhereInput{},
	}
	where, err := expandVmWhereInput(d)
	if err != nil {
		return diag.FromErr(err)
	}
	gp.RequestBody.Where = where
	vms, err := ct.Api.VM.GetVms(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	for _, d := range vms.Payload {
		output = append(output, map[string]interface{}{
			"id":     d.ID,
			"name":   d.Name,
			"status": d.Status,
		})
	}
	err = d.Set("vms", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}

func expandVmWhereInput(d *schema.ResourceData) (*models.VMWhereInput, error) {
	where := &models.VMWhereInput{}
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
	if t := d.Get("status").(string); t != "" {
		switch t {
		case "RUNNING":
			pt := models.VMStatusRUNNING
			where.Status = &pt
		case "STOPPED":
			pt := models.VMStatusSTOPPED
			where.Status = &pt
		case "SUSPENDED":
			pt := models.VMStatusSUSPENDED
			where.Status = &pt
		}
	} else {
		rawTypeIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("status_in").([]interface{}))
		if err != nil {
			return nil, err
		} else if len(rawTypeIn) > 0 {
			typeIn := make([]models.VMStatus, len(rawTypeIn))
			for i, t := range rawTypeIn {
				switch t {
				case "RUNNING":
					typeIn[i] = models.VMStatusRUNNING
				case "STOPPED":
					typeIn[i] = models.VMStatusSTOPPED
				case "SUSPENDED":
					typeIn[i] = models.VMStatusSUSPENDED
				}
			}
			where.StatusIn = typeIn
		}
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
	if hostId := d.Get("host_id").(string); hostId != "" {
		where.Host = &models.HostWhereInput{
			ID: &hostId,
		}
	} else {
		hostIdIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("host_id_in").([]interface{}))
		if err != nil {
			return nil, err
		} else if len(hostIdIn) > 0 {
			where.Host = &models.HostWhereInput{
				IDIn: hostIdIn,
			}
		}
	}
	return where, nil
}
