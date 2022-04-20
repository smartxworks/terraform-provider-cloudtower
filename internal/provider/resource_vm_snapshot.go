package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/smartxworks/cloudtower-go-sdk/client/vm_snapshot"
	"github.com/smartxworks/cloudtower-go-sdk/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVmSnapshot() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower vm snapshot resource",

		CreateContext: resourceVmSnapshotCreate,
		ReadContext:   resourceVmSnapshotRead,
		DeleteContext: resourceVmSnapshotDelete,

		Schema: map[string]*schema.Schema{
			"vm_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The id of vm to the snapshot belongs to",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of snapshot",
			},
			"consistent_type": {
				Type:        schema.TypeString,
				Default:     string(models.ConsistentTypeCRASHCONSISTENT),
				ForceNew:    true,
				Optional:    true,
				Description: "The consistent type of snapshot",
				ValidateDiagFunc: func(v interface{}, _ cty.Path) diag.Diagnostics {
					var diags diag.Diagnostics
					val, ok := v.(string)
					if !ok {
						return append(diags, diag.Diagnostic{
							Severity: diag.Error,
							Summary:  "Wrong type",
							Detail:   "Consistent type should be a string",
						})
					} else if val != string(models.ConsistentTypeCRASHCONSISTENT) && val != string(models.ConsistentTypeFILESYSTEMCONSISTENT) {
						return append(diags, diag.Diagnostic{
							Severity: diag.Error,
							Summary:  "Invalid consistent type",
							Detail: fmt.Sprintf("Consistent type should be one of %v, but get %s",
								[]string{string(models.ConsistentTypeCRASHCONSISTENT), string(models.ConsistentTypeFILESYSTEMCONSISTENT)},
								val,
							),
						})
					}
					return diags
				},
			},
		},
	}
}

func resourceVmSnapshotCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	cvsp := vm_snapshot.NewCreateVMSnapshotParams()
	consistent_type := models.ConsistentType(d.Get("consistent_type").(string))
	name := d.Get("name").(string)
	vm_id := d.Get("vm_id").(string)
	cvsp.RequestBody = &models.VMSnapshotCreationParams{
		Data: []*models.VMSnapshotCreationParamsDataItems0{
			{
				ConsistentType: &consistent_type,
				Name:           &name,
				VMID:           &vm_id,
			},
		},
	}
	snapshots, err := ct.Api.VMSnapshot.CreateVMSnapshot(cvsp)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(*snapshots.Payload[0].Data.ID)
	_, err = ct.WaitTasksFinish([]string{*snapshots.Payload[0].TaskID})
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceVmSnapshotRead(ctx, d, meta)
}

func resourceVmSnapshotRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)

	id := d.Id()
	gvsp := vm_snapshot.NewGetVMSnapshotsParams()
	gvsp.RequestBody = &models.GetVMSnapshotsRequestBody{
		Where: &models.VMSnapshotWhereInput{
			ID: &id,
		},
	}
	vmSnapshots, err := ct.Api.VMSnapshot.GetVMSnapshots(gvsp)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(vmSnapshots.Payload) < 1 {
		d.SetId("")
		return diags
	}
	snapshot := vmSnapshots.Payload[0]
	if err := d.Set("consistent_type", snapshot.ConsistentType); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceVmSnapshotDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)
	dvsp := vm_snapshot.NewDeleteVMSnapshotParams()
	id := d.Id()
	dvsp.RequestBody = &models.VMSnapshotDeletionParams{
		Where: &models.VMSnapshotWhereInput{
			ID: &id,
		},
	}
	snapshots, err := ct.Api.VMSnapshot.DeleteVMSnapshot(dvsp)
	if err != nil {
		return diag.FromErr(err)
	}
	taskIds := make([]string, 0)
	for _, c := range snapshots.Payload {
		if c.TaskID != nil {
			taskIds = append(taskIds, *c.TaskID)
		}
	}
	_, err = ct.WaitTasksFinish(taskIds)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return diags
}
