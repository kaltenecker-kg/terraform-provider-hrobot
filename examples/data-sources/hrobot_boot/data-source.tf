# Read the current boot configuration of a server (which option, if any, is
# armed for the next reboot). Activation itself is a one-shot operation and is
# not a managed resource.
data "hrobot_boot" "example" {
  server_number = 1234567
}

output "rescue_active" {
  value = try(data.hrobot_boot.example.rescue.active, false)
}
