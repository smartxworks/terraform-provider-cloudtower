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
  cloudtower_server = "yanzhen.dev-cloudtower.smartx.com"
}

data "cloudtower_datacenters" "all" {}

output "org_id" {
  value = data.cloudtower_datacenters.all.datacenters[0].organization.id
}