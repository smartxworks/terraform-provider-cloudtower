package cloudtower

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	apiclient "github.com/smartxworks/cloudtower-go-sdk/v2/client"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/organization"
)

// haGraphqlClient sends GraphQL requests to the active endpoint of an HA
// CloudTower (4.9.0+, multiple management IPs in an active-passive
// deployment). It owns no HA state: the active endpoint is discovered,
// cached and failed over by the SDK's ActivePassiveClient alone, which this
// client reads through CurrentActiveEndpoint. When the shared cache is empty
// or stale (a GraphQL request answered with a 307), a cheap REST request
// through the SDK drives its discovery/failover and refreshes the cache.
// ai generated
type haGraphqlClient struct {
	apClient *apiclient.ActivePassiveClient
	inner    *http.Client
}

func newHAGraphqlDoer(apClient *apiclient.ActivePassiveClient, userAgent string, insecure bool) *haGraphqlClient {
	return &haGraphqlClient{
		apClient: apClient,
		// redirects are failover signals, handled by the SDK instead of being
		// followed. ai generated
		inner: &http.Client{Transport: SetUserAgent(httpTransport(insecure), userAgent), CheckRedirect: stopRedirect},
	}
}

func stopRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// Do sends the GraphQL request to the active endpoint. ai generated
func (c *haGraphqlClient) Do(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
	}

	for attempt := 0; ; attempt++ {
		base := c.apClient.CurrentActiveEndpoint()
		if base == "" {
			if err := c.driveDiscovery(req.Context()); err != nil {
				return nil, err
			}
			base = c.apClient.CurrentActiveEndpoint()
			if base == "" {
				return nil, apiclient.ErrActivePassiveNoActiveHost
			}
		}
		resp, err := c.send(req, base, body)
		if err != nil {
			// network errors are not retried: a network error cannot
			// distinguish a request that never reached the server from one
			// that was applied but whose response was lost, and retrying
			// could execute a non-idempotent mutation twice. Best-effort
			// refresh the shared cache so the next request does not hit the
			// dead node again. ai generated
			if req.Context().Err() == nil {
				c.driveDiscovery(req.Context())
			}
			return nil, err
		}
		if resp.StatusCode != http.StatusTemporaryRedirect {
			return resp, nil
		}
		// a 307 is sent before the request is processed, so retrying is safe.
		// The SDK cache still points at the now-passive node; a REST request
		// through the SDK observes the same 307 and drives its failover.
		// ai generated
		resp.Body.Close()
		if attempt > 0 {
			return nil, fmt.Errorf("%w: graphql request is still redirected after failover", apiclient.ErrActivePassiveRetryExhausted)
		}
		if err := c.driveDiscovery(req.Context()); err != nil {
			return nil, err
		}
	}
}

// driveDiscovery refreshes the SDK's active endpoint cache with a cheap REST
// request: the SDK discovers when its cache is empty and fails over when the
// cached node answers with a 307. ai generated
func (c *haGraphqlClient) driveDiscovery(ctx context.Context) error {
	_, err := c.apClient.Organization.GetOrganizations(organization.NewGetOrganizationsParamsWithContext(ctx))
	return err
}

func (c *haGraphqlClient) send(req *http.Request, base string, body []byte) (*http.Response, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	out := req.Clone(req.Context())
	target := *req.URL
	target.Scheme = parsed.Scheme
	target.Host = parsed.Host
	out.URL = &target
	if body != nil {
		out.Body = io.NopCloser(bytes.NewReader(body))
		out.ContentLength = int64(len(body))
		out.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
	}
	return c.inner.Do(out)
}
