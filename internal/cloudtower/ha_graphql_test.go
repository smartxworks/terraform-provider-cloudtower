package cloudtower

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	apiclient "github.com/smartxworks/cloudtower-go-sdk/v2/client"
)

// fakeHACluster simulates an HA CloudTower: the active node serves the API
// while passive nodes answer every request with a 307 redirect to the active
// node. Discovery is exercised end to end through the SDK's
// ActivePassiveClient, which owns the active endpoint cache; the GraphQL
// doer under test only reads that cache.
type fakeHACluster struct {
	mu      sync.Mutex
	active  map[int]bool
	servers []*httptest.Server

	healthzHits []int
	graphqlHits []int
	restHits    []int
}

func newFakeHACluster(t *testing.T, nodes int) *fakeHACluster {
	t.Helper()
	c := &fakeHACluster{
		active:      make(map[int]bool, nodes),
		servers:     make([]*httptest.Server, nodes),
		healthzHits: make([]int, nodes),
		graphqlHits: make([]int, nodes),
		restHits:    make([]int, nodes),
	}
	for i := 0; i < nodes; i++ {
		i := i
		c.servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// snapshot the state under the lock, then respond without
			// holding it so slow responses do not block the test's checks
			c.mu.Lock()
			isProbe := r.URL.Path == "/api/healthz"
			isREST := strings.HasPrefix(r.URL.Path, restBasePath+"/")
			switch {
			case isProbe:
				c.healthzHits[i]++
			case isREST:
				c.restHits[i]++
			case r.URL.Path == graphqlPath:
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
			c.mu.Unlock()

			switch {
			case isProbe:
				if active {
					w.WriteHeader(http.StatusOK)
				} else if redirectURL != "" {
					http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
				} else {
					w.WriteHeader(http.StatusServiceUnavailable)
				}
			case isREST:
				w.Header().Set("Content-Type", "application/json")
				if active {
					w.Write([]byte(`[]`))
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

// apClient builds the SDK active-passive client over the cluster's nodes;
// construction already runs the SDK's discovery.
func (c *fakeHACluster) apClient(t *testing.T) *apiclient.ActivePassiveClient {
	t.Helper()
	eps := make([]apiclient.ActivePassiveEndpointConfig, 0, len(c.servers))
	for _, s := range c.servers {
		u, err := url.Parse(s.URL)
		if err != nil {
			t.Fatalf("url.Parse(%q) returned error: %v", s.URL, err)
		}
		eps = append(eps, apiclient.ActivePassiveEndpointConfig{
			Host:     u.Host,
			Schemes:  []string{u.Scheme},
			BasePath: restBasePath,
		})
	}
	client, err := apiclient.NewActivePassiveClient(context.Background(), apiclient.ActivePassiveClientConfig{
		Endpoints: eps,
	})
	if err != nil {
		t.Fatalf("NewActivePassiveClient returned error: %v", err)
	}
	return client
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
	payload, err := io.ReadAll(resp.Body)
	return string(payload), err
}

func TestHAGraphqlClientRoutesToActive(t *testing.T) {
	cluster := newFakeHACluster(t, 2)
	cluster.setActive(1)
	doer := newHAGraphqlDoer(cluster.apClient(t), userAgent, false)

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

	// the SDK's discovery cache is shared: once the first request has
	// discovered the active node, further requests never probe again
	if _, err := doAndRead(t, doer, `{"q":2}`); err != nil {
		t.Fatalf("second Do() returned error: %v", err)
	}
	if got := cluster.totalProbes(); got != len(cluster.servers) {
		t.Fatalf("got %d probe requests, want %d from a single lazy discovery", got, len(cluster.servers))
	}
}

func TestHAGraphqlClientFailsOverOnRedirect(t *testing.T) {
	cluster := newFakeHACluster(t, 2)
	cluster.setActive(0)
	doer := newHAGraphqlDoer(cluster.apClient(t), userAgent, false)

	// the first request discovers node 0 and fills the shared cache
	payload, err := doAndRead(t, doer, `{"q":0}`)
	if err != nil {
		t.Fatalf("first Do() returned error: %v", err)
	}
	if !strings.Contains(payload, `"node":0`) {
		t.Fatalf("first Do() = %s, want response from node 0", payload)
	}

	// the active node becomes passive: the GraphQL request gets a 307, a
	// cheap REST request through the SDK drives the failover, and the
	// request is retried against the new active node
	cluster.setActive(1)
	payload, err = doAndRead(t, doer, `{"q":1}`)
	if err != nil {
		t.Fatalf("Do() returned error: %v", err)
	}
	if !strings.Contains(payload, `"node":1`) {
		t.Fatalf("Do() = %s, want response from node 1 after failover", payload)
	}
	if !strings.Contains(payload, `{"q":1}`) {
		t.Fatalf("Do() = %s, want the retried request to keep its body", payload)
	}
	if got := cluster.hits(cluster.graphqlHits, 0); got != 2 {
		t.Fatalf("former active node got %d graphql requests, want 2 (the initial request + the 307)", got)
	}
	if got := cluster.hits(cluster.graphqlHits, 1); got != 1 {
		t.Fatalf("new active node got %d graphql requests, want 1", got)
	}
}

func TestHAGraphqlClientNetworkError(t *testing.T) {
	cluster := newFakeHACluster(t, 2)
	cluster.setActive(0)
	doer := newHAGraphqlDoer(cluster.apClient(t), userAgent, false)

	cluster.servers[0].Close()

	// a dead cached node fails the request without retrying it elsewhere: a
	// network error cannot tell whether the mutation was applied
	if _, err := doer.Do(newGraphQLRequest(t, `{"q":1}`)); err == nil {
		t.Fatal("Do() with a dead active node succeeded, want a network error")
	}
	if got := cluster.hits(cluster.graphqlHits, 1); got != 0 {
		t.Fatalf("graphql request was retried on the surviving node %d times, want 0", got)
	}

	// the failed request refreshed the shared cache, so the next request
	// discovers the surviving node once it takes over
	cluster.setActive(1)
	payload, err := doAndRead(t, doer, `{"q":2}`)
	if err != nil {
		t.Fatalf("second Do() returned error: %v", err)
	}
	if !strings.Contains(payload, `"node":1`) {
		t.Fatalf("second Do() = %s, want response from node 1", payload)
	}
}

// a node that probes healthy but keeps redirecting every request can never
// serve GraphQL; the failover attempt surfaces the SDK's exhaustion
func TestHAGraphqlClientRedirectExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, graphqlPath, http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q) returned error: %v", srv.URL, err)
	}
	apClient, err := apiclient.NewActivePassiveClient(context.Background(), apiclient.ActivePassiveClientConfig{
		Endpoints: []apiclient.ActivePassiveEndpointConfig{{
			Host:     u.Host,
			Schemes:  []string{u.Scheme},
			BasePath: restBasePath,
		}},
	})
	if err != nil {
		t.Fatalf("NewActivePassiveClient returned error: %v", err)
	}
	doer := newHAGraphqlDoer(apClient, userAgent, false)

	if _, err := doer.Do(newGraphQLRequest(t, `{"q":1}`)); err == nil {
		t.Fatal("Do() against an always-redirecting node succeeded, want an error")
	}
}

func TestHAGraphqlClientConcurrentRequests(t *testing.T) {
	cluster := newFakeHACluster(t, 2)
	cluster.setActive(1)
	doer := newHAGraphqlDoer(cluster.apClient(t), userAgent, false)

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
}
