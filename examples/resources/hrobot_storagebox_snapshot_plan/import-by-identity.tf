# Import by identity (Terraform 1.12+ / OpenTofu 1.12+).
import {
  to = hrobot_storagebox_snapshot_plan.daily
  identity = {
    storagebox_id = 12345
  }
}
