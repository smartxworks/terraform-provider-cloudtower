package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/smartxworks/cloudtower-go-sdk/client/vm_snapshot"
	"github.com/smartxworks/cloudtower-go-sdk/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVmSnapshot() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vm snapshot data source.",

		ReadContext: dataSourceVmSnapshotRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "vm snapshot's id",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "vm snapshot's name",
			},
			"vm_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "vm's id of the snapshot",
			},
			"vm_snapshots": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of queried vm snapshots",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "snapshots's id",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "snapshot's name",
						},
						"create_time": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "snapshot's create_time",
						},
					},
				},
			},
		},
	}
}

func dataSourceVmSnapshotRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	gp := vm_snapshot.NewGetVMSnapshotsParams()
	gp.RequestBody = &models.GetVMSnapshotsRequestBody{
		Where:   &models.VMSnapshotWhereInput{},
		OrderBy: models.VMSnapshotOrderByInputLocalCreatedAtASC.Pointer(),
	}
	if name := d.Get("name").(string); name != "" {
		gp.RequestBody.Where.Name = &name
	}
	// if nameContains := d.Get("name_contains").(string); nameContains != "" {
	// 	gp.RequestBody.Where.NameContains = &nameContains
	// }
	clusters, err := ct.Api.VMSnapshot.GetVMSnapshots(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	for _, d := range clusters.Payload {
		output = append(output, map[string]interface{}{
			"id":          d.ID,
			"name":        d.Name,
			"create_time": d.LocalCreatedAt,
		})
	}
	err = d.Set("vm_snapshots", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}
