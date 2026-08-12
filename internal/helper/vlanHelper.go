package helper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	apiclient "github.com/smartxworks/cloudtower-go-sdk/v2/client"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vlan"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

const vlanIDMax = 4095

func validateVlanID(s string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("must be a number, got %q", s)
	}
	if v < 0 || v > vlanIDMax {
		return 0, fmt.Errorf("must be between 0 and %d, got %d", vlanIDMax, v)
	}
	return v, nil
}

func ValidateNetworkId(val interface{}, path cty.Path) diag.Diagnostics {
	s, ok := val.(string)
	if !ok || s == "" {
		return diag.Errorf("network_id must be a non-empty string")
	}

	parts := strings.SplitN(s, "-", 2)
	if len(parts) == 1 {
		if _, err := validateVlanID(parts[0]); err != nil {
			return diag.Errorf("network_id %q invalid: %s", s, err)
		}
		return nil
	}

	start, err := validateVlanID(parts[0])
	if err != nil {
		return diag.Errorf("network_id %q invalid start: %s", s, err)
	}
	end, err := validateVlanID(parts[1])
	if err != nil {
		return diag.Errorf("network_id %q invalid end: %s", s, err)
	}
	if start > end {
		return diag.Errorf("network_id %q invalid: start (%d) must not be greater than end (%d)", s, start, end)
	}

	return nil
}

func GetVlanFromLocalId(client *apiclient.Cloudtower, localId string) (*models.Vlan, error) {
	params := vlan.NewGetVlansParams()
	params.RequestBody = &models.GetVlansRequestBody{
		Where: &models.VlanWhereInput{
			LocalID: &localId,
		},
	}
	res, err := client.Vlan.GetVlans(params)
	if err != nil {
		return nil, err
	}
	if len(res.Payload) == 0 {
		return nil, fmt.Errorf("vlan %s not found", localId)
	}
	return res.Payload[0], nil
}
