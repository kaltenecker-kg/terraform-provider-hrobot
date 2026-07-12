data "hrobot_ssh_key" "example" {
  fingerprint = "56:29:99:a4:5d:ed:ac:d7:6d:ed:ac:d7:6d:ed:ac:d7"
}

output "ssh_key_name" {
  value = data.hrobot_ssh_key.example.name
}
