# Import by identity (Terraform 1.12+ / OpenTofu 1.12+).
import {
  to = hrobot_vswitch.example
  identity = {
    id = 12345
  }
}
