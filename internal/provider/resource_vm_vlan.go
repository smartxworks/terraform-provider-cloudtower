package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vlan"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

func resourceVmVlan() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower VM VLAN resource.",

		CreateContext: resourceVmVlanCreate,
		ReadContext:   resourceVmVlanRead,
		UpdateContext: resourceVmVlanUpdate,
		DeleteContext: resourceVmVlanDelete,
		CustomizeDiff: resourceVmVlanCustomizeDiff,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "VLAN's name",
			},
			"vds_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the VDS this VLAN belongs to",
			},
			"mode_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "VLAN_ACCESS",
				ValidateFunc: validation.StringInSlice([]string{"VLAN_ACCESS", "VLAN_TRUNK"}, false),
				Description:  "VLAN mode type. Valid values: VLAN_ACCESS, VLAN_TRUNK",
			},
			"network_ids": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "VLAN network IDs. For ACCESS mode, pass a single VLAN ID (e.g. [\"100\"]). For TRUNK mode, pass IDs or ranges (e.g. [\"100\", \"200-205\"])",
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: helper.ValidateNetworkId,
				},
			},
			"local_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "VLAN's local ID",
			},
		},
	}
}

func resourceVmVlanCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	raw := d.Get("network_ids").([]interface{})
	modeType := d.Get("mode_type").(string)

	if modeType == "VLAN_ACCESS" {
		if len(raw) == 0 {
			return nil
		}
		if len(raw) != 1 {
			return fmt.Errorf("network_ids must contain exactly 1 element when mode_type is VLAN_ACCESS, got %d", len(raw))
		}
		s := raw[0].(string)
		for _, r := range s {
			if r == '-' {
				return fmt.Errorf("network_ids must be a single VLAN ID (no ranges) when mode_type is VLAN_ACCESS, got %q", s)
			}
		}
	}

	return nil
}

func resourceVmVlanCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	cvp := vlan.NewCreateVMVlanParams()
	name := d.Get("name").(string)
	vdsID := d.Get("vds_id").(string)
	modeType := models.VlanModeType(d.Get("mode_type").(string))
	params := &models.VMVlanCreationParams{
		Name:     &name,
		VdsID:    &vdsID,
		ModeType: &modeType,
	}

	networkIDs, err := helper.SliceInterfacesToTypeSlice[string](d.Get("network_ids").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	if len(networkIDs) > 0 {
		params.NetworkIds = networkIDs
	}

	cvp.RequestBody = []*models.VMVlanCreationParams{params}
	vlans, err := ct.Api.Vlan.CreateVMVlan(cvp)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(*vlans.Payload[0].Data.ID)
	taskIds := make([]string, 0)
	for _, v := range vlans.Payload {
		if v.TaskID != nil {
			taskIds = append(taskIds, *v.TaskID)
		}
	}
	_, err = ct.WaitTasksFinish(ctx, taskIds)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceVmVlanRead(ctx, d, meta)
}

func resourceVmVlanRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)

	id := d.Id()
	gvp := vlan.NewGetVlansParams()
	gvp.RequestBody = &models.GetVlansRequestBody{
		Where: &models.VlanWhereInput{
			ID: &id,
		},
	}
	vlans, err := ct.Api.Vlan.GetVlans(gvp)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(vlans.Payload) < 1 {
		d.SetId("")
		return diags
	}
	v := vlans.Payload[0]
	if err = d.Set("name", v.Name); err != nil {
		return diag.FromErr(err)
	}
	if v.ModeType != nil {
		if err = d.Set("mode_type", string(*v.ModeType)); err != nil {
			return diag.FromErr(err)
		}
	}
	if err = d.Set("local_id", v.LocalID); err != nil {
		return diag.FromErr(err)
	}

	networkIDs := make([]interface{}, len(v.NetworkIds))
	for i, id := range v.NetworkIds {
		networkIDs[i] = id
	}
	if err = d.Set("network_ids", networkIDs); err != nil {
		return diag.FromErr(err)
	}
	if v.Vds != nil && v.Vds.ID != nil {
		if err = d.Set("vds_id", *v.Vds.ID); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func resourceVmVlanUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	uvp := vlan.NewUpdateVlanParams()
	id := d.Id()
	data := &models.VMVlanUpdationParamsData{}
	if d.HasChange("name") {
		name := d.Get("name").(string)
		data.Name = &name
	}
	modeTypeChanged := d.HasChange("mode_type")
	if modeTypeChanged {
		modeType := models.VlanModeType(d.Get("mode_type").(string))
		data.ModeType = &modeType
	}
	// mode_type changes (e.g. TRUNK -> ACCESS) require the server to
	// re-validate network_ids, so send them together with mode_type.
	if modeTypeChanged || d.HasChange("network_ids") {
		networkIDs, err := helper.SliceInterfacesToTypeSlice[string](d.Get("network_ids").([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		data.NetworkIds = networkIDs
	}
	uvp.RequestBody = &models.VMVlanUpdationParams{
		Where: &models.VlanWhereInput{
			ID: &id,
		},
		Data: data,
	}
	vlans, err := ct.Api.Vlan.UpdateVlan(uvp)
	if err != nil {
		return diag.FromErr(err)
	}
	taskIds := make([]string, 0)
	for _, v := range vlans.Payload {
		if v.TaskID != nil {
			taskIds = append(taskIds, *v.TaskID)
		}
	}
	_, err = ct.WaitTasksFinish(ctx, taskIds)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceVmVlanRead(ctx, d, meta)
}

func resourceVmVlanDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)
	dvp := vlan.NewDeleteVlanParams()
	id := d.Id()
	dvp.RequestBody = &models.VlanDeletionParams{
		Where: &models.VlanWhereInput{
			ID: &id,
		},
	}
	vlans, err := ct.Api.Vlan.DeleteVlan(dvp)
	if err != nil {
		return diag.FromErr(err)
	}
	taskIds := make([]string, 0)
	for _, v := range vlans.Payload {
		if v.TaskID != nil {
			taskIds = append(taskIds, *v.TaskID)
		}
	}
	_, err = ct.WaitTasksFinish(ctx, taskIds)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return diags
}
