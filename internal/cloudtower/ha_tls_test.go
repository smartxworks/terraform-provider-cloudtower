package cloudtower

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	apiclient "github.com/smartxworks/cloudtower-go-sdk/v2/client"
)

// an HTTPS endpoint with a self-signed certificate must be rejected by
// default and accepted when TLS verification is skipped, on both the REST
// probe path and the GraphQL path. ai generated
func TestSelfSignedTLSEndpoint(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// a healthy probe (GET /api/healthz -> 200) marks the active node
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q) returned error: %v", srv.URL, err)
	}
	ep, err := parseEndpoint(u.Host, "https")
	if err != nil {
		t.Fatalf("parseEndpoint(%q) returned error: %v", u.Host, err)
	}

	if _, err := newEndpointAPIClient(ep, false).ProbeActivePassive(context.Background()); err == nil {
		t.Fatal("ProbeActivePassive() with TLS verification succeeded against a self-signed certificate, want an error")
	}
	active, err := newEndpointAPIClient(ep, true).ProbeActivePassive(context.Background())
	if err != nil {
		t.Fatalf("ProbeActivePassive() with skipped TLS verification returned error: %v", err)
	}
	if !active {
		t.Fatal("ProbeActivePassive() = false, want the healthy endpoint to report active")
	}

	// the GraphQL path goes through the SDK's active-passive client
	apClient, err := apiclient.NewActivePassiveClient(context.Background(), apiclient.ActivePassiveClientConfig{
		Endpoints: []apiclient.ActivePassiveEndpointConfig{{
			Host:     u.Host,
			Schemes:  []string{"https"},
			BasePath: restBasePath,
		}},
		HTTPClient: &http.Client{Transport: SetUserAgent(httpTransport(true), userAgent)},
	})
	if err != nil {
		t.Fatalf("NewActivePassiveClient returned error: %v", err)
	}
	doer := newHAGraphqlDoer(apClient, userAgent, true)
	resp, err := doer.Do(newGraphQLRequest(t, `{"q":1}`))
	if err != nil {
		t.Fatalf("Do() with skipped TLS verification returned error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Do() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
