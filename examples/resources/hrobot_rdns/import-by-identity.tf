# Import by identity (Terraform 1.12+ / OpenTofu 1.12+).
import {
  to = hrobot_rdns.example
  identity = {
    ip = "203.0.113.1"
  }
}
