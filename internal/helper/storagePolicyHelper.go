package helper

import (
	"fmt"

	apiclient "github.com/smartxworks/cloudtower-go-sdk/v2/client"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/elf_storage_policy"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

type StoragePolicyHelper struct {
	client           *apiclient.Cloudtower
	storagePolicyMap *map[string]string
}

func NewStoragePolicyHelper(client *apiclient.Cloudtower) *StoragePolicyHelper {
	return &StoragePolicyHelper{
		storagePolicyMap: nil,
		client:           client,
	}
}

func (helper *StoragePolicyHelper) GetElfStoragePolicyByLocalId(localId string) (string, error) {
	if helper.storagePolicyMap == nil {
		storagePolicyMap := make(map[string]string)
		params := elf_storage_policy.NewGetElfStoragePoliciesParams()
		params.RequestBody = &models.GetElfStoragePoliciesRequestBody{}
		res, err := helper.client.ElfStoragePolicy.GetElfStoragePolicies(params)
		if err != nil {
			return "", err
		}
		for _, policy := range res.Payload {
			//FIXME: perhap the localid's length is not always the same
			realLocalId := (*policy.LocalID)[37:]
			storagePolicyMap[realLocalId] = *policy.Name
		}
		helper.storagePolicyMap = &storagePolicyMap
	}
	if val, ok := (*helper.storagePolicyMap)[localId]; ok {
		return val, nil
	} else {
		return "", fmt.Errorf("storage policy %s not found", localId)
	}
}
