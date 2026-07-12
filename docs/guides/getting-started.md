---
page_title: "Getting started with the hrobot provider"
subcategory: ""
description: |-
  Configure the hrobot provider and manage your first Hetzner Robot resources.
---

# Getting started with the hrobot provider

Manages [Hetzner Robot](https://robot.hetzner.com/) dedicated-server resources — firewall,
SSH keys, reverse DNS, vSwitches, failover IPs, and Storage Box sub-resources — via the
[Robot webservice API](https://robot.hetzner.com/doc/webservice/en.html). Requires a Robot
account with the webservice/app user enabled.

## Install

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

## Authenticate

Set credentials via environment variables (preferred) or the provider block:

```sh
export HROBOT_USERNAME="#ws+your-user"
export HROBOT_PASSWORD="your-password"
```

```terraform
provider "hrobot" {
  # Or set username / password here; password is a sensitive value.
}
```

## First configuration

```terraform
data "hrobot_server" "web" {
  id = 1234567
}

resource "hrobot_firewall" "web" {
  server_number = data.hrobot_server.web.id
  status        = "active"

  input_rule = [
    { name = "ssh", ip_version = "ipv4", protocol = "tcp", dst_port = "22", action = "accept" },
    { name = "default-deny", action = "discard" },
  ]
}
```

## Import

Every resource imports by its natural key (see each resource page for the exact ID format):

```sh
tofu import hrobot_firewall.web 1234567
```

## Scope

Ordering, auctions, and cancellations are out of scope. One-shot actions (reset,
Wake-on-LAN, boot activation) are not resources; boot state is readable via the
`hrobot_boot` data source.
