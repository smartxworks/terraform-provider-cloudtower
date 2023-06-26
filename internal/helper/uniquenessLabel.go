package helper

import (
	"github.com/smartxworks/cloudtower-go-sdk/v2/client"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/label"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
)

var LabelPrefix = "system.cloudtower/terraform"

func GetUniquenessLabelKey(t string) string {
	return LabelPrefix + "/" + t
}

func CreateUniquenessLabel(client *client.Cloudtower, t string, id string) (*label.CreateLabelOK, error) {
	key := GetUniquenessLabelKey(t)
	clp := label.NewCreateLabelParams()
	clp.RequestBody = []*models.LabelCreationParams{
		{
			Key:   &key,
			Value: &id,
		},
	}
	return client.Label.CreateLabel(clp)
}

func GetUniquenessLabel(client *client.Cloudtower, t string, id string) (*label.GetLabelsOK, error) {
	key := GetUniquenessLabelKey(t)
	glp := label.NewGetLabelsParams()
	glp.RequestBody = &models.GetLabelsRequestBody{
		Where: &models.LabelWhereInput{
			Key:   &key,
			Value: &id,
		},
	}
	return client.Label.GetLabels(glp)
}

func DeleteUniquenessLabel(client *client.Cloudtower, t string, id string) (*label.DeleteLabelOK, error) {
	key := GetUniquenessLabelKey(t)
	dlp := label.NewDeleteLabelParams()
	dlp.RequestBody = &models.LabelDeletionParams{
		Where: &models.LabelWhereInput{
			Key:   &key,
			Value: &id,
		},
	}
	return client.Label.DeleteLabel(dlp)
}
