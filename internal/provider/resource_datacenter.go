package provider

import (
	"context"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/smartxworks/cloudtower-go-sdk/client/datacenter"
	"github.com/smartxworks/cloudtower-go-sdk/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDatacenter() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "CloudTower datacenter resource.",

		CreateContext: resourceDatacenterCreate,
		ReadContext:   resourceDatacenterRead,
		UpdateContext: resourceDatacenterUpdate,
		DeleteContext: resourceDatacenterDelete,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "datacenter's id",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "datacenter's name",
			},
		},
	}
}

func resourceDatacenterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	cdp := datacenter.NewCreateDatacenterParams()
	name := d.Get("name").(string)
	cdp.RequestBody = []*models.DatacenterCreationParams{{
		Name:           &name,
		OrganizationID: &ct.OrgId,
	}}
	datacenters, err := ct.Api.Datacenter.CreateDatacenter(cdp)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(*datacenters.Payload[0].Data.ID)

	return resourceDatacenterRead(ctx, d, meta)
}

func resourceDatacenterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)

	id := d.Id()
	gdp := datacenter.NewGetDatacentersParams()
	gdp.RequestBody = &models.GetDatacentersRequestBody{
		Where: &models.DatacenterWhereInput{
			ID: &id,
		},
	}
	datacenters, err := ct.Api.Datacenter.GetDatacenters(gdp)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(datacenters.Payload) < 1 {
		d.SetId("")
		return diags
	}
	if err = d.Set("name", datacenters.Payload[0].Name); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceDatacenterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	udp := datacenter.NewUpdateDatacenterParams()
	name := d.Get("name").(string)
	id := d.Id()
	udp.RequestBody = &models.DatacenterUpdationParams{
		Where: &models.DatacenterWhereInput{
			ID: &id,
		},
		Data: &models.DatacenterUpdationParamsData{
			Name: &name,
		},
	}
	_, err := ct.Api.Datacenter.UpdateDatacenter(udp)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceDatacenterRead(ctx, d, meta)
}

func resourceDatacenterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)
	ddp := datacenter.NewDeleteDatacenterParams()
	id := d.Id()
	ddp.RequestBody = &models.DatacenterDeletionParams{
		Where: &models.DatacenterWhereInput{
			ID: &id,
		},
	}
	_, err := ct.Api.Datacenter.DeleteDatacenter(ddp)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return diags
}
