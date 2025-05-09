package provider

import (
	"context"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/cluster"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCluster() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower cluster resource.",

		CreateContext: resourceClusterCreate,
		ReadContext:   resourceClusterRead,
		UpdateContext: resourceClusterUpdate,
		DeleteContext: resourceClusterDelete,

		Schema: map[string]*schema.Schema{
			"ip": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "cluster's IP",
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "cluster's username",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "cluster's password",
			},
			"datacenter_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "the id of the datacenter this cluster belongs to",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "cluster's id",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "cluster's name",
			},
		},
	}
}

func resourceClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	ccp := cluster.NewConnectClusterParams()
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	ip := d.Get("ip").(string)
	datacenterId := d.Get("datacenter_id").(string)
	ccp.RequestBody = []*models.ClusterCreationParams{{
		IP:           &ip,
		Username:     &username,
		Password:     &password,
		DatacenterID: &datacenterId,
	}}
	clusters, err := ct.Api.Cluster.ConnectCluster(ccp)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(*clusters.Payload[0].Data.ID)
	err = waitClusterTasksFinish(ctx, ct, clusters.Payload)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceClusterRead(ctx, d, meta)
}

func resourceClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)

	id := d.Id()
	gcp := cluster.NewGetClustersParams()
	gcp.RequestBody = &models.GetClustersRequestBody{
		Where: &models.ClusterWhereInput{
			ID: &id,
		},
	}
	clusters, err := ct.Api.Cluster.GetClusters(gcp)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(clusters.Payload) < 1 {
		d.SetId("")
		return diags
	}
	if err = d.Set("name", clusters.Payload[0].Name); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	ucp := cluster.NewUpdateClusterParams()
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	ip := d.Get("ip").(string)
	datacenterId := d.Get("datacenter_id").(string)
	id := d.Id()
	ucp.RequestBody = &models.ClusterUpdationParams{
		Where: &models.ClusterWhereInput{
			ID: &id,
		},
		Data: &models.ClusterUpdationParamsData{
			IP:           &ip,
			Username:     &username,
			Password:     &password,
			DatacenterID: &datacenterId,
		},
	}
	clusters, err := ct.Api.Cluster.UpdateCluster(ucp)
	if err != nil {
		return diag.FromErr(err)
	}
	err = waitClusterTasksFinish(ctx, ct, clusters.Payload)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceClusterRead(ctx, d, meta)
}

func resourceClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)
	dcp := cluster.NewDeleteClusterParams()
	id := d.Id()
	dcp.RequestBody = &models.ClusterDeletionParams{
		Where: &models.ClusterWhereInput{
			ID: &id,
		},
	}
	clusters, err := ct.Api.Cluster.DeleteCluster(dcp)
	if err != nil {
		return diag.FromErr(err)
	}
	taskIds := make([]string, 0)
	for _, c := range clusters.Payload {
		if c.TaskID != nil {
			taskIds = append(taskIds, *c.TaskID)
		}
	}
	_, err = ct.WaitTasksFinish(ctx, taskIds)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return diags
}

func waitClusterTasksFinish(ctx context.Context, ct *cloudtower.Client, clusters []*models.WithTaskCluster) error {
	taskIds := make([]string, 0)
	for _, c := range clusters {
		if c.TaskID != nil {
			taskIds = append(taskIds, *c.TaskID)
		}
	}
	_, err := ct.WaitTasksFinish(ctx, taskIds)
	return err
}
