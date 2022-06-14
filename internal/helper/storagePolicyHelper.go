package helper

import (
	"fmt"

	apiclient "github.com/smartxworks/cloudtower-go-sdk/v2/client"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/elf_storage_policy"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

func GetElfStoragePolicyByLocalId(client *apiclient.Cloudtower, localId string) (string, error) {
	params := elf_storage_policy.NewGetElfStoragePoliciesParams()
	params.RequestBody = &models.GetElfStoragePoliciesRequestBody{
		Where: &models.ElfStoragePolicyWhereInput{
			LocalIDEndsWith: &localId,
		},
	}
	res, err := client.ElfStoragePolicy.GetElfStoragePolicies(params)
	if err != nil {
		return "", err
	}
	if len(res.Payload) == 0 {
		return "", fmt.Errorf("no storage policy found for local id: %s", localId)
	} else {
		replicaNum := *res.Payload[0].ReplicaNum
		var provision string
		isThinProvision := *res.Payload[0].ThinProvision
		if isThinProvision == true {
			provision = "THIN"
		} else {
			provision = "THICK"
		}
		return fmt.Sprintf("REPLICA_%d_%s_PROVISION", replicaNum, provision), nil
	}
}
