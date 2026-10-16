# Deploy & release

## Environments

|                  | Production               | Staging                          |
| ---------------- | ------------------------ | -------------------------------- |
| Git branch       | `production`             | `development`                    |
| Website (Vercel) | `https://useburrow.app`  | `https://staging.useburrow.app`  |
| Tunnel server    | `https://brmbl.dev`      | `https://staging.brmbl.dev`      |
| Tunnel URLs      | `https://<id>.brmbl.dev` | `https://<id>.staging.brmbl.dev` |
| DigitalOcean app | `burrow`                 | `burrow-staging`                 |
| Upstash Redis    | `burrow`                 | `burrow-staging`                 |
| OpenTofu root    | `infra/production`       | `infra/staging`                  |

Vercel builds `packages/website` independently. Production uses
`SITE_URL=https://useburrow.app`; previews use
`SITE_URL=https://staging.useburrow.app` and are marked `noindex`.
`www.useburrow.app` redirects to the production apex.

## Pipeline

1. **CI** runs website lint/types/build, Go vet/build/tests, OpenTofu formatting
   and validation, workflow lint, and a Docker build.
2. After successful branch CI, **Deploy** publishes multi-architecture images
   to GHCR and Docker Hub. Each image has an immutable `sha-<commit>` tag and
   source-revision labels. Promotion reuses that commit's existing image.
3. OpenTofu applies the selected environment using the image's **digest**, not
   a mutable tag. It owns the complete App Platform specification, Upstash
   database and Cloudflare website/tunnel DNS.
4. App Platform supplies wildcard certificate verification tokens. Deployment
   reads these tokens and applies their TXT records through OpenTofu. A daily
   maintenance workflow refreshes them for certificate renewals.
5. Deployment waits for HTTPS health, then creates a real WSS/yamux tunnel and
   checks HTTP body, headers, query strings, and API-like paths end to end.
6. After a successful production deployment, **Release** uploads GoReleaser
   archives/checksums to GitHub and publishes all four native npm packages,
   followed by `@miniaxolotl/burrowctl`.

The npm registry serves npm, pnpm, Yarn, and Bun. Native archives cover Linux and
macOS on amd64/arm64. Docker images cover Linux amd64/arm64. There is currently
no Homebrew tap, winget package, or apt repository in this project.

Release-please prepares the version/changelog PR on `development`. Merge the
release PR, let staging pass, then fast-forward `production` to the release-tag
commit. Package publication checks that the tag matches the deployed commit.
The `latest` container tags follow production; `preview` follows staging.

## Infrastructure and secrets

OpenTofu **1.12.6** follows Fern's encrypted PostgreSQL-backend pattern. Neon
project `burrow-infra-state` owns separate state databases, restricted CI roles,
and AES-GCM encryption keys for staging and production. State uses a direct TLS
connection with PostgreSQL advisory locking. Application tunnel records live in
Upstash, separate from infrastructure state.

The random-provider declarations and lock entries are retained for the owner-only
upgrade so deployment's read-only initialization can load existing
`module.tunnels.random_password.admin` state and remove that obsolete resource.
Remove those provider pins in a follow-up only after both staging and production
have applied the removal and their state no longer contains the resource.

Each GitHub environment supplies:

- `OPENTOFU_STATE_DATABASE_URL` → `PG_CONN_STR`
- `OPENTOFU_STATE_ENCRYPTION` → `TF_ENCRYPTION`
- `DIGITALOCEAN_ACCESS_TOKEN`
- `CLOUDFLARE_API_TOKEN`
- `UPSTASH_EMAIL` and `UPSTASH_API_KEY`

Repository secrets `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, and `NPM_TOKEN`
support publishing. GHCR uses the workflow's scoped `GITHUB_TOKEN`.
Backend connection strings and encryption keys must be backed up together;
neither belongs in source control or a normal build artifact.

```bash
# Local validation, without cloud credentials or a state connection
mise exec opentofu@1.12.6 -- bash scripts/infra/check.sh

# Apply using environment-scoped credentials and an exact image digest
TF_VAR_image_digest=sha256:<64-hex-digest> bash scripts/infra/apply.sh staging

# Exercise the real public tunnel path
BURROW_SMOKE_SERVER=staging.brmbl.dev go test ./burrowctl/internal -run TestLiveTunnel -count=1 -v
```

App Platform runs **one instance per environment** because active WebSocket and
yamux sessions live in Go memory. During a deployment, clients reconnect using
their existing reconnect behavior. Horizontal scaling requires connection-aware
routing and is not enabled by changing the replica count alone.

## Local Docker

```bash
cp .env.example .env
docker compose up -d
curl http://localhost:25701/health
```

For self-hosted HTTPS, set `BURROW_DOMAIN` to your tunnel domain and
`BURROW_TLS=true`, and terminate TLS in front of port 25701. Upstash connections
use `rediss://`; the container includes public CA certificates.
