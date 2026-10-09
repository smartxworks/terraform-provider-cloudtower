package cloudtower

import "testing"

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		server  string
		want    endpoint
		wantErr bool
	}{
		{name: "bare host", server: "tower.example.com", want: endpoint{scheme: "http", host: "tower.example.com"}},
		{name: "bare host and port", server: "tower.example.com:8080", want: endpoint{scheme: "http", host: "tower.example.com:8080"}},
		{name: "http scheme", server: "http://tower.example.com", want: endpoint{scheme: "http", host: "tower.example.com"}},
		{name: "https scheme", server: "https://tower.example.com", want: endpoint{scheme: "https", host: "tower.example.com"}},
		{name: "https scheme with port", server: "https://tower.example.com:8443", want: endpoint{scheme: "https", host: "tower.example.com:8443"}},
		{name: "trailing slash", server: "https://tower.example.com/", want: endpoint{scheme: "https", host: "tower.example.com"}},
		{name: "surrounding whitespace", server: "  tower.example.com  ", want: endpoint{scheme: "http", host: "tower.example.com"}},
		{name: "empty", server: "", wantErr: true},
		{name: "whitespace only", server: "   ", wantErr: true},
		{name: "unsupported scheme", server: "ftp://tower.example.com", wantErr: true},
		{name: "path not allowed", server: "tower.example.com/tower", wantErr: true},
		{name: "query not allowed", server: "tower.example.com?a=b", wantErr: true},
		{name: "user info not allowed", server: "user@tower.example.com", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEndpoint(tt.server)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseEndpoint(%q) = %#v, want error", tt.server, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseEndpoint(%q) returned error: %v", tt.server, err)
			}
			if got != tt.want {
				t.Fatalf("parseEndpoint(%q) = %#v, want %#v", tt.server, got, tt.want)
			}
		})
	}
}

func TestParseEndpoints(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		if _, err := parseEndpoints(nil); err == nil {
			t.Fatal("parseEndpoints(nil) should return an error")
		}
	})
	t.Run("duplicates", func(t *testing.T) {
		if _, err := parseEndpoints([]string{"tower.example.com", "http://tower.example.com"}); err == nil {
			t.Fatal("parseEndpoints with duplicate endpoints should return an error")
		}
	})
	t.Run("multiple endpoints", func(t *testing.T) {
		got, err := parseEndpoints([]string{"https://tower-a.example.com", "tower-b.example.com:8080"})
		if err != nil {
			t.Fatalf("parseEndpoints returned error: %v", err)
		}
		want := []endpoint{
			{scheme: "https", host: "tower-a.example.com"},
			{scheme: "http", host: "tower-b.example.com:8080"},
		}
		if len(got) != len(want) {
			t.Fatalf("parseEndpoints returned %d endpoints, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("parseEndpoints[%d] = %#v, want %#v", i, got[i], want[i])
			}
		}
	})
}

func TestEndpointURLs(t *testing.T) {
	ep := endpoint{scheme: "https", host: "tower.example.com"}
	if got := ep.baseURL(); got != "https://tower.example.com" {
		t.Fatalf("baseURL() = %q", got)
	}
	if got := ep.graphqlURL(); got != "https://tower.example.com/api" {
		t.Fatalf("graphqlURL() = %q", got)
	}
}
