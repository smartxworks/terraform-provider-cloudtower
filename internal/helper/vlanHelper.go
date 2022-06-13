package helper

import (
	"fmt"

	apiclient "github.com/smartxworks/cloudtower-go-sdk/v2/client"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vlan"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

type VlanHelper struct {
	client *apiclient.Cloudtower
}

func NewVlanHelper(client *apiclient.Cloudtower) *VlanHelper {
	return &VlanHelper{
		client: client,
	}
}

func (helper *VlanHelper) GetVlanFromLocalId(localId string) (*models.Vlan, error) {
	params := vlan.NewGetVlansParams()
	params.RequestBody = &models.GetVlansRequestBody{
		Where: &models.VlanWhereInput{
			LocalID: &localId,
		},
	}
	res, err := helper.client.Vlan.GetVlans(params)
	if err != nil {
		return nil, err
	}
	if len(res.Payload) == 0 {
		return nil, fmt.Errorf("vlan %s not found", localId)
	}
	return res.Payload[0], nil
}
