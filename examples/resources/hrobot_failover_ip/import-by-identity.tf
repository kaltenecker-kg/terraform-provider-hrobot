# Import by identity (Terraform 1.12+ / OpenTofu 1.12+).
import {
  to = hrobot_failover_ip.example
  identity = {
    ip = "203.0.113.5"
  }
}
