resource "hrobot_ssh_key" "example" {
  name       = "deploy"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKeyMaterial deploy@example"
}
