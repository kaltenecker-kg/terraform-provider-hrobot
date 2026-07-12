# The failover IP itself is ordered outside Terraform; this resource routes it
# to a server and can move it between servers.
resource "hrobot_failover_ip" "example" {
  ip               = "203.0.113.5"
  active_server_ip = "198.51.100.10"
}
