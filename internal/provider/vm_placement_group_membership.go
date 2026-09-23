package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/utils"
	"github.com/hasura/go-graphql-client"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_placement_group"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

type VmPlacementGroupUpdateInput struct {
	Vms *VmPlacementGroupVMsUpdateInput `json:"vms,omitempty"`
}

type VmPlacementGroupVMsUpdateInput struct {
	Connect []VmWhereUniqueInput `json:"connect"`
}

type VmPlacementGroupWhereUniqueInput map[string]interface{}

type vmPlacementGroupMembershipUpdate struct {
	groupID string
	vmIDs   []string
}

var vmPlacementGroupMembershipMu sync.Mutex

func syncVMPlacementGroups(ctx context.Context, ct *cloudtower.Client, vm *models.VM, desiredGroupIDs []string) error {
	if vm == nil || vm.ID == nil || vm.Cluster == nil || vm.Cluster.ID == nil {
		return fmt.Errorf("cannot update VM placement groups without VM and cluster IDs")
	}

	currentGroupIDs := make([]string, 0, len(vm.VMPlacementGroup))
	for _, group := range vm.VMPlacementGroup {
		if group != nil && group.ID != nil {
			currentGroupIDs = append(currentGroupIDs, *group.ID)
		}
	}
	changedGroupIDs := symmetricDifference(currentGroupIDs, desiredGroupIDs)
	if len(changedGroupIDs) == 0 {
		return nil
	}
	vmPlacementGroupMembershipMu.Lock()
	defer vmPlacementGroupMembershipMu.Unlock()

	params := vm_placement_group.NewGetVMPlacementGroupsParamsWithContext(ctx)
	params.RequestBody = &models.GetVMPlacementGroupsRequestBody{
		Where: &models.VMPlacementGroupWhereInput{IDIn: changedGroupIDs},
	}
	groups, err := ct.Api.VMPlacementGroup.GetVMPlacementGroups(params)
	if err != nil {
		return err
	}

	updates, err := planVMPlacementGroupMembershipUpdates(
		*vm.ID,
		*vm.Cluster.ID,
		currentGroupIDs,
		desiredGroupIDs,
		groups.Payload,
	)
	if err != nil {
		return err
	}

	errs := make([]error, len(updates))
	var wg sync.WaitGroup
	for index := range updates {
		index := index
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[index] = applyVMPlacementGroupMembershipUpdate(ctx, ct, updates[index])
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}

func applyVMPlacementGroupMembershipUpdate(
	ctx context.Context,
	ct *cloudtower.Client,
	update vmPlacementGroupMembershipUpdate,
) error {
	connect := make([]VmWhereUniqueInput, 0, len(update.vmIDs))
	for _, vmID := range update.vmIDs {
		connect = append(connect, VmWhereUniqueInput{"id": vmID})
	}
	var mutation struct {
		UpdateVMPlacementGroup struct {
			ID graphql.String
		} `graphql:"updateVmPlacementGroup(data: $data, where: $where)"`
	}
	_, err := utils.RetryWithExponentialBackoff(ctx, func() (interface{}, error) {
		return nil, ct.GraphqlApi.Mutate(ctx, &mutation, map[string]interface{}{
			"data": VmPlacementGroupUpdateInput{
				Vms: &VmPlacementGroupVMsUpdateInput{Connect: connect},
			},
			"where": VmPlacementGroupWhereUniqueInput{"id": update.groupID},
		}, graphql.OperationName("updateVmPlacementGroup"))
	}, utils.RetryWithExponentialBackoffOptions{})
	if err != nil {
		return fmt.Errorf("update VM placement group %q membership: %w", update.groupID, err)
	}
	if _, err := ct.WaitTaskForResource(ctx, update.groupID, "updateVmPlacementGroup"); err != nil {
		return fmt.Errorf("wait for VM placement group %q update: %w", update.groupID, err)
	}
	return nil
}

func planVMPlacementGroupMembershipUpdates(
	vmID string,
	vmClusterID string,
	currentGroupIDs []string,
	desiredGroupIDs []string,
	groups []*models.VMPlacementGroup,
) ([]vmPlacementGroupMembershipUpdate, error) {
	changedGroupIDs := symmetricDifference(currentGroupIDs, desiredGroupIDs)
	desired := stringSliceSet(desiredGroupIDs)
	groupByID := make(map[string]*models.VMPlacementGroup, len(groups))
	for _, group := range groups {
		if group != nil && group.ID != nil {
			groupByID[*group.ID] = group
		}
	}

	missing := make([]string, 0)
	for _, groupID := range changedGroupIDs {
		if groupByID[groupID] == nil {
			missing = append(missing, groupID)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("VM placement groups not found: %s", strings.Join(missing, ", "))
	}

	updates := make([]vmPlacementGroupMembershipUpdate, 0, len(changedGroupIDs))
	for _, groupID := range changedGroupIDs {
		group := groupByID[groupID]
		if desired[groupID] {
			if group.Cluster == nil || group.Cluster.ID == nil || *group.Cluster.ID != vmClusterID {
				return nil, fmt.Errorf("VM placement group %q is not in VM cluster %q", groupID, vmClusterID)
			}
		}

		members := make(map[string]bool, len(group.Vms)+1)
		for _, member := range group.Vms {
			if member != nil && member.ID != nil {
				members[*member.ID] = true
			}
		}
		if desired[groupID] {
			members[vmID] = true
		} else {
			delete(members, vmID)
		}

		vmIDs := make([]string, 0, len(members))
		for memberID := range members {
			vmIDs = append(vmIDs, memberID)
		}
		sort.Strings(vmIDs)
		updates = append(updates, vmPlacementGroupMembershipUpdate{
			groupID: groupID,
			vmIDs:   vmIDs,
		})
	}

	return updates, nil
}

func symmetricDifference(left, right []string) []string {
	leftSet := stringSliceSet(left)
	rightSet := stringSliceSet(right)
	result := make([]string, 0)
	for value := range leftSet {
		if !rightSet[value] {
			result = append(result, value)
		}
	}
	for value := range rightSet {
		if !leftSet[value] {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func stringSliceSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}
