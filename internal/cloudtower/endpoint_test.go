package cloudtower

import "testing"

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		server  string
		scheme  string
		want    endpoint
		wantErr bool
	}{
		{name: "bare host", server: "tower.example.com", scheme: "http", want: endpoint{scheme: "http", host: "tower.example.com"}},
		{name: "bare host and port", server: "tower.example.com:8080", scheme: "http", want: endpoint{scheme: "http", host: "tower.example.com:8080"}},
		{name: "https scheme from attribute", server: "tower.example.com", scheme: "https", want: endpoint{scheme: "https", host: "tower.example.com"}},
		{name: "surrounding whitespace", server: "  tower.example.com  ", scheme: "http", want: endpoint{scheme: "http", host: "tower.example.com"}},
		{name: "empty", server: "", scheme: "http", wantErr: true},
		{name: "whitespace only", server: "   ", scheme: "http", wantErr: true},
		{name: "http scheme not allowed", server: "http://tower.example.com", scheme: "http", wantErr: true},
		{name: "https scheme not allowed", server: "https://tower.example.com", scheme: "https", wantErr: true},
		{name: "path not allowed", server: "tower.example.com/tower", scheme: "http", wantErr: true},
		{name: "query not allowed", server: "tower.example.com?a=b", scheme: "http", wantErr: true},
		{name: "user info not allowed", server: "user@tower.example.com", scheme: "http", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEndpoint(tt.server, tt.scheme)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseEndpoint(%q, %q) = %#v, want error", tt.server, tt.scheme, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseEndpoint(%q, %q) returned error: %v", tt.server, tt.scheme, err)
			}
			if got != tt.want {
				t.Fatalf("parseEndpoint(%q, %q) = %#v, want %#v", tt.server, tt.scheme, got, tt.want)
			}
		})
	}
}

func TestParseEndpoints(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		if _, err := parseEndpoints(nil, "http"); err == nil {
			t.Fatal("parseEndpoints(nil) should return an error")
		}
	})
	t.Run("duplicates", func(t *testing.T) {
		if _, err := parseEndpoints([]string{"tower.example.com", "tower.example.com"}, "http"); err == nil {
			t.Fatal("parseEndpoints with duplicate endpoints should return an error")
		}
	})
	t.Run("multiple endpoints share the scheme", func(t *testing.T) {
		got, err := parseEndpoints([]string{"tower-a.example.com", "tower-b.example.com:8080"}, "https")
		if err != nil {
			t.Fatalf("parseEndpoints returned error: %v", err)
		}
		want := []endpoint{
			{scheme: "https", host: "tower-a.example.com"},
			{scheme: "https", host: "tower-b.example.com:8080"},
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
