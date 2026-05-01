# terraform-provider-hrobot

Terraform / OpenTofu provider for the
[Hetzner Robot API](https://robot.hetzner.com/doc/webservice/en.html)
(dedicated servers). Built on
[terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework)
and the [`hrobot-go`](../hrobot-go) client library.

> Status: scaffold. Read-only data sources only. No managed resources yet.

## Local development

The `go.mod` carries a `replace` directive that points
`github.com/kaltenecker-kg/hrobot-go` at `../hrobot-go`, so changes to the
client library are picked up immediately.

```sh
task tidy
task build
task install      # drops the binary into ~/.terraform.d/plugins/...
```

To use the locally-installed build, configure a dev override (recommended)
in `~/.terraformrc` or `~/.tofurc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/kaltenecker-kg/hrobot" = "/Users/<you>/go/bin"
  }
  direct {}
}
```

## Configuration

```hcl
provider "hrobot" {
  username = var.hrobot_username   # or HROBOT_USERNAME
  password = var.hrobot_password   # or HROBOT_PASSWORD
}
```

## Data sources

- `hrobot_server` — look up a single server by `id` (server number).
- `hrobot_servers` — list all servers on the account.

## Resources

- `hrobot_firewall` — manage the inbound/outbound firewall configuration of a
  server. Rule order is preserved (Hetzner evaluates top-down). The API caps
  inbound rules at **10**; exceeding that returns `FIREWALL_RULE_LIMIT_EXCEEDED`.
  Import with the server number: `terraform import hrobot_firewall.example 1234567`.

See `examples/` for a working configuration.

## Scope

Ordering, auction, and cancellation endpoints are intentionally out of scope.
