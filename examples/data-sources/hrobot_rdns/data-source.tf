data "hrobot_rdns" "example" {
  ip = "203.0.113.1"
}

output "ptr" {
  value = data.hrobot_rdns.example.ptr
}
