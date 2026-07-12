# Changelog

## Unreleased

FEATURES:

- **data-source/hrobot_ssh_key**, **hrobot_ssh_keys**: Look up an SSH key by fingerprint, or list all keys
- **data-source/hrobot_ip**, **hrobot_ips**: Look up a single IP (traffic-warning and separate-MAC state), or list all
- **data-source/hrobot_subnet**, **hrobot_subnets**: Look up a subnet by network address, or list all
- **data-source/hrobot_failover**, **hrobot_failovers**: Look up a failover IP and its routing target, or list all
- **data-source/hrobot_vswitch**, **hrobot_vswitches**: Look up a vSwitch (with attached servers/subnets), or list all
- **data-source/hrobot_storagebox**, **hrobot_storageboxes**: Look up a Storage Box, or list all
- **data-source/hrobot_storagebox_subaccounts**, **hrobot_storagebox_snapshots**: List a Storage Box's sub-accounts and snapshots
- **data-source/hrobot_rdns**: Look up the reverse DNS (PTR) entry for an IP
- **data-source/hrobot_boot**: Read a server's boot configuration
- **data-source/hrobot_traffic**: Query traffic statistics for an IP over a time range

## 1.1.0 - 2026-07-12

IMPROVEMENTS:

- **provider**: Identify the provider and client library in the API User-Agent via hrobot-go's `WithApplication`,
  reported as `terraform-provider-hrobot/<version> hrobot-go/<version>`

DEPENDENCIES:

- Bump `hrobot-go` from v1.1.0 to v1.2.0

## 1.0.0 - 2026-07-12

FEATURES:

- **provider**: Configure via the `username`, `password`, and `base_url`
  attributes, or the `HROBOT_USERNAME`, `HROBOT_PASSWORD`, and `HROBOT_BASE_URL`
  environment variables
- **data-source/hrobot_server**: Look up a single dedicated server by number
- **data-source/hrobot_servers**: List all dedicated servers on the account
- **resource/hrobot_firewall**: Manage inbound and outbound firewall rules, with
  import by server number

BUILD:

- Consume the published `hrobot-go` v1.0.0 release and drop the local
  `replace => ../hrobot-go` directive
- Add an MPL-2.0 `LICENSE` and a `terraform-registry-manifest.json` declaring
  protocol version 6.0
- Generate registry documentation under `docs/` with `tfplugindocs`
- Publish GPG-signed, cross-platform releases with GoReleaser on `v*` tags
