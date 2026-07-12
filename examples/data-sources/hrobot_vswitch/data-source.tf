data "hrobot_vswitch" "example" {
  id = 12345
}

output "vswitch_servers" {
  value = [for s in data.hrobot_vswitch.example.servers : s.server_number]
}
