data "hrobot_server" "web" {
  id = 1234567
}

output "web_ip" {
  value = data.hrobot_server.web.server_ip
}
