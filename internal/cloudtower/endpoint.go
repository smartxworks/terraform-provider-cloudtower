package cloudtower

import (
	"fmt"
	"net/url"
	"strings"
)

// endpoint is a single CloudTower management endpoint.
type endpoint struct {
	scheme string // "http" or "https"
	host   string // "host" or "host:port"
}

// parseEndpoints parses the CloudTower server addresses configured by the
// user. Each address accepts an optional scheme, e.g. "tower.example.com",
// "http://tower.example.com:8080" or "https://tower.example.com"; bare
// addresses default to "http".
func parseEndpoints(servers []string) ([]endpoint, error) {
	if len(servers) == 0 {
		return nil, fmt.Errorf("no CloudTower server configured: set cloudtower_server or cloudtower_servers")
	}
	seen := make(map[string]bool, len(servers))
	endpoints := make([]endpoint, 0, len(servers))
	for _, server := range servers {
		ep, err := parseEndpoint(server)
		if err != nil {
			return nil, err
		}
		if seen[ep.baseURL()] {
			return nil, fmt.Errorf("duplicate CloudTower server endpoint %q", ep.baseURL())
		}
		seen[ep.baseURL()] = true
		endpoints = append(endpoints, ep)
	}
	return endpoints, nil
}

func parseEndpoint(server string) (endpoint, error) {
	trimmed := strings.TrimSpace(server)
	if trimmed == "" {
		return endpoint{}, fmt.Errorf("invalid CloudTower server address %q: must not be empty", server)
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "http://" + trimmed
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return endpoint{}, fmt.Errorf("invalid CloudTower server address %q: %w", server, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return endpoint{}, fmt.Errorf("invalid CloudTower server address %q: scheme must be http or https", server)
	}
	if u.Host == "" {
		return endpoint{}, fmt.Errorf("invalid CloudTower server address %q: missing host", server)
	}
	if (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return endpoint{}, fmt.Errorf("invalid CloudTower server address %q: must be host[:port] with an optional http(s):// scheme", server)
	}
	return endpoint{scheme: u.Scheme, host: u.Host}, nil
}

// baseURL returns the base URL of the endpoint, e.g. "https://tower.example.com".
func (e endpoint) baseURL() string {
	return e.scheme + "://" + e.host
}

// graphqlURL returns the GraphQL endpoint of the management endpoint.
func (e endpoint) graphqlURL() string {
	return e.baseURL() + graphqlPath
}
