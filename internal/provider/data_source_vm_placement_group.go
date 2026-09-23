package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_placement_group"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

func dataSourceVMPlacementGroup() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower VM placement group data source.",

		ReadContext: dataSourceVMPlacementGroupRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id_in"},
				Description:   "Filter VM placement groups by ID.",
			},
			"id_in": {
				Type:          schema.TypeList,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"id"},
				Description:   "Filter VM placement groups by IDs.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_in"},
				Description:   "Filter VM placement groups by name.",
			},
			"name_in": {
				Type:          schema.TypeList,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"name"},
				Description:   "Filter VM placement groups by names.",
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter VM placement groups by a name substring.",
			},
			"cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"cluster_id_in"},
				Description:   "Filter VM placement groups by cluster ID.",
			},
			"cluster_id_in": {
				Type:          schema.TypeList,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"cluster_id"},
				Description:   "Filter VM placement groups by cluster IDs.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Filter VM placement groups by whether the group is enabled.",
			},
			"vm_placement_groups": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The matching VM placement groups.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The VM placement group ID.",
					},
					"name": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The VM placement group name.",
					},
					"description": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The VM placement group description.",
					},
					"cluster_id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The cluster ID of the VM placement group.",
					},
					"enabled": {
						Type:        schema.TypeBool,
						Computed:    true,
						Description: "Whether the VM placement group is enabled.",
					},
					"required_host_rule":  dataSourceVMPlacementGroupHostRuleSchema("Required placement rule between VMs in this group and selected hosts."),
					"preferred_host_rule": dataSourceVMPlacementGroupHostRuleSchema("Preferred placement rule between VMs in this group and selected hosts."),
					"vm_affinity": {
						Type:        schema.TypeList,
						Computed:    true,
						Description: "The placement rule between VMs in this group.",
						Elem: &schema.Resource{Schema: map[string]*schema.Schema{
							"enabled": {
								Type:        schema.TypeBool,
								Computed:    true,
								Description: "Whether the VM affinity rule is enabled.",
							},
							"enforcement": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "Whether the VM affinity rule is REQUIRED or PREFERRED.",
							},
							"placement": {
								Type:        schema.TypeString,
								Computed:    true,
								Description: "Whether VMs should run on the SAME_HOST or DIFFERENT_HOSTS.",
							},
						}},
					},
					"vm_ids": {
						Type:        schema.TypeSet,
						Computed:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Description: "The IDs of VMs in the placement group.",
					},
				}},
			},
		},
	}
}

func dataSourceVMPlacementGroupHostRuleSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: description,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the host placement rule is enabled.",
			},
			"placement": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Whether VMs should run ON_SELECTED_HOSTS or OUTSIDE_SELECTED_HOSTS.",
			},
			"host_ids": {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Hosts selected by this rule.",
			},
		}},
	}
}

func dataSourceVMPlacementGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)

	where, err := expandVMPlacementGroupWhereInput(d)
	if err != nil {
		return diag.FromErr(err)
	}

	params := vm_placement_group.NewGetVMPlacementGroupsParamsWithContext(ctx)
	params.RequestBody = &models.GetVMPlacementGroupsRequestBody{Where: where}
	groups, err := ct.Api.VMPlacementGroup.GetVMPlacementGroups(params)
	if err != nil {
		return diag.FromErr(err)
	}

	output := make([]map[string]interface{}, 0, len(groups.Payload))
	for _, group := range groups.Payload {
		output = append(output, flattenVMPlacementGroup(group))
	}
	if err := d.Set("vm_placement_groups", output); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().UnixNano(), 10))
	return nil
}

func expandVMPlacementGroupWhereInput(d *schema.ResourceData) (*models.VMPlacementGroupWhereInput, error) {
	where := &models.VMPlacementGroupWhereInput{}
	if id := d.Get("id").(string); id != "" {
		where.ID = &id
	} else {
		ids, err := helper.SliceInterfacesToTypeSlice[string](d.Get("id_in").([]interface{}))
		if err != nil {
			return nil, err
		}
		where.IDIn = ids
	}
	if name := d.Get("name").(string); name != "" {
		where.Name = &name
	} else {
		names, err := helper.SliceInterfacesToTypeSlice[string](d.Get("name_in").([]interface{}))
		if err != nil {
			return nil, err
		}
		where.NameIn = names
	}
	if nameContains := d.Get("name_contains").(string); nameContains != "" {
		where.NameContains = &nameContains
	}
	if clusterID := d.Get("cluster_id").(string); clusterID != "" {
		where.Cluster = &models.ClusterWhereInput{ID: &clusterID}
	} else {
		clusterIDs, err := helper.SliceInterfacesToTypeSlice[string](d.Get("cluster_id_in").([]interface{}))
		if err != nil {
			return nil, err
		}
		if len(clusterIDs) > 0 {
			where.Cluster = &models.ClusterWhereInput{IDIn: clusterIDs}
		}
	}
	if enabled, ok := d.GetOkExists("enabled"); ok {
		value := enabled.(bool)
		where.Enabled = &value
	}

	return where, nil
}

func flattenVMPlacementGroup(group *models.VMPlacementGroup) map[string]interface{} {
	result := map[string]interface{}{
		"vm_ids": []string{},
	}
	if group.ID != nil {
		result["id"] = *group.ID
	}
	if group.Name != nil {
		result["name"] = *group.Name
	}
	if group.Description != nil {
		result["description"] = *group.Description
	}
	if group.Cluster != nil && group.Cluster.ID != nil {
		result["cluster_id"] = *group.Cluster.ID
	}
	if group.Enabled != nil {
		result["enabled"] = *group.Enabled
	}

	mustHostIDs := nestedHostIDs(group.VMHostMustHostUuids)
	mustEnabled := group.VMHostMustEnabled != nil && *group.VMHostMustEnabled
	mustOnSelectedHosts := group.VMHostMustPolicy == nil || *group.VMHostMustPolicy
	result["required_host_rule"] = flattenVMPlacementGroupHostRule(mustEnabled, mustOnSelectedHosts, mustHostIDs)

	preferHostIDs := nestedHostIDs(group.VMHostPreferHostUuids)
	preferEnabled := group.VMHostPreferEnabled != nil && *group.VMHostPreferEnabled
	preferOnSelectedHosts := group.VMHostPreferPolicy == nil || *group.VMHostPreferPolicy
	result["preferred_host_rule"] = flattenVMPlacementGroupHostRule(preferEnabled, preferOnSelectedHosts, preferHostIDs)

	vmPolicy := models.VMVMPolicyMUSTSAME
	if group.VMVMPolicy != nil {
		vmPolicy = *group.VMVMPolicy
	}
	vmPolicyEnabled := group.VMVMPolicyEnabled != nil && *group.VMVMPolicyEnabled
	result["vm_affinity"] = flattenVMPlacementGroupVMAffinity(vmPolicy, vmPolicyEnabled)

	vmIDs := make([]string, 0, len(group.Vms))
	for _, vm := range group.Vms {
		if vm != nil && vm.ID != nil {
			vmIDs = append(vmIDs, *vm.ID)
		}
	}
	result["vm_ids"] = vmIDs

	return result
}

func nestedHostIDs(hosts []*models.NestedHost) []string {
	hostIDs := make([]string, 0, len(hosts))
	for _, host := range hosts {
		if host != nil && host.ID != nil {
			hostIDs = append(hostIDs, *host.ID)
		}
	}
	return hostIDs
}

func flattenVMPlacementGroupHostRule(enabled, onSelectedHosts bool, hostIDs []string) []interface{} {
	placement := hostRulePlacementOutside
	if onSelectedHosts {
		placement = hostRulePlacementSelected
	}
	return []interface{}{map[string]interface{}{
		"enabled":   enabled,
		"placement": placement,
		"host_ids":  hostIDs,
	}}
}

func flattenVMPlacementGroupVMAffinity(policy models.VMVMPolicy, enabled bool) []interface{} {
	enforcement := vmAffinityEnforcementPreferred
	placement := vmAffinityPlacementDifferent
	switch policy {
	case models.VMVMPolicyMUSTSAME:
		enforcement = vmAffinityEnforcementRequired
		placement = vmAffinityPlacementSameHost
	case models.VMVMPolicyMUSTDIFFERENT:
		enforcement = vmAffinityEnforcementRequired
	case models.VMVMPolicyPREFERSAME:
		placement = vmAffinityPlacementSameHost
	}
	return []interface{}{map[string]interface{}{
		"enabled":     enabled,
		"enforcement": enforcement,
		"placement":   placement,
	}}
}
