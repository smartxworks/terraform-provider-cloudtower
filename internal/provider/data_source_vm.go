package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/Sczlog/cloudtower-go-sdk/client/vm"
	"github.com/Sczlog/cloudtower-go-sdk/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVm() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vm data source.",

		ReadContext: dataSourceVmRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter VMs by name",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter VMs by name contain a certain string",
			},
			"status": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "filter VMs by status",
				ValidateFunc: validation.StringInSlice([]string{"RUNNING", "STOPPED", "SUSPENDED"}, false),
			},
			"cluster_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter VMs by cluster id",
			},
			"host_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter VMs by host id",
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
	}
	if clusterId := d.Get("cluster_id").(string); clusterId != "" {
		where.Cluster = &models.ClusterWhereInput{
			ID: &clusterId,
		}
	}
	if hostId := d.Get("host_id").(string); hostId != "" {
		where.Host = &models.HostWhereInput{
			ID: &hostId,
		}
	}
	return where, nil
}
