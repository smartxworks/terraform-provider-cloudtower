package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_placement_group"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

const (
	vmAffinityEnforcementRequired  = "REQUIRED"
	vmAffinityEnforcementPreferred = "PREFERRED"
	vmAffinityPlacementSameHost    = "SAME_HOST"
	vmAffinityPlacementDifferent   = "DIFFERENT_HOSTS"
	hostRulePlacementSelected      = "ON_SELECTED_HOSTS"
	hostRulePlacementOutside       = "OUTSIDE_SELECTED_HOSTS"
)

func resourceVMPlacementGroup() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower VM placement group resource. At least one host placement or VM affinity rule must be enabled. VM membership is managed from the `cloudtower_vm` resource through its `placement_group_ids` attribute.",

		CreateContext: resourceVMPlacementGroupCreate,
		ReadContext:   resourceVMPlacementGroupRead,
		UpdateContext: resourceVMPlacementGroupUpdate,
		DeleteContext: resourceVMPlacementGroupDelete,
		CustomizeDiff: validateVMPlacementGroupRules,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The VM placement group ID.",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The VM placement group name.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The VM placement group description.",
			},
			"cluster_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The cluster ID of the VM placement group.",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the VM placement group is enabled.",
			},
			"required_host_rule":  vmPlacementGroupHostRuleSchema("Required placement rule between VMs in this group and selected hosts."),
			"preferred_host_rule": vmPlacementGroupHostRuleSchema("Preferred placement rule between VMs in this group and selected hosts."),
			"vm_affinity": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				MaxItems:    1,
				Description: "Placement rule between VMs in this group.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"enabled": {
						Type:        schema.TypeBool,
						Required:    true,
						Description: "Whether the VM affinity rule is enabled.",
					},
					"enforcement": {
						Type:         schema.TypeString,
						Optional:     true,
						Default:      vmAffinityEnforcementPreferred,
						ValidateFunc: validation.StringInSlice([]string{vmAffinityEnforcementRequired, vmAffinityEnforcementPreferred}, false),
						Description:  "Whether the VM affinity rule is REQUIRED or PREFERRED.",
					},
					"placement": {
						Type:         schema.TypeString,
						Optional:     true,
						Default:      vmAffinityPlacementSameHost,
						ValidateFunc: validation.StringInSlice([]string{vmAffinityPlacementSameHost, vmAffinityPlacementDifferent}, false),
						Description:  "Whether VMs should run on the SAME_HOST or DIFFERENT_HOSTS.",
					},
				}},
			},
		},
	}
}

func vmPlacementGroupHostRuleSchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		MinItems:    1,
		MaxItems:    1,
		Description: description,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the host placement rule is enabled.",
			},
			"placement": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      hostRulePlacementSelected,
				ValidateFunc: validation.StringInSlice([]string{hostRulePlacementSelected, hostRulePlacementOutside}, false),
				Description:  "Whether VMs should run ON_SELECTED_HOSTS or OUTSIDE_SELECTED_HOSTS.",
			},
			"host_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString, ValidateFunc: validation.StringIsNotEmpty},
				Description: "Hosts selected by this rule. At least one host is required when the rule is enabled.",
			},
		}},
	}
}

func validateVMPlacementGroupRules(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	return validateVMPlacementGroupRuleValues(
		d.Get("required_host_rule"),
		d.Get("preferred_host_rule"),
		d.Get("vm_affinity"),
	)
}

func validateVMPlacementGroupRuleValues(requiredHostRule, preferredHostRule, vmAffinity interface{}) error {
	required := singletonRuleValue(requiredHostRule)
	preferred := singletonRuleValue(preferredHostRule)
	affinity := singletonRuleValue(vmAffinity)
	if !required["enabled"].(bool) && !preferred["enabled"].(bool) && !affinity["enabled"].(bool) {
		return fmt.Errorf("at least one of required_host_rule, preferred_host_rule, or vm_affinity must be enabled")
	}
	for name, rule := range map[string]map[string]interface{}{
		"required_host_rule":  required,
		"preferred_host_rule": preferred,
	} {
		if rule["enabled"].(bool) && len(stringSetValues(rule["host_ids"])) == 0 {
			return fmt.Errorf("%s.host_ids must contain at least one host when the rule is enabled", name)
		}
	}
	return nil
}

func singletonRuleValue(value interface{}) map[string]interface{} {
	return value.([]interface{})[0].(map[string]interface{})
}

func resourceVMPlacementGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	params := vm_placement_group.NewCreateVMPlacementGroupParamsWithContext(ctx)
	params.RequestBody = []*models.VMPlacementGroupCreationParams{expandVMPlacementGroupCreationParams(d)}

	groups, err := ct.Api.VMPlacementGroup.CreateVMPlacementGroup(params)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(groups.Payload) != 1 || groups.Payload[0] == nil || groups.Payload[0].Data == nil || groups.Payload[0].Data.ID == nil {
		return diag.Errorf("create VM placement group returned an invalid response")
	}

	d.SetId(*groups.Payload[0].Data.ID)
	if err := waitVMPlacementGroupTasksFinish(ctx, ct, groups.Payload); err != nil {
		return diag.FromErr(err)
	}
	return resourceVMPlacementGroupRead(ctx, d, meta)
}

func resourceVMPlacementGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	id := d.Id()
	params := vm_placement_group.NewGetVMPlacementGroupsParamsWithContext(ctx)
	params.RequestBody = &models.GetVMPlacementGroupsRequestBody{
		Where: &models.VMPlacementGroupWhereInput{ID: &id},
	}

	groups, err := ct.Api.VMPlacementGroup.GetVMPlacementGroups(params)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(groups.Payload) == 0 {
		d.SetId("")
		return nil
	}

	group := flattenVMPlacementGroup(groups.Payload[0])
	for _, field := range []string{
		"id", "name", "description", "cluster_id", "enabled",
		"required_host_rule", "preferred_host_rule", "vm_affinity",
	} {
		if err := d.Set(field, group[field]); err != nil {
			return diag.FromErr(fmt.Errorf("set VM placement group field %q: %w", field, err))
		}
	}

	return nil
}

func resourceVMPlacementGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	id := d.Id()
	params := vm_placement_group.NewUpdateVMPlacementGroupParamsWithContext(ctx)
	params.RequestBody = &models.VMPlacementGroupUpdationParams{
		Where: &models.VMPlacementGroupWhereInput{ID: &id},
		Data:  expandVMPlacementGroupUpdationParamsData(d),
	}

	groups, err := ct.Api.VMPlacementGroup.UpdateVMPlacementGroup(params)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := waitVMPlacementGroupTasksFinish(ctx, ct, groups.Payload); err != nil {
		return diag.FromErr(err)
	}
	return resourceVMPlacementGroupRead(ctx, d, meta)
}

func resourceVMPlacementGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ct := meta.(*cloudtower.Client)
	id := d.Id()
	params := vm_placement_group.NewDeleteVMPlacementGroupParamsWithContext(ctx)
	params.RequestBody = &models.VMPlacementGroupDeletionParams{
		Where: &models.VMPlacementGroupWhereInput{ID: &id},
	}

	groups, err := ct.Api.VMPlacementGroup.DeleteVMPlacementGroup(params)
	if err != nil {
		return diag.FromErr(err)
	}
	taskIDs := make([]string, 0, len(groups.Payload))
	for _, group := range groups.Payload {
		if group != nil && group.TaskID != nil {
			taskIDs = append(taskIDs, *group.TaskID)
		}
	}
	if _, err := ct.WaitTasksFinish(ctx, taskIDs); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func expandVMPlacementGroupCreationParams(d *schema.ResourceData) *models.VMPlacementGroupCreationParams {
	name := d.Get("name").(string)
	description := d.Get("description").(string)
	clusterID := d.Get("cluster_id").(string)
	enabled := d.Get("enabled").(bool)
	mustHostIDs, mustEnabled, mustPolicy := expandVMPlacementGroupHostRule(d.Get("required_host_rule"))
	preferHostIDs, preferEnabled, preferPolicy := expandVMPlacementGroupHostRule(d.Get("preferred_host_rule"))
	vmPolicy, vmPolicyEnabled := expandVMPlacementGroupVMAffinity(d.Get("vm_affinity"))

	params := &models.VMPlacementGroupCreationParams{
		Name:                &name,
		Description:         &description,
		ClusterID:           &clusterID,
		Enabled:             &enabled,
		VMHostMustEnabled:   &mustEnabled,
		VMHostMustPolicy:    &mustPolicy,
		VMHostPreferEnabled: &preferEnabled,
		VMHostPreferPolicy:  &preferPolicy,
		VMVMPolicyEnabled:   &vmPolicyEnabled,
		VMVMPolicy:          &vmPolicy,
	}
	if len(mustHostIDs) > 0 {
		params.MustHosts = &models.HostWhereInput{IDIn: mustHostIDs}
	}
	if len(preferHostIDs) > 0 {
		params.PreferHosts = &models.HostWhereInput{IDIn: preferHostIDs}
	}
	return params
}

func expandVMPlacementGroupHostRule(value interface{}) ([]string, bool, bool) {
	rule := singletonRuleValue(value)
	return stringSetValues(rule["host_ids"]), rule["enabled"].(bool), rule["placement"].(string) == hostRulePlacementSelected
}

func expandVMPlacementGroupVMAffinity(value interface{}) (models.VMVMPolicy, bool) {
	rule := singletonRuleValue(value)
	enabled := rule["enabled"].(bool)
	required := rule["enforcement"].(string) == vmAffinityEnforcementRequired
	sameHost := rule["placement"].(string) == vmAffinityPlacementSameHost
	switch {
	case required && sameHost:
		return models.VMVMPolicyMUSTSAME, enabled
	case required:
		return models.VMVMPolicyMUSTDIFFERENT, enabled
	case sameHost:
		return models.VMVMPolicyPREFERSAME, enabled
	default:
		return models.VMVMPolicyPREFERDIFFERENT, enabled
	}
}

func expandVMPlacementGroupUpdationParamsData(d *schema.ResourceData) *models.VMPlacementGroupUpdationParamsData {
	creation := expandVMPlacementGroupCreationParams(d)
	data := &models.VMPlacementGroupUpdationParamsData{
		Name:                creation.Name,
		Description:         creation.Description,
		Enabled:             creation.Enabled,
		VMHostMustEnabled:   creation.VMHostMustEnabled,
		VMHostMustPolicy:    creation.VMHostMustPolicy,
		VMHostPreferEnabled: creation.VMHostPreferEnabled,
		VMHostPreferPolicy:  creation.VMHostPreferPolicy,
		VMVMPolicyEnabled:   creation.VMVMPolicyEnabled,
		VMVMPolicy:          creation.VMVMPolicy,
	}
	data.MustHosts = hostWhereInputMatchingIDs(stringSetValues(singletonRuleValue(d.Get("required_host_rule"))["host_ids"]))
	data.PreferHosts = hostWhereInputMatchingIDs(stringSetValues(singletonRuleValue(d.Get("preferred_host_rule"))["host_ids"]))
	return data
}

func hostWhereInputMatchingIDs(ids []string) *models.HostWhereInput {
	if len(ids) > 0 {
		return &models.HostWhereInput{IDIn: ids}
	}
	emptyID := ""
	return &models.HostWhereInput{ID: &emptyID}
}

func stringSetValues(value interface{}) []string {
	set, ok := value.(*schema.Set)
	if !ok || set == nil {
		return nil
	}
	values := make([]string, 0, set.Len())
	for _, item := range set.List() {
		values = append(values, item.(string))
	}
	return values
}

func resourceDataFieldConfigured(d *schema.ResourceData, field string) bool {
	return rawConfigFieldConfigured(d.GetRawConfig(), field)
}

func rawConfigFieldConfigured(rawConfig cty.Value, field string) bool {
	if rawConfig.IsNull() || !rawConfig.IsKnown() || !rawConfig.Type().IsObjectType() || !rawConfig.Type().HasAttribute(field) {
		return false
	}
	value := rawConfig.GetAttr(field)
	return !value.IsNull()
}

func waitVMPlacementGroupTasksFinish(ctx context.Context, ct *cloudtower.Client, groups []*models.WithTaskVMPlacementGroup) error {
	taskIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		if group != nil && group.TaskID != nil {
			taskIDs = append(taskIDs, *group.TaskID)
		}
	}
	_, err := ct.WaitTasksFinish(ctx, taskIDs)
	return err
}
