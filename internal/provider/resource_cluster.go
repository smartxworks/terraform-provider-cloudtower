package provider

import (
	"context"
	"github.com/Yuyz0112/cloudtower-go-sdk/client/cluster"
	"github.com/Yuyz0112/cloudtower-go-sdk/models"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"

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
			"ip": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"password": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"datacenter_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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
	ccp.RequestBody = []*models.ClusterCreationParams{&models.ClusterCreationParams{
		IP:           &ip,
		Username:     &username,
		Password:     &password,
		DatacenterID: datacenterId,
	}}
	clusters, err := ct.Api.Cluster.ConnectCluster(ccp)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(*clusters.Payload[0].Data.ID)
	waitClusterTasksFinish(ct, clusters.Payload)

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
			IP:           ip,
			Username:     username,
			Password:     password,
			DatacenterID: datacenterId,
		},
	}
	clusters, err := ct.Api.Cluster.UpdateCluster(ucp)
	if err != nil {
		return diag.FromErr(err)
	}
	waitClusterTasksFinish(ct, clusters.Payload)

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
	for _, cluster := range clusters.Payload {
		if cluster.TaskID != nil {
			taskIds = append(taskIds, *cluster.TaskID)
		}
	}
	ct.WaitTasksFinish(taskIds)

	d.SetId("")
	return diags
}

func waitClusterTasksFinish(ct *cloudtower.Client, clusters []*models.WithTaskCluster) {
	taskIds := make([]string, 0)
	for _, cluster := range clusters {
		if cluster.TaskID != nil {
			taskIds = append(taskIds, *cluster.TaskID)
		}
	}
	ct.WaitTasksFinish(taskIds)
}
