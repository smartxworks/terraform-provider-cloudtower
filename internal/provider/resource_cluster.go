package provider

import (
	"context"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/cluster"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/label"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var resourceTypeCluster = "cluster"

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
			"label_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "cluster's terraform label",
			},
		},
	}
}

func resourceClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	ip := d.Get("ip").(string)
	datacenterId := d.Get("datacenter_id").(string)
	// check if cluster is connected
	gcp := cluster.NewGetClustersParams()
	gcp.RequestBody = &models.GetClustersRequestBody{
		Where: &models.ClusterWhereInput{
			IP: &ip,
		},
	}
	clusters, err := ct.Api.Cluster.GetClusters(gcp)
	if err != nil {
		return diag.FromErr(err)
	}
	var labelId string
	if len(clusters.Payload) > 0 {
		// check uniqueness by label
		labels, err := helper.GetUniquenessLabel(ct.Api, resourceTypeCluster, *clusters.Payload[0].ID)
		if err != nil {
			return diag.FromErr(err)
		}
		if len(labels.Payload) > 0 {
			if *labels.Payload[0].ClusterNum >= 1 {
				// if label exist and already connect to a cluster
				// means this cluster is already under management
				return diag.Errorf("cluster %s is already under management", ip)
			}
			// if label exist but not connect to any cluster
			// means this cluster may under terraform management sometime before, but remove by other reason
			// re add the resource to this cluster
			labelId = *labels.Payload[0].ID
		}
		d.SetId(*clusters.Payload[0].ID)
	} else {
		// if cluster ip is not connected, connect it
		ccp := cluster.NewConnectClusterParams()
		// cannot connect the same
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
		err = waitClusterTasksFinish(ct, clusters.Payload)
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId(*clusters.Payload[0].Data.ID)
	}
	// if label not exist, mean this cluster is not under management
	// create a new label
	if labelId == "" {
		labels, err := helper.CreateUniquenessLabel(ct.Api, resourceTypeCluster, *clusters.Payload[0].ID)
		if err != nil {
			return diag.FromErr(err)
		}
		labelId = *labels.Payload[0].Data.ID
	}
	altrp := label.NewAddLabelsToResourcesParams()
	altrp.RequestBody = &models.AddLabelsToResourcesParams{
		Where: &models.LabelWhereInput{
			ID: &labelId,
		},
		Data: &models.AddLabelsToResourcesParamsData{
			Clusters: &models.ClusterWhereInput{
				ID: &d.State().ID,
			},
		},
	}
	_, err = ct.Api.Label.AddLabelsToResources(altrp)
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
	if id == "" {
		// if id is null, try to read by ip
		ip := d.Get("ip").(string)
		gcp.RequestBody = &models.GetClustersRequestBody{
			Where: &models.ClusterWhereInput{
				IP: &ip,
			},
		}
	} else {
		gcp.RequestBody = &models.GetClustersRequestBody{
			Where: &models.ClusterWhereInput{
				ID: &id,
			},
		}
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
	err = waitClusterTasksFinish(ct, clusters.Payload)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceClusterRead(ctx, d, meta)
}

func resourceClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	ct := meta.(*cloudtower.Client)
	id := d.Id()
	// remove label
	_, err := helper.DeleteUniquenessLabel(ct.Api, resourceTypeCluster, id)
	if err != nil {
		return diag.FromErr(err)
	}
	dcp := cluster.NewDeleteClusterParams()
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
	_, err = ct.WaitTasksFinish(taskIds)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return diags
}

func waitClusterTasksFinish(ct *cloudtower.Client, clusters []*models.WithTaskCluster) error {
	taskIds := make([]string, 0)
	for _, c := range clusters {
		if c.TaskID != nil {
			taskIds = append(taskIds, *c.TaskID)
		}
	}
	_, err := ct.WaitTasksFinish(taskIds)
	return err
}
