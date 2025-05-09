package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/utils"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

func WaitVmToolsRunning(ctx context.Context, ct *cloudtower.Client, vmId string) (*models.VM, error) {
	if vmId == "" {
		return nil, fmt.Errorf("vmId cannot be empty")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	params := vm.NewGetVmsParams()
	params.RequestBody = &models.GetVmsRequestBody{
		Where: &models.VMWhereInput{
			ID: &vmId,
		},
		First: utils.Pointy[int32](1),
	}

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("vm %s tools status is not running after 10 minutes", vmId)
			}
			return nil, ctx.Err()
		default:
			var res *vm.GetVmsOK
			var err error

			res, err = utils.RetryWithExponentialBackoff(ctx, func() (*vm.GetVmsOK, error) {
				return ct.Api.VM.GetVms(params)
			}, utils.RetryWithExponentialBackoffOptions{})

			if err != nil {
				return nil, fmt.Errorf("failed to get VM status after %d retries: %v", 3, err)
			}

			if res == nil || res.Payload == nil || len(res.Payload) == 0 {
				return nil, fmt.Errorf("no VM found with id: %s", vmId)
			}

			if res.Payload[0].VMToolsStatus != nil && *res.Payload[0].VMToolsStatus == models.VMToolsStatusRUNNING {
				return res.Payload[0], nil
			}

		}
	}
}

func StartVmTemporary(ctx context.Context, ct *cloudtower.Client, vmId string) (func() error, error) {
	getParams := vm.NewGetVmsParams()
	getParams.RequestBody = &models.GetVmsRequestBody{
		Where: &models.VMWhereInput{
			ID: &vmId,
		},
		First: utils.Pointy[int32](1),
	}

	var res *vm.GetVmsOK
	var err error

	res, err = utils.RetryWithExponentialBackoff(ctx, func() (*vm.GetVmsOK, error) {
		return ct.Api.VM.GetVms(getParams)
	}, utils.RetryWithExponentialBackoffOptions{})

	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %v", err)
	}

	if res == nil || res.Payload == nil || len(res.Payload) == 0 {
		return nil, fmt.Errorf("no VM found with id: %s", vmId)
	}
	switch *res.Payload[0].Status {
	case models.VMStatusRUNNING:
		// vm has already been running, do nothing in onDone
		return func() error {
			return nil
		}, nil
	case models.VMStatusSTOPPED:
		// if stopped, start the vm
		startParams := vm.NewStartVMParams()
		startParams.RequestBody = &models.VMStartParams{
			Where: &models.VMWhereInput{
				ID: &vmId,
			},
			Data: &models.VMStartParamsData{
				HostID: res.Payload[0].Host.ID,
			},
		}
		res, err := utils.RetryWithExponentialBackoff(ctx, func() (*vm.StartVMOK, error) {
			return ct.Api.VM.StartVM(startParams)
		}, utils.RetryWithExponentialBackoffOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to start VM: %v", err)
		}
		_, err = ct.WaitTasksFinish(ctx, []string{*res.Payload[0].TaskID})
		if err != nil {
			return nil, fmt.Errorf("failed to wait for VM to start: %v", err)
		}
		// vm has been started temporary, need to power off when done
		return func() error {
			powerOffParams := vm.NewPoweroffVMParams()
			powerOffParams.RequestBody = &models.VMOperateParams{
				Where: &models.VMWhereInput{
					ID: &vmId,
				},
			}
			resp, err := utils.RetryWithExponentialBackoff(ctx, func() (*vm.PoweroffVMOK, error) {
				return ct.Api.VM.PoweroffVM(powerOffParams)
			}, utils.RetryWithExponentialBackoffOptions{})

			if err != nil {
				return fmt.Errorf("failed to power off VM: %v", err)
			}
			_, err = ct.WaitTasksFinish(ctx, []string{*resp.Payload[0].TaskID})
			if err != nil {
				return fmt.Errorf("failed to wait for VM to power off: %v", err)
			}
			return nil
		}, nil
	default:
		// vm is in other status, cannot start temporary
		return nil, fmt.Errorf("VM %s status is %s, cannot start temporary", vmId, *res.Payload[0].Status)
	}
}
