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
// user. Addresses are bare host[:port] without a URL scheme: the scheme is
// configured once for every server through the provider's scheme attribute,
// so an HA cluster can never mix http and https endpoints. ai generated
func parseEndpoints(servers []string, scheme string) ([]endpoint, error) {
	if len(servers) == 0 {
		return nil, fmt.Errorf("no CloudTower server configured: set cloudtower_server or cloudtower_servers")
	}
	seen := make(map[string]bool, len(servers))
	endpoints := make([]endpoint, 0, len(servers))
	for _, server := range servers {
		ep, err := parseEndpoint(server, scheme)
		if err != nil {
			return nil, err
		}
		if seen[ep.host] {
			return nil, fmt.Errorf("duplicate CloudTower server endpoint %q", ep.host)
		}
		seen[ep.host] = true
		endpoints = append(endpoints, ep)
	}
	return endpoints, nil
}

func parseEndpoint(server string, scheme string) (endpoint, error) {
	trimmed := strings.TrimSpace(server)
	if trimmed == "" {
		return endpoint{}, fmt.Errorf("invalid CloudTower server address %q: must not be empty", server)
	}
	if strings.Contains(trimmed, "://") {
		return endpoint{}, fmt.Errorf("invalid CloudTower server address %q: do not include a URL scheme; configure the scheme once for every server via the scheme attribute", server)
	}
	u, err := url.Parse("http://" + trimmed)
	if err != nil || u.Host != trimmed || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return endpoint{}, fmt.Errorf("invalid CloudTower server address %q: must be host[:port]", server)
	}
	return endpoint{scheme: scheme, host: trimmed}, nil
}

// baseURL returns the base URL of the endpoint, e.g. "https://tower.example.com".
func (e endpoint) baseURL() string {
	return e.scheme + "://" + e.host
}

// graphqlURL returns the GraphQL endpoint of the management endpoint.
func (e endpoint) graphqlURL() string {
	return e.baseURL() + graphqlPath
}
