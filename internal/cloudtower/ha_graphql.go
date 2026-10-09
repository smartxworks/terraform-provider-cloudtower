package cloudtower

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hasura/go-graphql-client"
	apiclient "github.com/smartxworks/cloudtower-go-sdk/v2/client"
)

// haProbeTimeout bounds every single endpoint probe so a hanging endpoint
// cannot block discovery indefinitely. It matches the SDK's default probe
// timeout (go-openapi DefaultTimeout).
const haProbeTimeout = 30 * time.Second

// endpointProber reports whether an endpoint of an HA CloudTower is the
// active node. It is implemented by the SDK's Cloudtower.ProbeActivePassive,
// which probes the endpoint (GET /api/healthz): 200 marks the active node and
// 307 a passive one.
type endpointProber interface {
	ProbeActivePassive(ctx context.Context) (bool, error)
}

// haGraphqlClient sends GraphQL requests to the active endpoint of an HA
// CloudTower (4.9.0+, multiple management IPs in an active-passive
// deployment), where passive nodes answer API requests with a 307 redirect.
// Endpoint probing is carried by the SDK (ProbeActivePassive); this client
// caches the discovered active endpoint and mirrors the default failover
// strategy of the SDK's ActivePassiveClient (FailoverStrategyDefault): a 307
// response means the cached node became passive, so the cache is invalidated
// and the request is retried once against the newly discovered active node.
// Network errors also invalidate the cache but are returned to the caller
// instead of being retried.
type haGraphqlClient struct {
	endpoints    []endpoint
	probers      []endpointProber
	inner        *http.Client
	probeTimeout time.Duration

	mu           sync.Mutex
	active       string        // base URL of the cached active endpoint, empty when unknown
	discoverWait chan struct{} // closed when an in-flight discovery finishes
	discoverErr  error         // result of the last discovery
}

func newHAGraphqlClient(endpoints []endpoint, userAgent string) *graphql.Client {
	return graphql.NewClient(endpoints[0].graphqlURL(), newHAGraphqlDoer(endpoints, userAgent))
}

func newHAGraphqlDoer(endpoints []endpoint, userAgent string) *haGraphqlClient {
	probers := make([]endpointProber, 0, len(endpoints))
	for _, ep := range endpoints {
		probers = append(probers, newEndpointAPIClient(ep))
	}
	return &haGraphqlClient{
		endpoints: endpoints,
		probers:   probers,
		// redirects are handled by failover instead of being followed
		inner:        &http.Client{Transport: SetUserAgent(http.DefaultTransport, userAgent), CheckRedirect: stopRedirect},
		probeTimeout: haProbeTimeout,
	}
}

func stopRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// Do sends the GraphQL request to the active endpoint.
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
		base, err := c.activeEndpoint(req.Context())
		if err != nil {
			return nil, err
		}
		resp, err := c.send(req, base, body)
		if err != nil {
			// the cached node is gone: invalidate it for the next request, but
			// do not retry — a network error cannot distinguish a request that
			// never reached the server from one that was applied but whose
			// response was lost, and retrying could execute a non-idempotent
			// mutation twice. A canceled request says nothing about the node,
			// so the cache is kept.
			if req.Context().Err() == nil {
				c.clearActiveIf(base)
			}
			return nil, err
		}
		if resp.StatusCode != http.StatusTemporaryRedirect {
			return resp, nil
		}
		// the cached node became passive — a 307 is sent before the request
		// is processed, so retrying is safe: rediscover the active node and
		// retry once
		resp.Body.Close()
		c.clearActiveIf(base)
		if attempt > 0 {
			return nil, fmt.Errorf("%w: graphql request is still redirected after failover", apiclient.ErrActivePassiveRetryExhausted)
		}
	}
}

// activeEndpoint returns the base URL of the active endpoint. When no
// endpoint is cached yet, every endpoint is probed with the SDK's
// ProbeActivePassive; the active node is expected to be unique, mirroring the
// discovery semantics of the SDK's ActivePassiveClient. Concurrent callers
// wait for an in-flight discovery instead of probing again, and each caller
// gives up when its own context is canceled.
func (c *haGraphqlClient) activeEndpoint(ctx context.Context) (string, error) {
	for {
		c.mu.Lock()
		if c.active != "" {
			base := c.active
			c.mu.Unlock()
			return base, nil
		}
		if wait := c.discoverWait; wait != nil {
			c.mu.Unlock()
			select {
			case <-wait:
				c.mu.Lock()
				base, err := c.active, c.discoverErr
				next := c.discoverWait
				c.mu.Unlock()
				if base != "" {
					return base, nil
				}
				if err != nil && next == nil {
					if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
						return "", err
					}
					// the shared discovery was abandoned because the context of
					// its starter ended: probe again with this caller's context
					// instead of inheriting its cancellation
					continue
				}
				// the finished discovery found no usable endpoint and a new
				// one may already be in flight: loop and join its result
				// instead of returning a stale error
				continue
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		wait := make(chan struct{})
		c.discoverWait = wait
		c.mu.Unlock()

		base, err := c.discover(ctx)

		c.mu.Lock()
		if err == nil {
			c.active = base
		}
		c.discoverErr = err
		c.discoverWait = nil
		close(wait)
		c.mu.Unlock()

		if err != nil {
			return "", err
		}
		return base, nil
	}
}

// discover probes every endpoint and returns the unique active one. It must
// be called without holding c.mu.
func (c *haGraphqlClient) discover(ctx context.Context) (string, error) {
	active := make([]string, 0, 1)
	failures := make([]string, 0)
	for i, ep := range c.endpoints {
		probeCtx, cancel := context.WithTimeout(ctx, c.probeTimeout)
		isActive, err := c.probers[i].ProbeActivePassive(probeCtx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				// the caller is gone, stop probing
				return "", ctx.Err()
			}
			failures = append(failures, fmt.Sprintf("%s: %v", ep.baseURL(), err))
			continue
		}
		if isActive {
			active = append(active, ep.baseURL())
		}
	}
	switch len(active) {
	case 1:
		return active[0], nil
	case 0:
		if len(failures) == 0 {
			return "", apiclient.ErrActivePassiveNoActiveHost
		}
		return "", fmt.Errorf("%w: %s", apiclient.ErrActivePassiveNoActiveHost, strings.Join(failures, "; "))
	default:
		return "", fmt.Errorf("%w: %s", apiclient.ErrActivePassiveMultipleActives, strings.Join(active, ", "))
	}
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

func (c *haGraphqlClient) clearActiveIf(base string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active == base {
		c.active = ""
	}
}
