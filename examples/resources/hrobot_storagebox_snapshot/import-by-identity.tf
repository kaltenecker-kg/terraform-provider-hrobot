# Import by identity (Terraform 1.12+ / OpenTofu 1.12+).
import {
  to = hrobot_storagebox_snapshot.manual
  identity = {
    storagebox_id = 12345
    name          = "2026-07-12T20-00-00"
  }
}
