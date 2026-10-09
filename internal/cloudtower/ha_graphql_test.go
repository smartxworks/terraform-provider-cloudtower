package cloudtower

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	apiclient "github.com/smartxworks/cloudtower-go-sdk/v2/client"
)

// fakeHACluster simulates an HA CloudTower: the active node serves the API
// while passive nodes answer every request with a 307 redirect to the active
// node. Endpoint probing is exercised end to end against /api/healthz through
// the SDK's ProbeActivePassive (200 = active, 307 = passive).
type fakeHACluster struct {
	mu      sync.Mutex
	active  map[int]bool
	servers []*httptest.Server

	// probeDelay delays every probe response, widening the discovery window.
	probeDelay time.Duration
	// probeGate blocks probe responses until it is closed.
	probeGate chan struct{}

	healthzHits []int
	graphqlHits []int
}

func newFakeHACluster(t *testing.T, nodes int) *fakeHACluster {
	t.Helper()
	c := &fakeHACluster{
		active:      make(map[int]bool, nodes),
		servers:     make([]*httptest.Server, nodes),
		healthzHits: make([]int, nodes),
		graphqlHits: make([]int, nodes),
	}
	for i := 0; i < nodes; i++ {
		i := i
		c.servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// snapshot the state under the lock, then respond without
			// holding it so slow responses do not block the test's checks
			c.mu.Lock()
			isProbe := r.URL.Path == "/api/healthz"
			if isProbe {
				c.healthzHits[i]++
			} else if r.URL.Path == graphqlPath {
				c.graphqlHits[i]++
			}
			active := c.active[i]
			redirectURL := ""
			for j, a := range c.active {
				if a {
					redirectURL = c.servers[j].URL + r.URL.Path
					break
				}
			}
			delay, gate := time.Duration(0), c.probeGate
			if isProbe {
				delay = c.probeDelay
			}
			c.mu.Unlock()

			if isProbe && gate != nil {
				select {
				case <-gate:
				case <-r.Context().Done():
					return
				}
			}
			if delay > 0 {
				time.Sleep(delay)
			}

			switch {
			case isProbe:
				if active {
					w.WriteHeader(http.StatusOK)
				} else if redirectURL != "" {
					http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
				} else {
					w.WriteHeader(http.StatusServiceUnavailable)
				}
			case r.URL.Path == graphqlPath:
				if active {
					w.Header().Set("Content-Type", "application/json")
					// echo the request body so tests can verify that
					// retried requests keep their payload
					body, _ := io.ReadAll(r.Body)
					w.Write([]byte(`{"data":{"node":` + strconv.Itoa(i) + `,"echo":"` + string(body) + `"}}`))
				} else if redirectURL != "" {
					http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
				} else {
					w.WriteHeader(http.StatusServiceUnavailable)
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
	}
	t.Cleanup(func() {
		for _, s := range c.servers {
			s.Close()
		}
	})
	return c
}

func (c *fakeHACluster) setActive(idx int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.active = map[int]bool{idx: true}
}

func (c *fakeHACluster) endpoints(t *testing.T) []endpoint {
	t.Helper()
	endpoints := make([]endpoint, 0, len(c.servers))
	for _, s := range c.servers {
		ep, err := parseEndpoint(s.URL)
		if err != nil {
			t.Fatalf("parseEndpoint(%q) returned error: %v", s.URL, err)
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints
}

func (c *fakeHACluster) hits(hits []int, idx int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return hits[idx]
}

func (c *fakeHACluster) totalProbes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	total := 0
	for _, n := range c.healthzHits {
		total += n
	}
	return total
}

func newGraphQLRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "https://cloudtower.invalid"+graphqlPath, strings.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest returned error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "test-token")
	return req
}

func doAndRead(t *testing.T, doer *haGraphqlClient, body string) (string, error) {
	t.Helper()
	resp, err := doer.Do(newGraphQLRequest(t, body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Do() status = %d, want 200", resp.StatusCode)
	}
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body returned error: %v", err)
	}
	return string(payload), nil
}

func TestHAGraphqlClientRoutesToActiveAndCaches(t *testing.T) {
	cluster := newFakeHACluster(t, 2)
	cluster.setActive(1)
	doer := newHAGraphqlDoer(cluster.endpoints(t), userAgent)

	payload, err := doAndRead(t, doer, `{"q":1}`)
	if err != nil {
		t.Fatalf("Do() returned error: %v", err)
	}
	if !strings.Contains(payload, `"node":1`) {
		t.Fatalf("Do() = %s, want response from node 1", payload)
	}
	if !strings.Contains(payload, `{"q":1}`) {
		t.Fatalf("Do() = %s, want request body echoed", payload)
	}
	if got := cluster.hits(cluster.graphqlHits, 0); got != 0 {
		t.Fatalf("passive node got %d graphql requests, want 0", got)
	}

	// the discovered endpoint is cached: a second request must not probe again
	if _, err := doAndRead(t, doer, `{"q":2}`); err != nil {
		t.Fatalf("second Do() returned error: %v", err)
	}
	for i := range cluster.servers {
		if got := cluster.hits(cluster.healthzHits, i); got != 1 {
			t.Fatalf("node %d got %d probe requests, want 1", i, got)
		}
	}
}

func TestHAGraphqlClientConcurrentDiscoverySingleflight(t *testing.T) {
	cluster := newFakeHACluster(t, 2)
	cluster.setActive(1)
	// widen the discovery window so every caller joins the same discovery
	cluster.probeDelay = 100 * time.Millisecond
	doer := newHAGraphqlDoer(cluster.endpoints(t), userAgent)

	const callers = 4
	errs := make([]error, callers)
	payloads := make([]string, callers)
	reqs := make([]*http.Request, callers)
	for i := range reqs {
		reqs[i] = newGraphQLRequest(t, `{"q":1}`)
	}
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := doer.Do(reqs[i])
			if err != nil {
				errs[i] = err
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				errs[i] = err
				return
			}
			payloads[i] = string(body)
		}()
	}
	wg.Wait()

	for i := 0; i < callers; i++ {
		if errs[i] != nil {
			t.Fatalf("caller %d returned error: %v", i, errs[i])
		}
		if !strings.Contains(payloads[i], `"node":1`) {
			t.Fatalf("caller %d = %s, want response from node 1", i, payloads[i])
		}
	}
	// concurrent callers must share a single discovery
	if got := cluster.totalProbes(); got != len(cluster.servers) {
		t.Fatalf("got %d probe requests, want %d", got, len(cluster.servers))
	}
}

func TestHAGraphqlClientWaiterRespectsContext(t *testing.T) {
	cluster := newFakeHACluster(t, 1)
	cluster.setActive(0)
	// probes block until the gate is released
	cluster.probeGate = make(chan struct{})
	doer := newHAGraphqlDoer(cluster.endpoints(t), userAgent)

	type result struct {
		resp *http.Response
		err  error
	}
	leader := make(chan result, 1)
	go func() {
		resp, err := doer.Do(newGraphQLRequest(t, `{"q":1}`))
		leader <- result{resp, err}
	}()

	// wait until the leader is probing inside the discovery
	deadline := time.Now().Add(2 * time.Second)
	for cluster.hits(cluster.healthzHits, 0) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("leader did not start probing")
		}
		time.Sleep(5 * time.Millisecond)
	}

	waiterCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waiter := make(chan result, 1)
	go func() {
		req := newGraphQLRequest(t, `{"q":2}`).WithContext(waiterCtx)
		resp, err := doer.Do(req)
		waiter <- result{resp, err}
	}()
	// give the waiter time to join the in-flight discovery, then cancel it
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case res := <-waiter:
		if !errors.Is(res.err, context.Canceled) {
			t.Fatalf("waiter error = %v, want context.Canceled", res.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not observe its context cancellation")
	}

	// the leader finishes once the probe is released
	close(cluster.probeGate)
	select {
	case res := <-leader:
		if res.err != nil {
			t.Fatalf("leader Do() returned error: %v", res.err)
		}
		res.resp.Body.Close()
	case <-time.After(2 * time.Second):
		t.Fatal("leader did not finish after the probe was released")
	}
}

func TestHAGraphqlClientProbeTimeout(t *testing.T) {
	cluster := newFakeHACluster(t, 1)
	cluster.setActive(0)
	cluster.probeDelay = 500 * time.Millisecond
	doer := newHAGraphqlDoer(cluster.endpoints(t), userAgent)
	doer.probeTimeout = 50 * time.Millisecond

	// the probe exceeds its timeout and discovery reports no active host
	_, err := doer.Do(newGraphQLRequest(t, `{"q":1}`))
	if !errors.Is(err, apiclient.ErrActivePassiveNoActiveHost) {
		t.Fatalf("Do() error = %v, want ErrActivePassiveNoActiveHost", err)
	}
}

func TestHAGraphqlClientKeepsCacheOnCanceledRequest(t *testing.T) {
	cluster := newFakeHACluster(t, 2)
	cluster.setActive(1)
	doer := newHAGraphqlDoer(cluster.endpoints(t), userAgent)

	if _, err := doAndRead(t, doer, `{"q":1}`); err != nil {
		t.Fatalf("Do() returned error: %v", err)
	}
	probes := cluster.totalProbes()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := newGraphQLRequest(t, `{"q":2}`).WithContext(ctx)
	if _, err := doer.Do(req); !errors.Is(err, context.Canceled) {
		t.Fatalf("Do() error = %v, want context.Canceled", err)
	}

	// a canceled request says nothing about the node, the cached endpoint
	// must survive and serve the next request without re-probing
	if _, err := doAndRead(t, doer, `{"q":3}`); err != nil {
		t.Fatalf("Do() after cancellation returned error: %v", err)
	}
	if got := cluster.totalProbes(); got != probes {
		t.Fatalf("got %d probe requests, want %d (cached endpoint should be kept)", got, probes)
	}
}

func TestHAGraphqlClientWaiterIgnoresStarterCancellation(t *testing.T) {
	cluster := newFakeHACluster(t, 1)
	cluster.setActive(0)
	cluster.probeGate = make(chan struct{})
	doer := newHAGraphqlDoer(cluster.endpoints(t), userAgent)

	type result struct {
		resp *http.Response
		err  error
	}
	starterCtx, cancelStarter := context.WithCancel(context.Background())
	defer cancelStarter()
	starter := make(chan result, 1)
	go func() {
		req := newGraphQLRequest(t, `{"q":1}`).WithContext(starterCtx)
		resp, err := doer.Do(req)
		starter <- result{resp, err}
	}()

	// wait until the starter is probing inside the discovery
	deadline := time.Now().Add(2 * time.Second)
	for cluster.hits(cluster.healthzHits, 0) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("starter did not start probing")
		}
		time.Sleep(5 * time.Millisecond)
	}

	// the waiter joins the in-flight discovery, then the starter is canceled
	waiter := make(chan result, 1)
	go func() {
		resp, err := doer.Do(newGraphQLRequest(t, `{"q":2}`))
		waiter <- result{resp, err}
	}()
	time.Sleep(50 * time.Millisecond)
	cancelStarter()

	select {
	case res := <-starter:
		if !errors.Is(res.err, context.Canceled) {
			t.Fatalf("starter error = %v, want context.Canceled", res.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("starter did not observe its context cancellation")
	}

	// the waiter's own context is alive: it must rediscover with it instead
	// of inheriting the starter's cancellation
	close(cluster.probeGate)
	select {
	case res := <-waiter:
		if res.err != nil {
			t.Fatalf("waiter Do() returned error: %v", res.err)
		}
		defer res.resp.Body.Close()
		body, err := io.ReadAll(res.resp.Body)
		if err != nil {
			t.Fatalf("reading response body returned error: %v", err)
		}
		if !strings.Contains(string(body), `"node":0`) {
			t.Fatalf("waiter = %s, want response from node 0", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not finish after the probe was released")
	}
}

func TestHAGraphqlClientFailsOverOnRedirect(t *testing.T) {
	cluster := newFakeHACluster(t, 2)
	cluster.setActive(0)
	doer := newHAGraphqlDoer(cluster.endpoints(t), userAgent)

	if _, err := doAndRead(t, doer, `{"q":1}`); err != nil {
		t.Fatalf("first Do() returned error: %v", err)
	}

	// the active node switches to node 1, node 0 becomes passive
	cluster.setActive(1)
	payload, err := doAndRead(t, doer, `{"q":2}`)
	if err != nil {
		t.Fatalf("Do() after failover returned error: %v", err)
	}
	if !strings.Contains(payload, `"node":1`) || !strings.Contains(payload, `{"q":2}`) {
		t.Fatalf("Do() = %s, want retried request served by node 1", payload)
	}
	// the passive node must only have seen the redirected attempt
	if got := cluster.hits(cluster.graphqlHits, 0); got != 2 {
		t.Fatalf("node 0 got %d graphql requests, want 2 (1 served, 1 redirected)", got)
	}
	// failover rediscovers the active endpoint through the SDK probe
	if got := cluster.hits(cluster.healthzHits, 1); got != 2 {
		t.Fatalf("node 1 got %d probe requests, want 2", got)
	}
}

func TestHAGraphqlClientNetworkError(t *testing.T) {
	cluster := newFakeHACluster(t, 2)
	cluster.setActive(0)
	doer := newHAGraphqlDoer(cluster.endpoints(t), userAgent)

	if _, err := doAndRead(t, doer, `{"q":1}`); err != nil {
		t.Fatalf("first Do() returned error: %v", err)
	}

	// node 0 dies and node 1 takes over
	cluster.servers[0].Close()
	cluster.setActive(1)

	if _, err := doer.Do(newGraphQLRequest(t, `{"q":2}`)); err == nil {
		t.Fatal("Do() against a dead node should return an error")
	}
	// network errors are not retried within Do: the request may have reached
	// the server already and a retry could execute the mutation twice
	if got := cluster.hits(cluster.graphqlHits, 1); got != 0 {
		t.Fatalf("node 1 got %d graphql requests, want 0", got)
	}

	// the invalidated endpoint is rediscovered through the SDK probe
	payload, err := doAndRead(t, doer, `{"q":3}`)
	if err != nil {
		t.Fatalf("Do() after rediscovery returned error: %v", err)
	}
	if !strings.Contains(payload, `"node":1`) {
		t.Fatalf("Do() = %s, want response from node 1", payload)
	}
}

func TestHAGraphqlClientNoActiveHost(t *testing.T) {
	cluster := newFakeHACluster(t, 2)
	doer := newHAGraphqlDoer(cluster.endpoints(t), userAgent)

	_, err := doer.Do(newGraphQLRequest(t, `{"q":1}`))
	if !errors.Is(err, apiclient.ErrActivePassiveNoActiveHost) {
		t.Fatalf("Do() error = %v, want ErrActivePassiveNoActiveHost", err)
	}
}

func TestHAGraphqlClientMultipleActiveHosts(t *testing.T) {
	cluster := newFakeHACluster(t, 2)
	cluster.mu.Lock()
	cluster.active = map[int]bool{0: true, 1: true}
	cluster.mu.Unlock()
	doer := newHAGraphqlDoer(cluster.endpoints(t), userAgent)

	_, err := doer.Do(newGraphQLRequest(t, `{"q":1}`))
	if !errors.Is(err, apiclient.ErrActivePassiveMultipleActives) {
		t.Fatalf("Do() error = %v, want ErrActivePassiveMultipleActives", err)
	}
}

func TestHAGraphqlClientRedirectExhausted(t *testing.T) {
	// a node that probes healthy but keeps redirecting GraphQL requests can
	// never serve them
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, graphqlPath, http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	ep, err := parseEndpoint(srv.URL)
	if err != nil {
		t.Fatalf("parseEndpoint(%q) returned error: %v", srv.URL, err)
	}
	doer := newHAGraphqlDoer([]endpoint{ep}, userAgent)

	_, err = doer.Do(newGraphQLRequest(t, `{"q":1}`))
	if !errors.Is(err, apiclient.ErrActivePassiveRetryExhausted) {
		t.Fatalf("Do() error = %v, want ErrActivePassiveRetryExhausted", err)
	}
}
