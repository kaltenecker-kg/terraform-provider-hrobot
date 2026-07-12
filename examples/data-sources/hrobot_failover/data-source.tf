data "hrobot_failover" "example" {
  ip = "203.0.113.5"
}

output "failover_routed_to" {
  value = data.hrobot_failover.example.active_server_ip
}
