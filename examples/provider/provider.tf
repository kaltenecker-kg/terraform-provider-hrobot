terraform {
  required_providers {
    hrobot = {
      source = "registry.terraform.io/kaltenecker-kg/hrobot"
    }
  }
}

# username, password, and base_url fall back to the HROBOT_USERNAME,
# HROBOT_PASSWORD, and HROBOT_BASE_URL environment variables when omitted.
provider "hrobot" {}
