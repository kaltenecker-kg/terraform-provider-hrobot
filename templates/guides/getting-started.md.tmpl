---
page_title: "Getting started with the hrobot provider"
subcategory: ""
description: |-
  Configure the hrobot provider and manage your first Hetzner Robot resources with OpenTofu or Terraform.
---

# Getting started with the hrobot provider

The `hrobot` provider manages [Hetzner Robot](https://robot.hetzner.com/) dedicated-server
resources — firewall, SSH keys, reverse DNS, vSwitches, failover IPs, and Storage Box
sub-resources — through the [Robot webservice API](https://robot.hetzner.com/doc/webservice/en.html).

## Requirements

- A Hetzner Robot account with the **webservice/app user** enabled
  (Robot → Settings → Webservice and app settings).
- OpenTofu >= 1.6 or Terraform >= 1.0.

## Provider installation

```terraform
terraform {
  required_providers {
    hrobot = {
      source  = "kaltenecker-kg/hrobot"
      version = "~> 1.2"
    }
  }
}
```

## Authentication

Supply the webservice credentials on the provider block or via environment variables.
Prefer environment variables so secrets stay out of your configuration:

```terraform
provider "hrobot" {}
```

```sh
export HROBOT_USERNAME="#ws+your-user"
export HROBOT_PASSWORD="your-password"
```

Alternatively, set them inline (the password is a sensitive value, so source it from a variable):

```terraform
provider "hrobot" {
  username = "#ws+your-user"
  password = var.hrobot_password
}
```

## Your first configuration

Look up a server, then manage its firewall:

```terraform
data "hrobot_server" "web" {
  id = 1234567
}

resource "hrobot_firewall" "web" {
  server_number = data.hrobot_server.web.id
  status        = "active"

  input_rule = [
    { name = "ssh", ip_version = "ipv4", protocol = "tcp", dst_port = "22", action = "accept" },
    { name = "https", ip_version = "ipv4", protocol = "tcp", dst_port = "443", action = "accept" },
    { name = "default-deny", action = "discard" },
  ]
}
```

Apply it:

```sh
tofu init
tofu plan
tofu apply
```

## Importing existing resources

Every resource supports `import`. The identifier is the natural key of the object — for
example the server number for a firewall, a fingerprint for an SSH key, or an IP for a
reverse-DNS entry:

```sh
tofu import hrobot_firewall.web 1234567
```

See each resource's page for its exact import ID format.

## Scope

Server ordering, auctions, and cancellations are intentionally out of scope. One-shot
actions (server reset, Wake-on-LAN, and boot activation) are not modeled as resources;
boot state is readable via the `hrobot_boot` data source.
