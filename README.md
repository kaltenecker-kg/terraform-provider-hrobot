# terraform-provider-hrobot

Terraform / OpenTofu provider for the
[Hetzner Robot API](https://robot.hetzner.com/doc/webservice/en.html)
(dedicated servers). Built on
[terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework)
and the [`hrobot-go`](https://github.com/kaltenecker-kg/hrobot-go) client
library.

## Configuration

```hcl
provider "hrobot" {
  username = var.hrobot_username   # or HROBOT_USERNAME
  password = var.hrobot_password   # or HROBOT_PASSWORD
  # base_url = "..."               # or HROBOT_BASE_URL; defaults to the library default
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
GitHub release in the layout the Terraform/OpenTofu registry ingests. The
workflow requires two repository secrets:

- `GPG_PRIVATE_KEY` — ASCII-armored signing key whose public half is registered
  with the provider namespace in the registry.
- `PASSPHRASE` — passphrase for that key.

## Scope

Ordering, auction, and cancellation endpoints are intentionally out of scope.
