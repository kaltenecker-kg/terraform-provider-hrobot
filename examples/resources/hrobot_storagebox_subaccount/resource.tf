# The Storage Box is ordered outside Terraform; reference it by ID.
resource "hrobot_storagebox_subaccount" "backup" {
  storagebox_id  = 12345
  home_directory = "/backups"
  samba          = true
  ssh            = true
}

# The auto-generated password is available (sensitive) for the initial setup.
output "subaccount_password" {
  value     = hrobot_storagebox_subaccount.backup.password
  sensitive = true
}
