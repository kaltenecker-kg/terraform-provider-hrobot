# Changelog

## Unreleased

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
