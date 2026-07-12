# Security Policy — `terraform-provider-hrobot`

## Verifying release signatures

Releases are signed with an OpenPGP key certified by the **Kaltenecker KG (KKG) root**.

**Current signing key**

```text
terraform-provider-hrobot@kaltenecker.cloud
Fingerprint: 08BB 6872 1570 CDD6 FFA9  D1A5 FB46 499A B01B F96E
```

Fetch it over WKD, along with the KKG root that certifies it:

```bash
gpg --locate-keys oss@kaltenecker.cloud terraform-provider-hrobot@kaltenecker.cloud
```

Then verify a release:

```bash
gpg --verify terraform-provider-hrobot_<version>_SHA256SUMS.sig \
             terraform-provider-hrobot_<version>_SHA256SUMS
```

A good signature reports the signing subkey and primary fingerprint `…FB46499AB01BF96E`.

## Which key is real

The Terraform/OpenTofu registries are append-only. **They may list more than one key for this
namespace** (e.g. after a key rotation). `terraform init` cannot tell which is current.

The authoritative list is **<https://kaltenecker.cloud/KEYS>** (signed by the KKG root). Only keys
marked `CURRENT` there are valid; anything marked `RETIRED` must not be trusted. Confirm the
`KEYS` signature:

```bash
curl -sO https://kaltenecker.cloud/KEYS
curl -sO https://kaltenecker.cloud/KEYS.sig
gpg --verify KEYS.sig KEYS      # signed by the KKG root, oss@kaltenecker.cloud
```

## Reporting a vulnerability

Email `oss@kaltenecker.cloud`.
