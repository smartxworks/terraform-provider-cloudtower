package cloudtower

import (
	apiclient "github.com/Yuyz0112/cloudtower-go-sdk/client"
	"github.com/Yuyz0112/cloudtower-go-sdk/client/operations"
	"github.com/Yuyz0112/cloudtower-go-sdk/models"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"time"
)

const (
	KiB = 1 << 10
	MiB = 1 << 20
	GiB = 1 << 30
)

func StrPtr(s string) *string {
	return &s
}
func BoolPtr(b bool) *bool {
	return &b
}
func FloatPtr(f float64) *float64 {
	return &f
}

type Client struct {
	server   string
	username string
	passwd   string
	source   models.UserSource
	token    string
	OrgId    string
	Api      *apiclient.AtTowerOperationAPI
}

func NewClient(server string, username string, passwd string, source models.UserSource) (*Client, error) {
	transport := httptransport.New(server, "/v2/api", []string{"http"})
	transport.Transport = SetUserAgent(transport.Transport, "terraform-provider-cloudtower")
	api := apiclient.New(transport, strfmt.Default)
	loginParams := operations.NewLoginParams()
	loginParams.RequestBody = &models.LoginInput{
		Username: StrPtr(username),
		Password: StrPtr(passwd),
		Source:   &source,
	}
	loginResp, err := api.Operations.Login(loginParams)
	if err != nil {
		return nil, err
	}
	bearerTokenAuth := httptransport.BearerToken(*loginResp.Payload.Data.Token)
	transport.DefaultAuthentication = bearerTokenAuth
	api = apiclient.New(transport, strfmt.Default)

	gop := operations.NewGetOrganizationsParams()
	orgs, err := api.Operations.GetOrganizations(gop)
	if err != nil {
		return nil, err
	}

	return &Client{
		server:   server,
		username: username,
		passwd:   passwd,
		source:   source,
		token:    *loginResp.Payload.Data.Token,
		OrgId:    *orgs.Payload[0].ID,
		Api:      api,
	}, nil
}

func (c *Client) WaitTasksFinish(taskIds []string) (*operations.GetTasksOK, error) {
	if len(taskIds) == 0 {
		return operations.NewGetTasksOK(), nil
	}
	tasksParams := operations.NewGetTasksParams()
	tasksParams.RequestBody = &models.GetTasksRequestBody{
		Where: &models.TaskWhereInput{
			IDIn: taskIds,
		},
	}
	for {
		tasksResp, err := c.Api.Operations.GetTasks(tasksParams)
		if err != nil {
			return nil, err
		}
		allFinished := true
		for _, v := range tasksResp.Payload {
			if *v.Status != models.TaskStatusSUCCESSED && *v.Status != models.TaskStatusFAILED {
				allFinished = false
			}
		}
		if !allFinished {
			time.Sleep(5 * time.Second)
			continue
		}
		return tasksResp, nil
	}
}
