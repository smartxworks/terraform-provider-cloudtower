package cloudtower

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/hasura/go-graphql-client"
	apiclient "github.com/smartxworks/cloudtower-go-sdk/v2/client"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/organization"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/task"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/user"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
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
	server     string
	username   string
	passwd     string
	source     models.UserSource
	token      string
	OrgId      string
	Api        *apiclient.Cloudtower
	GraphqlApi *graphql.Client
}

func NewClient(server string, username string, passwd string, source models.UserSource) (*Client, error) {
	transport := httptransport.New(server, "/v2/api", []string{"http"})
	transport.Transport = SetUserAgent(transport.Transport, "terraform-provider-cloudtower")
	api := apiclient.New(transport, strfmt.Default)
	loginParams := user.NewLoginParams()
	loginParams.RequestBody = &models.LoginInput{
		Username: StrPtr(username),
		Password: StrPtr(passwd),
		Source:   &source,
	}
	loginResp, err := api.User.Login(loginParams)
	if err != nil {
		return nil, err
	}
	bearerTokenAuth := httptransport.BearerToken(*loginResp.Payload.Data.Token)
	transport.DefaultAuthentication = bearerTokenAuth
	api = apiclient.New(transport, strfmt.Default)

	gop := organization.NewGetOrganizationsParams()
	orgs, err := api.Organization.GetOrganizations(gop)
	if err != nil {
		return nil, err
	}
	graphqlClient := graphql.NewClient(fmt.Sprintf("http://%s/api", server), nil)

	return &Client{
		server:   server,
		username: username,
		passwd:   passwd,
		source:   source,
		token:    *loginResp.Payload.Data.Token,
		OrgId:    *orgs.Payload[0].ID,
		Api:      api,
		GraphqlApi: graphqlClient.WithRequestModifier(func(r *http.Request) {
			r.Header.Set("Authorization", *loginResp.Payload.Data.Token)
		}),
	}, nil
}

func (c *Client) WaitTasksFinish(taskIds []string) (*task.GetTasksOK, error) {
	if len(taskIds) == 0 {
		return task.NewGetTasksOK(), nil
	}
	tasksParams := task.NewGetTasksParams()
	tasksParams.RequestBody = &models.GetTasksRequestBody{
		Where: &models.TaskWhereInput{
			IDIn: taskIds,
		},
	}
	for {
		tasksResp, err := c.Api.Task.GetTasks(tasksParams)
		if err != nil {
			return nil, err
		}
		allFinished := true
		for _, v := range tasksResp.Payload {
			if *v.Status != models.TaskStatusSUCCESSED && *v.Status != models.TaskStatusFAILED {
				allFinished = false
			}
			if *v.Status == models.TaskStatusFAILED {
				return nil, errors.New(*v.ErrorMessage)
			}
		}
		if !allFinished {
			time.Sleep(5 * time.Second)
			continue
		}
		return tasksResp, nil
	}
}

func (c *Client) WaitTaskForResource(id string) (*task.GetTasksOK, error) {
	tasksParams := task.NewGetTasksParams()
	var first int32 = 1
	tasksParams.RequestBody = &models.GetTasksRequestBody{
		Where: &models.TaskWhereInput{
			ResourceID: &id,
		},
		OrderBy: models.TaskOrderByInputLocalCreatedAtDESC.Pointer(),
		First:   &first,
	}
	for {
		tasksResp, err := c.Api.Task.GetTasks(tasksParams)
		if err != nil {
			return nil, err
		}
		allFinished := true
		for _, v := range tasksResp.Payload {
			if *v.Status != models.TaskStatusSUCCESSED && *v.Status != models.TaskStatusFAILED {
				allFinished = false
			}
			if *v.Status == models.TaskStatusFAILED {
				return nil, errors.New(*v.ErrorMessage)
			}
		}
		if !allFinished {
			time.Sleep(5 * time.Second)
			continue
		}
		return tasksResp, nil
	}
}
