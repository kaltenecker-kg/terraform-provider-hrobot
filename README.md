# terraform-provider-hrobot

Terraform / OpenTofu provider for the
[Hetzner Robot API](https://robot.hetzner.com/doc/webservice/en.html)
(dedicated servers). Built on
[terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework)
and the [`hrobot-go`](https://github.com/kaltenecker-kg/hrobot-go) client
library.

## Configuration

```hcl
# username, password, and base_url fall back to the HROBOT_USERNAME,
# HROBOT_PASSWORD, and HROBOT_BASE_URL environment variables when omitted.
provider "hrobot" {}
```

To set the credentials in configuration instead, declare `variable` blocks for
them and assign `username`/`password` on the provider.

## Data sources

- `hrobot_server` / `hrobot_servers` — a single server (by number) or all servers.
- `hrobot_ssh_key` / `hrobot_ssh_keys` — a single SSH key (by fingerprint) or all keys.
- `hrobot_ip` / `hrobot_ips` — a single IP (by address) or all single IPs, with
  traffic-warning and separate-MAC state.
- `hrobot_subnet` / `hrobot_subnets` — a single subnet (by network address) or all subnets.
- `hrobot_failover` / `hrobot_failovers` — a failover IP and its current routing target.
- `hrobot_vswitch` / `hrobot_vswitches` — a vSwitch (with attached servers/subnets) or the summary list.
- `hrobot_storagebox` / `hrobot_storageboxes` — a Storage Box or all Storage Boxes.
- `hrobot_storagebox_subaccounts` / `hrobot_storagebox_snapshots` — sub-accounts and
  snapshots of a Storage Box.
- `hrobot_rdns` — the reverse DNS (PTR) entry for an IP.
- `hrobot_boot` — the current boot configuration of a server (rescue/linux/vnc/windows/plesk/cpanel).
- `hrobot_traffic` — traffic statistics for an IP over a time range.

## Resources

- `hrobot_firewall` — manage the inbound/outbound firewall configuration of a
  server. Rule order is preserved (Hetzner evaluates top-down). The API caps
  inbound rules at **10**; exceeding that returns `FIREWALL_RULE_LIMIT_EXCEEDED`.
  Import with the server number: `terraform import hrobot_firewall.example 1234567`.

Reference documentation for every attribute lives under [`docs/`](docs/), and
runnable configuration lives under [`examples/`](examples/).

## Local development

The provider consumes a published [`hrobot-go`](https://github.com/kaltenecker-kg/hrobot-go)
release via `go.mod`. To build and install it into the local plugin cache:

```sh
task build
task install      # drops the binary into ~/.terraform.d/plugins/...
```

To exercise the locally-installed build, configure a dev override in
`~/.terraformrc` or `~/.tofurc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/kaltenecker-kg/hrobot" = "/Users/<you>/go/bin"
  }
  direct {}
}
```

### Common tasks

```sh
task check              # lint, vet, test, tidy-check, verify, vulncheck
task docs               # regenerate docs/ from the provider schema and examples/
task release-snapshot   # build an unpublished goreleaser snapshot
```

Regenerate `docs/` with `task docs` whenever a schema description or an example
changes, and commit the result.

## Releasing

Pushing a `vX.Y.Z` tag runs the [release workflow](.github/workflows/release.yml),
which builds cross-platform binaries with GoReleaser and publishes a GPG-signed
GitHub release in the layout the Terraform/OpenTofu registry ingests.

Release signing keys and their setup are managed out-of-band and documented
privately.

## Scope

Ordering, auction, and cancellation endpoints are intentionally out of scope.

One-shot actions — server reset, Wake-on-LAN, and boot activation
(rescue/linux/vnc/windows) — do not fit Terraform's declarative model and are not
exposed as resources. Boot state is readable via the `hrobot_boot` data source;
use the `hrobot-go` client directly for the actions themselves.
