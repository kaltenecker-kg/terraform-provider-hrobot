# Import by identity (Terraform 1.12+ / OpenTofu 1.12+).
import {
  to = hrobot_storagebox_subaccount.backup
  identity = {
    storagebox_id = 12345
    username      = "u12345-sub1"
  }
}
