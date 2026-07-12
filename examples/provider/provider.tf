terraform {
  required_providers {
    hrobot = {
      source = "registry.terraform.io/kaltenecker-kg/hrobot"
    }
  }
}

provider "hrobot" {
  # username and password default to the HROBOT_USERNAME and HROBOT_PASSWORD
  # environment variables when omitted here.
  username = var.hrobot_username
  password = var.hrobot_password
}
