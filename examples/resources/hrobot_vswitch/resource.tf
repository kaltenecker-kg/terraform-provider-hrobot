resource "hrobot_vswitch" "example" {
  name = "internal"
  vlan = 4000

  # Server numbers to attach. Omit to leave membership unmanaged; set to [] to detach all.
  server_numbers = [1234567, 2345678]
}
