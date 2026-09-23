package provider

import (
	"reflect"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

func TestProviderRegistersVMPlacementGroup(t *testing.T) {
	provider := New("test")()
	if err := provider.InternalValidate(); err != nil {
		t.Fatalf("provider schema is invalid: %v", err)
	}
	if provider.ResourcesMap["cloudtower_vm_placement_group"] == nil {
		t.Fatal("cloudtower_vm_placement_group resource is not registered")
	}
	if provider.DataSourcesMap["cloudtower_vm_placement_group"] == nil {
		t.Fatal("cloudtower_vm_placement_group data source is not registered")
	}
}

func TestExpandVMPlacementGroupParams(t *testing.T) {
	resource := resourceVMPlacementGroup()
	d := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"name":        "group-a",
		"description": "description",
		"cluster_id":  "cluster-a",
		"enabled":     true,
		"required_host_rule": []interface{}{map[string]interface{}{
			"enabled":   true,
			"placement": hostRulePlacementSelected,
			"host_ids":  []interface{}{"host-a", "host-b"},
		}},
		"preferred_host_rule": []interface{}{map[string]interface{}{
			"enabled":   true,
			"placement": hostRulePlacementOutside,
			"host_ids":  []interface{}{"host-c"},
		}},
		"vm_affinity": []interface{}{map[string]interface{}{
			"enabled":     true,
			"enforcement": vmAffinityEnforcementPreferred,
			"placement":   vmAffinityPlacementDifferent,
		}},
	})

	create := expandVMPlacementGroupCreationParams(d)
	if create.VMHostMustEnabled == nil || !*create.VMHostMustEnabled || create.VMHostMustPolicy == nil || !*create.VMHostMustPolicy {
		t.Fatalf("unexpected required-host rule flags: enabled=%v policy=%v", create.VMHostMustEnabled, create.VMHostMustPolicy)
	}
	if create.MustHosts == nil || !sameStringSet(create.MustHosts.IDIn, []string{"host-a", "host-b"}) {
		t.Fatalf("unexpected must hosts: %#v", create.MustHosts)
	}
	if create.VMHostPreferEnabled == nil || !*create.VMHostPreferEnabled || create.VMHostPreferPolicy == nil || *create.VMHostPreferPolicy {
		t.Fatalf("unexpected avoided-host rule flags: enabled=%v policy=%v", create.VMHostPreferEnabled, create.VMHostPreferPolicy)
	}
	if create.PreferHosts == nil || !sameStringSet(create.PreferHosts.IDIn, []string{"host-c"}) {
		t.Fatalf("unexpected avoided hosts: %#v", create.PreferHosts)
	}

	update := expandVMPlacementGroupUpdationParamsData(d)
	if update.PreferHosts == nil || !sameStringSet(update.PreferHosts.IDIn, []string{"host-c"}) {
		t.Fatalf("unexpected updated preferred hosts: %#v", update.PreferHosts)
	}
	if update.VMVMPolicy == nil || *update.VMVMPolicy != models.VMVMPolicyPREFERDIFFERENT {
		t.Fatalf("unexpected VM policy: %#v", update.VMVMPolicy)
	}
	if update.Vms != nil {
		t.Fatalf("placement group resource update must not manage VM membership: %#v", update.Vms)
	}
	emptyMatch := hostWhereInputMatchingIDs(nil)
	if emptyMatch.ID == nil || *emptyMatch.ID != "" {
		t.Fatalf("an empty host list must not expand to an unfiltered host query: %#v", emptyMatch)
	}
}

func TestExpandVMPlacementGroupVMAffinity(t *testing.T) {
	tests := []struct {
		name        string
		enforcement string
		placement   string
		want        models.VMVMPolicy
	}{
		{name: "required same", enforcement: vmAffinityEnforcementRequired, placement: vmAffinityPlacementSameHost, want: models.VMVMPolicyMUSTSAME},
		{name: "required different", enforcement: vmAffinityEnforcementRequired, placement: vmAffinityPlacementDifferent, want: models.VMVMPolicyMUSTDIFFERENT},
		{name: "preferred same", enforcement: vmAffinityEnforcementPreferred, placement: vmAffinityPlacementSameHost, want: models.VMVMPolicyPREFERSAME},
		{name: "preferred different", enforcement: vmAffinityEnforcementPreferred, placement: vmAffinityPlacementDifferent, want: models.VMVMPolicyPREFERDIFFERENT},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy, enabled := expandVMPlacementGroupVMAffinity([]interface{}{map[string]interface{}{
				"enabled":     true,
				"enforcement": test.enforcement,
				"placement":   test.placement,
			}})
			if !enabled || policy != test.want {
				t.Fatalf("got policy=%s enabled=%t, want policy=%s enabled=true", policy, enabled, test.want)
			}
		})
	}

	policy, enabled := expandVMPlacementGroupVMAffinity([]interface{}{map[string]interface{}{
		"enabled":     false,
		"enforcement": vmAffinityEnforcementRequired,
		"placement":   vmAffinityPlacementSameHost,
	}})
	if enabled || policy != models.VMVMPolicyMUSTSAME {
		t.Fatalf("disabled affinity got policy=%s enabled=%t", policy, enabled)
	}
}

func TestFlattenVMPlacementGroup(t *testing.T) {
	id := "group-a"
	name := "group-a"
	description := "description"
	clusterID := "cluster-a"
	clusterName := "cluster-a"
	enabled := true
	mustEnabled := true
	mustPolicy := false
	preferEnabled := true
	preferPolicy := true
	vmPolicyEnabled := true
	vmPolicy := models.VMVMPolicyMUSTSAME
	hostA := "host-a"
	hostB := "host-b"
	vmA := "vm-a"

	group := &models.VMPlacementGroup{
		ID:                    &id,
		Name:                  &name,
		Description:           &description,
		Cluster:               &models.NestedCluster{ID: &clusterID, Name: &clusterName},
		Enabled:               &enabled,
		VMHostMustEnabled:     &mustEnabled,
		VMHostMustPolicy:      &mustPolicy,
		VMHostMustHostUuids:   []*models.NestedHost{{ID: &hostA}},
		VMHostPreferEnabled:   &preferEnabled,
		VMHostPreferPolicy:    &preferPolicy,
		VMHostPreferHostUuids: []*models.NestedHost{{ID: &hostB}},
		VMVMPolicyEnabled:     &vmPolicyEnabled,
		VMVMPolicy:            &vmPolicy,
		Vms:                   []*models.NestedVM{{ID: &vmA}},
	}

	value := flattenVMPlacementGroup(group)
	wantRequiredRule := []interface{}{map[string]interface{}{
		"enabled":   true,
		"placement": hostRulePlacementOutside,
		"host_ids":  []string{"host-a"},
	}}
	if !reflect.DeepEqual(value["required_host_rule"], wantRequiredRule) {
		t.Fatalf("unexpected required host rule: %#v", value)
	}
	wantPreferredRule := []interface{}{map[string]interface{}{
		"enabled":   true,
		"placement": hostRulePlacementSelected,
		"host_ids":  []string{"host-b"},
	}}
	if !reflect.DeepEqual(value["preferred_host_rule"], wantPreferredRule) {
		t.Fatalf("unexpected preferred host rule: %#v", value)
	}
	wantAffinity := []interface{}{map[string]interface{}{
		"enabled":     true,
		"enforcement": vmAffinityEnforcementRequired,
		"placement":   vmAffinityPlacementSameHost,
	}}
	if !reflect.DeepEqual(value["vm_affinity"], wantAffinity) {
		t.Fatalf("unexpected VM affinity: %#v", value)
	}
	if !reflect.DeepEqual(value["vm_ids"], []string{"vm-a"}) {
		t.Fatalf("unexpected VM IDs: %#v", value)
	}
}

func TestFlattenVMPlacementGroupPreservesDisabledRules(t *testing.T) {
	disabled := false
	mustPolicy := false
	preferPolicy := true
	vmPolicy := models.VMVMPolicyPREFERDIFFERENT
	mustHostID := "host-a"
	preferHostID := "host-b"
	value := flattenVMPlacementGroup(&models.VMPlacementGroup{
		VMHostMustEnabled:     &disabled,
		VMHostMustPolicy:      &mustPolicy,
		VMHostMustHostUuids:   []*models.NestedHost{{ID: &mustHostID}},
		VMHostPreferEnabled:   &disabled,
		VMHostPreferPolicy:    &preferPolicy,
		VMHostPreferHostUuids: []*models.NestedHost{{ID: &preferHostID}},
		VMVMPolicyEnabled:     &disabled,
		VMVMPolicy:            &vmPolicy,
	})
	wantRequired := []interface{}{map[string]interface{}{
		"enabled":   false,
		"placement": hostRulePlacementOutside,
		"host_ids":  []string{"host-a"},
	}}
	if !reflect.DeepEqual(value["required_host_rule"], wantRequired) {
		t.Fatalf("disabled required host rule was not preserved: %#v", value)
	}
	wantPreferred := []interface{}{map[string]interface{}{
		"enabled":   false,
		"placement": hostRulePlacementSelected,
		"host_ids":  []string{"host-b"},
	}}
	if !reflect.DeepEqual(value["preferred_host_rule"], wantPreferred) {
		t.Fatalf("disabled preferred host rule was not preserved: %#v", value)
	}
	wantAffinity := []interface{}{map[string]interface{}{
		"enabled":     false,
		"enforcement": vmAffinityEnforcementPreferred,
		"placement":   vmAffinityPlacementDifferent,
	}}
	if !reflect.DeepEqual(value["vm_affinity"], wantAffinity) {
		t.Fatalf("disabled VM affinity rule was not preserved: %#v", value)
	}
}

func TestVMPlacementGroupRuleSchemaValidation(t *testing.T) {
	resource := resourceVMPlacementGroup()
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "all rules disabled",
			config: map[string]interface{}{
				"name": "group-a", "cluster_id": "cluster-a",
				"required_host_rule": []interface{}{map[string]interface{}{
					"enabled": false,
				}},
				"preferred_host_rule": []interface{}{map[string]interface{}{
					"enabled": false,
				}},
				"vm_affinity": []interface{}{map[string]interface{}{
					"enabled": false,
				}},
			},
			wantErr: true,
		},
		{
			name: "enabled host rule without hosts",
			config: map[string]interface{}{
				"name": "group-a", "cluster_id": "cluster-a",
				"required_host_rule": []interface{}{map[string]interface{}{
					"enabled": true,
				}},
				"preferred_host_rule": []interface{}{map[string]interface{}{
					"enabled": false,
				}},
				"vm_affinity": []interface{}{map[string]interface{}{
					"enabled": false,
				}},
			},
			wantErr: true,
		},
		{
			name: "VM affinity only",
			config: map[string]interface{}{
				"name": "group-a", "cluster_id": "cluster-a",
				"required_host_rule": []interface{}{map[string]interface{}{
					"enabled": false,
				}},
				"preferred_host_rule": []interface{}{map[string]interface{}{
					"enabled": false,
				}},
				"vm_affinity": []interface{}{map[string]interface{}{
					"enabled":     true,
					"enforcement": vmAffinityEnforcementRequired,
					"placement":   vmAffinityPlacementDifferent,
				}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, resource.Schema, test.config)
			err := validateVMPlacementGroupRuleValues(
				d.Get("required_host_rule"),
				d.Get("preferred_host_rule"),
				d.Get("vm_affinity"),
			)
			if (err != nil) != test.wantErr {
				t.Fatalf("validation error = %v, want error = %t", err, test.wantErr)
			}
		})
	}
}

func TestExpandVMPlacementGroupWhereInputIncludesFalse(t *testing.T) {
	datasource := dataSourceVMPlacementGroup()
	d := schema.TestResourceDataRaw(t, datasource.Schema, map[string]interface{}{
		"enabled": false,
	})
	where, err := expandVMPlacementGroupWhereInput(d)
	if err != nil {
		t.Fatalf("expand VM placement group filter: %v", err)
	}
	if where.Enabled == nil || *where.Enabled {
		t.Fatalf("enabled=false filter was lost: %#v", where.Enabled)
	}
}

func TestExpandVMWhereInputWithPlacementGroups(t *testing.T) {
	datasource := dataSourceVm()
	d := schema.TestResourceDataRaw(t, datasource.Schema, map[string]interface{}{
		"placement_group_id_in": []interface{}{"group-a", "group-b"},
	})
	where, err := expandVmWhereInput(d)
	if err != nil {
		t.Fatalf("expand VM filter: %v", err)
	}
	if where.VMPlacementGroupSome == nil || !sameStringSet(where.VMPlacementGroupSome.IDIn, []string{"group-a", "group-b"}) {
		t.Fatalf("unexpected VM placement group filter: %#v", where.VMPlacementGroupSome)
	}
}

func TestVMPlacementGroupIDsConfigurationDetection(t *testing.T) {
	setType := cty.Set(cty.String)
	tests := []struct {
		name       string
		value      cty.Value
		configured bool
	}{
		{name: "omitted", value: cty.NullVal(setType), configured: false},
		{name: "unknown", value: cty.UnknownVal(setType), configured: true},
		{name: "explicit empty", value: cty.SetValEmpty(cty.String), configured: true},
		{name: "configured", value: cty.SetVal([]cty.Value{cty.StringVal("group-a")}), configured: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rawConfig := cty.ObjectVal(map[string]cty.Value{
				"placement_group_ids": test.value,
			})
			configured := rawConfigFieldConfigured(rawConfig, "placement_group_ids")
			if configured != test.configured {
				t.Fatalf("configured = %t, want %t", configured, test.configured)
			}
		})
	}
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	values := make(map[string]bool, len(left))
	for _, value := range left {
		values[value] = true
	}
	for _, value := range right {
		if !values[value] {
			return false
		}
	}
	return true
}
