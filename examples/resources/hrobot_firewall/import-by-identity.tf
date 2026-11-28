# Import by identity (Terraform 1.12+ / OpenTofu 1.12+).
import {
  to = hrobot_firewall.example
  identity = {
    server_number = 1234567
  }
}
