data "hrobot_servers" "all" {}

output "server_names" {
  value = [for s in data.hrobot_servers.all.servers : s.server_name]
}
