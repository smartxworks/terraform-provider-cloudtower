package cloudtower

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// the transport built with TLS verification skipped must accept a
// self-signed certificate, while the default one rejects it. ai generated
func TestHTTPTransportSkipsTLSVerify(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if _, err := (&http.Client{Transport: httpTransport(false)}).Get(srv.URL); err == nil {
		t.Fatal("GET with TLS verification succeeded against a self-signed certificate, want an error")
	}
	resp, err := (&http.Client{Transport: httpTransport(true)}).Get(srv.URL)
	if err != nil {
		t.Fatalf("GET with skipped TLS verification returned error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// the option must not break the plain HTTP path. ai generated
func TestNewClientPlainHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/login"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]string{"token": "test-token"},
			})
		case strings.HasSuffix(r.URL.Path, "/get-organizations"):
			json.NewEncoder(w).Encode([]map[string]string{{"id": "test-org"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	for _, insecure := range []bool{false, true} {
		if _, err := NewClient(host, "user", "pass", "LOCAL", "http", insecure); err != nil {
			t.Fatalf("NewClient(insecure=%v) over plain HTTP returned error: %v", insecure, err)
		}
	}
}
