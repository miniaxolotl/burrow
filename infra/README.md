# Burrow infrastructure

OpenTofu manages the production and staging DigitalOcean App Platform apps,
Upstash Redis databases, and Cloudflare DNS. Both roots use separate encrypted
Neon PostgreSQL state databases and native advisory locks.

- `modules/tunnel-environment`: one app, one Redis database, TLS DNS, and website DNS.
- `staging`: `staging.brmbl.dev` and `staging.useburrow.app`.
- `production`: `brmbl.dev`, `useburrow.app`, and `www.useburrow.app`.
- `scripts/infra/apply.sh`: apply immutable images and reconcile wildcard certificates.

Run `bash scripts/infra/check.sh` with the version in `.opentofu-version`.
Provider selections and checksums are committed in each root's lockfile.

See [deployment operations](../documentation/deploy.md) for credentials,
branch promotion, verification, and package releases.
