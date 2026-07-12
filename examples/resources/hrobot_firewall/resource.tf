resource "hrobot_firewall" "example" {
  server_number = 1234567
  status        = "active"
  filter_ipv6   = true
  whitelist_hos = true

  # Rules are evaluated top to bottom. Hetzner allows at most 10 input rules.
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
