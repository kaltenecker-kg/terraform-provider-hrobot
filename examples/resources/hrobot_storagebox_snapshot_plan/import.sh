#!/usr/bin/env bash
# The snapshot plan is a singleton per box, imported by the Storage Box ID.
terraform import hrobot_storagebox_snapshot_plan.daily 12345
