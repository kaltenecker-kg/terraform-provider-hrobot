# All SSH keys stored in the Robot account.
data "hrobot_ssh_keys" "all" {}

output "ssh_key_fingerprints" {
  value = [for k in data.hrobot_ssh_keys.all.ssh_keys : k.fingerprint]
}
