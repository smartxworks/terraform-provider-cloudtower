# Configure the provider for a CloudTower with a single management endpoint.
# The server address accepts an optional scheme and defaults to http.
provider "cloudtower" {
  cloudtower_server = "https://tower.example.com"
  username          = "admin"
  password          = "password"
  user_source       = "LOCAL"
}

# For an HA CloudTower (4.9.0+, multiple management IPs in an active-passive
# deployment), configure every management endpoint of the cluster instead of
# cloudtower_server. The provider probes the endpoints to find the active node
# and fails over automatically when the active node switches.
#
# provider "cloudtower" {
#   cloudtower_servers = ["https://tower-a.example.com", "https://tower-b.example.com"]
#   username           = "admin"
#   password           = "password"
#   user_source        = "LOCAL"
#
#   # Endpoints addressed by bare IP typically carry self-signed certificates
#   # that cannot be verified; skip_tls_verify disables TLS certificate
#   # verification (also via CLOUDTOWER_SKIP_TLS_VERIFY). Use with caution.
#   skip_tls_verify = true
# }
