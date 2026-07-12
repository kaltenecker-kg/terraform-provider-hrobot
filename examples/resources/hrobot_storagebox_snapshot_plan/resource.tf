# Take a snapshot every day at 03:00 and keep the 7 most recent.
resource "hrobot_storagebox_snapshot_plan" "daily" {
  storagebox_id = 12345
  status        = "enabled"
  hour          = 3
  minute        = 0
  max_snapshots = 7
}
