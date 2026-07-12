# Daily traffic for an IP across a date range.
data "hrobot_traffic" "example" {
  type = "day"
  from = "2026-07-01T00"
  to   = "2026-07-07T23"
  ip   = "203.0.113.10"
}

output "traffic" {
  value = data.hrobot_traffic.example.data
}
