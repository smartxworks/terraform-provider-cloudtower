package cloudtower

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// a TLS endpoint with a self-signed certificate must be rejected by default
// and accepted when TLS verification is skipped. ai generated
func TestSelfSignedTLSEndpoint(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// a healthy probe (GET /api/healthz -> 200) marks the active node
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ep, err := parseEndpoint(srv.URL)
	if err != nil {
		t.Fatalf("parseEndpoint(%q) returned error: %v", srv.URL, err)
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

	if _, err := newHAGraphqlDoer([]endpoint{ep}, userAgent, false).Do(newGraphQLRequest(t, `{"q":1}`)); err == nil {
		t.Fatal("Do() with TLS verification succeeded against a self-signed certificate, want an error")
	}
	doer := newHAGraphqlDoer([]endpoint{ep}, userAgent, true)
	resp, err := doer.Do(newGraphQLRequest(t, `{"q":1}`))
	if err != nil {
		t.Fatalf("Do() with skipped TLS verification returned error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Do() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
