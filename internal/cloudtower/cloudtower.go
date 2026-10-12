package cloudtower

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/utils"
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

const (
	userAgent = "terraform-provider-cloudtower"
	// restBasePath is the base path of the CloudTower swagger API.
	restBasePath = "/v2/api"
	// graphqlPath is the path of the CloudTower GraphQL endpoint.
	graphqlPath = "/api"
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

// Client wraps the CloudTower API clients used by the provider.
type Client struct {
	OrgId      string
	Api        *apiclient.Cloudtower
	GraphqlApi *graphql.Client
}

// NewClient creates a CloudTower API client for a server with a single
// management endpoint. The server address accepts an optional scheme, e.g.
// "tower.example.com" or "https://tower.example.com". insecure skips TLS
// certificate verification (ai generated).
func NewClient(ctx context.Context, server string, username string, passwd string, source models.UserSource, insecure bool) (*Client, error) {
	return newClient(ctx, []string{server}, username, passwd, source, insecure)
}

// NewHAClient creates a CloudTower API client for an HA CloudTower (4.9.0+,
// multiple management IPs in an active-passive deployment). servers lists the
// management endpoints of the same HA cluster, e.g.
// ["https://tower-a.example.com", "https://tower-b.example.com"]. The client
// probes the endpoints to find the active node and fails over automatically
// when the active node switches. insecure skips TLS certificate verification
// (ai generated).
func NewHAClient(ctx context.Context, servers []string, username string, passwd string, source models.UserSource, insecure bool) (*Client, error) {
	return newClient(ctx, servers, username, passwd, source, insecure)
}

// httpTransport builds the base transport for the CloudTower API clients,
// cloning http.DefaultTransport so per-client TLS settings do not leak into
// the global default. insecure skips TLS certificate verification, for
// endpoints whose self-signed certificates cannot be verified (e.g. bare IP
// addresses). ai generated
func httpTransport(insecure bool) http.RoundTripper {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return transport
}

func newClient(ctx context.Context, servers []string, username string, passwd string, source models.UserSource, insecure bool) (*Client, error) {
	endpoints, err := parseEndpoints(servers)
	if err != nil {
		return nil, err
	}

	var api *apiclient.Cloudtower
	if len(endpoints) == 1 {
		api = newEndpointAPIClient(endpoints[0], insecure)
	} else {
		// an HA CloudTower is accessed through the SDK's active-passive
		// client, which probes the endpoints for the active node and fails
		// over automatically
		apClient, err := apiclient.NewActivePassiveClient(ctx, apiclient.ActivePassiveClientConfig{
			Endpoints:  toActivePassiveEndpoints(endpoints),
			HTTPClient: &http.Client{Transport: SetUserAgent(httpTransport(insecure), userAgent)},
		})
		if err != nil {
			return nil, err
		}
		api = apClient.Cloudtower
	}

	var graphqlClient *graphql.Client
	if len(endpoints) == 1 {
		graphqlClient = graphql.NewClient(endpoints[0].graphqlURL(), &http.Client{Transport: SetUserAgent(httpTransport(insecure), userAgent)})
	} else {
		graphqlClient = newHAGraphqlClient(endpoints, userAgent, insecure)
	}

	loginParams := user.NewLoginParamsWithContext(ctx)
	loginParams.RequestBody = &models.LoginInput{
		Username: StrPtr(username),
		Password: StrPtr(passwd),
		Source:   &source,
	}
	if source == models.UserSourceLDAP {
		authConfigId, err := getLdapConfig(ctx, graphqlClient)
		if err != nil {
			return nil, err
		}
		if authConfigId == nil {
			return nil, errors.New("LDAP config not found")
		}
		loginParams.RequestBody.AuthConfigID = authConfigId
		loginParams.RequestBody.Source = models.UserSourceAUTHN.Pointer()
	}
	loginResp, err := api.User.Login(loginParams)
	if err != nil {
		return nil, err
	}
	api.SetDefaultAuthentication(httptransport.BearerToken(*loginResp.Payload.Data.Token))

	gop := organization.NewGetOrganizationsParamsWithContext(ctx)
	orgs, err := api.Organization.GetOrganizations(gop)
	if err != nil {
		return nil, err
	}

	return &Client{
		OrgId: *orgs.Payload[0].ID,
		Api:   api,
		GraphqlApi: graphqlClient.WithRequestModifier(func(r *http.Request) {
			r.Header.Set("Authorization", *loginResp.Payload.Data.Token)
		}),
	}, nil
}

func toActivePassiveEndpoints(endpoints []endpoint) []apiclient.ActivePassiveEndpointConfig {
	result := make([]apiclient.ActivePassiveEndpointConfig, 0, len(endpoints))
	for _, ep := range endpoints {
		result = append(result, apiclient.ActivePassiveEndpointConfig{
			Host:     ep.host,
			Schemes:  []string{ep.scheme},
			BasePath: restBasePath,
		})
	}
	return result
}

// newEndpointAPIClient creates an SDK client for a single management
// endpoint. It backs the plain single endpoint setup and provides the SDK's
// ProbeActivePassive for the endpoints of an HA CloudTower.
func newEndpointAPIClient(ep endpoint, insecure bool) *apiclient.Cloudtower {
	transport := httptransport.New(ep.host, restBasePath, []string{ep.scheme})
	transport.Transport = SetUserAgent(httpTransport(insecure), userAgent)
	return apiclient.New(transport, strfmt.Default)
}

func (c *Client) WaitTasksFinish(ctx context.Context, taskIds []string) (*task.GetTasksOK, error) {
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
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			tasksResp, err := utils.RetryWithExponentialBackoff(ctx, func() (*task.GetTasksOK, error) {
				return c.Api.Task.GetTasks(tasksParams)
			}, utils.RetryWithExponentialBackoffOptions{})
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
				continue
			}
			return tasksResp, nil
		}
	}
}

func (c *Client) WaitTaskForResource(ctx context.Context, id string, task_type string) (*task.GetTasksOK, error) {
	tasksParams := task.NewGetTasksParams()
	var first int32 = 1
	tasksParams.RequestBody = &models.GetTasksRequestBody{
		Where: &models.TaskWhereInput{
			ResourceID:       &id,
			ResourceMutation: &task_type,
		},
		OrderBy: models.TaskOrderByInputLocalCreatedAtDESC.Pointer(),
		First:   &first,
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			tasksResp, err := utils.RetryWithExponentialBackoff(ctx, func() (*task.GetTasksOK, error) {
				return c.Api.Task.GetTasks(tasksParams)
			}, utils.RetryWithExponentialBackoffOptions{})
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
				continue
			}
			return tasksResp, nil
		}
	}
}

func getLdapConfig(ctx context.Context, graphqlClient *graphql.Client) (*string, error) {
	var authnStrategies struct {
		AuthnStrategies []struct {
			Id   graphql.String
			Type graphql.String
		} `graphql:"authnStrategies"`
	}
	_, err := utils.RetryWithExponentialBackoff(ctx, func() (interface{}, error) {
		return nil, graphqlClient.Query(ctx, &authnStrategies, map[string]interface{}{})
	}, utils.RetryWithExponentialBackoffOptions{})
	if err != nil {
		return nil, err
	}

	for _, v := range authnStrategies.AuthnStrategies {
		if v.Type == "LDAP" {
			id := string(v.Id)
			return &id, nil
		}
	}
	return nil, nil
}
