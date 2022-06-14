package helper

import (
	"fmt"

	apiclient "github.com/smartxworks/cloudtower-go-sdk/v2/client"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/elf_image"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/svt_image"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

func GetElfImageFromLocalId(client *apiclient.Cloudtower, localId string) (*models.ElfImage, error) {
	params := elf_image.NewGetElfImagesParams()
	params.RequestBody = &models.GetElfImagesRequestBody{
		Where: &models.ElfImageWhereInput{
			LocalID: &localId,
		},
	}
	res, err := client.ElfImage.GetElfImages(params)
	if err != nil {
		return nil, err
	}
	if len(res.Payload) == 0 {
		return nil, fmt.Errorf("elf image %s not found", localId)
	}
	return res.Payload[0], nil
}

func GetSvtIMageFromLocalId(client *apiclient.Cloudtower, localId string) (*models.SvtImage, error) {
	params := svt_image.NewGetSvtImagesParams()
	params.RequestBody = &models.GetSvtImagesRequestBody{
		Where: &models.SvtImageWhereInput{
			LocalID: &localId,
		},
	}
	res, err := client.SvtImage.GetSvtImages(params)
	if err != nil {
		return nil, err
	}
	if len(res.Payload) == 0 {
		return nil, fmt.Errorf("svt image %s not found", localId)
	}
	return res.Payload[0], nil
}
