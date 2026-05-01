terraform {
  required_providers {
    hrobot = {
      source = "registry.terraform.io/kaltenecker-kg/hrobot"
    }
  }
}

provider "hrobot" {
  # username / password come from HROBOT_USERNAME / HROBOT_PASSWORD
}

data "hrobot_servers" "all" {}

output "server_names" {
  value = [for s in data.hrobot_servers.all.servers : s.server_name]
}

resource "hrobot_firewall" "example" {
  server_number = 1234567
  status        = "active"
  filter_ipv6   = true
  whitelist_hos = true

  input_rule = [
    {
      name       = "ssh"
      ip_version = "ipv4"
      protocol   = "tcp"
      dst_port   = "22"
      action     = "accept"
    },
    {
      name   = "default-deny"
      action = "discard"
    },
  ]
}
