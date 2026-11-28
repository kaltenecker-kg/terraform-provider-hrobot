# Import by identity (Terraform 1.12+ / OpenTofu 1.12+).
import {
  to = hrobot_ssh_key.example
  identity = {
    fingerprint = "56:29:99:a4:5d:ed:ac:d7:6d:ed:ac:d7:6d:ed:ac:d7"
  }
}
