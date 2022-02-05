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

resource "cloudtower_datacenter" "idc" {
  name = "IDC"
}

output "idc_datacenter" {
  value = cloudtower_datacenter.idc
}

resource "cloudtower_cluster" "df_1761" {
  ip = "192.168.17.39"
  username = "root"
  password = "tower2022"
  datacenter_id = cloudtower_datacenter.idc.id
}

output "df_1761_cluster" {
  value = cloudtower_cluster.df_1761
}