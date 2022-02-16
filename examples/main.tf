terraform {
  required_providers {
    cloudtower = {
      version = "~> 0.1.0"
      source  = "registry.terraform.io/smartx/cloudtower"
    }
  }
}

provider "cloudtower" {
  username          = "yanzhen"
  user_source       = "LDAP"
  cloudtower_server = "terraform.dev-cloudtower.smartx.com"
}

data "cloudtower_datacenter" "all" {
  name_contains = "new"
}

resource "cloudtower_cluster" "df_1761" {
  ip            = "192.168.17.39"
  username      = "root"
  password      = "tower2022"
  datacenter_id = data.cloudtower_datacenter.all.datacenters[0].id
}

data "cloudtower_cluster" "all" {
  name_contains = "-"
}

output "test" {
  value = data.cloudtower_cluster.all.clusters
}
