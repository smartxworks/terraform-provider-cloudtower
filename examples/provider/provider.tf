# Configure the provider for a CloudTower with a single management endpoint.
# The server address is a bare host[:port]; the scheme attribute selects http
# or https (default http) for every configured server.
provider "cloudtower" {
  cloudtower_server = "tower.example.com"
  scheme            = "https"
  username          = "admin"
  password          = "password"
  user_source       = "LOCAL"
}

# For an HA CloudTower (4.9.0+, multiple management IPs in an active-passive
# deployment), configure every management endpoint of the cluster instead of
# cloudtower_server. The provider probes the endpoints to find the active node
# and fails over automatically when the active node switches.
#
# Endpoints addressed by bare IP typically carry self-signed certificates that
# cannot be verified; skip_tls_verify (env CLOUDTOWER_SKIP_TLS_VERIFY) disables
# TLS certificate verification. Use with caution.
#
# provider "cloudtower" {
#   cloudtower_servers = ["tower-a.example.com", "tower-b.example.com"]
#   scheme             = "https"
#   skip_tls_verify    = true
#   username           = "admin"
#   password           = "password"
#   user_source        = "LOCAL"
# }
