package provider

import (
	"strings"
	"testing"

	"github.com/hasura/go-graphql-client"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

func TestPlanVMPlacementGroupMembershipUpdates(t *testing.T) {
	clusterID := "cluster-a"
	clusterName := "cluster-a"
	groupAID := "group-a"
	groupBID := "group-b"
	groupName := "group"
	vmID := "vm-a"
	otherVMID := "vm-b"

	updates, err := planVMPlacementGroupMembershipUpdates(
		vmID,
		clusterID,
		[]string{groupAID},
		[]string{groupBID},
		[]*models.VMPlacementGroup{
			{
				ID:      &groupAID,
				Name:    &groupName,
				Cluster: &models.NestedCluster{ID: &clusterID, Name: &clusterName},
				Vms: []*models.NestedVM{
					{ID: &vmID},
					{ID: &otherVMID},
				},
			},
			{
				ID:      &groupBID,
				Name:    &groupName,
				Cluster: &models.NestedCluster{ID: &clusterID, Name: &clusterName},
				Vms:     []*models.NestedVM{{ID: &otherVMID}},
			},
		},
	)
	if err != nil {
		t.Fatalf("plan membership updates: %v", err)
	}
	if len(updates) != 2 {
		t.Fatalf("got %d updates, want 2", len(updates))
	}
	if updates[0].groupID != groupAID || !sameStringSet(updates[0].vmIDs, []string{otherVMID}) {
		t.Fatalf("unexpected removal update: %#v", updates[0])
	}
	if updates[1].groupID != groupBID || !sameStringSet(updates[1].vmIDs, []string{otherVMID, vmID}) {
		t.Fatalf("unexpected addition update: %#v", updates[1])
	}
}

func TestPlanVMPlacementGroupMembershipUpdatesRejectsInvalidTargets(t *testing.T) {
	clusterA := "cluster-a"
	clusterB := "cluster-b"
	clusterName := "cluster"
	groupID := "group-a"
	groupName := "group-a"

	_, err := planVMPlacementGroupMembershipUpdates(
		"vm-a",
		clusterA,
		nil,
		[]string{groupID},
		[]*models.VMPlacementGroup{{
			ID:      &groupID,
			Name:    &groupName,
			Cluster: &models.NestedCluster{ID: &clusterB, Name: &clusterName},
		}},
	)
	if err == nil || !strings.Contains(err.Error(), "not in VM cluster") {
		t.Fatalf("expected a cross-cluster error, got %v", err)
	}

	_, err = planVMPlacementGroupMembershipUpdates("vm-a", clusterA, nil, []string{"missing"}, nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected a missing group error, got %v", err)
	}
}

func TestVMPlacementGroupMutationTypes(t *testing.T) {
	var mutation struct {
		UpdateVMPlacementGroup struct {
			ID graphql.String
		} `graphql:"updateVmPlacementGroup(data: $data, where: $where)"`
	}
	query, err := graphql.ConstructMutation(&mutation, map[string]interface{}{
		"data": VmPlacementGroupUpdateInput{
			Vms: &VmPlacementGroupVMsUpdateInput{
				Connect: []VmWhereUniqueInput{{"id": "vm-a"}},
			},
		},
		"where": VmPlacementGroupWhereUniqueInput{"id": "group-a"},
	}, graphql.OperationName("updateVmPlacementGroup"))
	if err != nil {
		t.Fatalf("construct mutation: %v", err)
	}
	for _, expected := range []string{
		"$data:VmPlacementGroupUpdateInput!",
		"$where:VmPlacementGroupWhereUniqueInput!",
		"updateVmPlacementGroup(data: $data, where: $where)",
	} {
		if !strings.Contains(query, expected) {
			t.Fatalf("mutation %q does not contain %q", query, expected)
		}
	}
}
