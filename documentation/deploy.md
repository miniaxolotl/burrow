# Deploy & release

## Branch and release policy

Feature PRs integrate into `development`. Release Please prepares a single product
version and changelog there; all native packages, the wrapper and both Go binaries
share it. Both `feat` and `feature` produce feature notes. A source GitHub release
is metadata, not a deployment.

Only `production` publishes npm, binary archives, container images and the Vercel
production site. Development runs CI and Release Please. Existing staging resources
are retained; this policy does not retire their data, DNS or certificates.

## Retained environments

|                   | Production               | Staging                          |
| ----------------- | ------------------------ | -------------------------------- |
| Deployment branch | `production`             | Disabled; retained resources     |
| Website (Vercel)  | `https://useburrow.app`  | `https://staging.useburrow.app`  |
| Tunnel server     | `https://brmbl.dev`      | `https://staging.brmbl.dev`      |
| Tunnel URLs       | `https://<id>.brmbl.dev` | `https://<id>.staging.brmbl.dev` |
| DigitalOcean app  | `burrow`                 | `burrow-staging`                 |
| Upstash Redis     | `burrow`                 | `burrow-staging`                 |
| OpenTofu root     | `infra/production`       | `infra/staging`                  |

Vercel builds `packages/website` from `production` only, with
`SITE_URL=https://useburrow.app`. Preview deployments are disabled and the project
ignore command skips every ref except production. Protected promotion CI validates
the complete website tree before the production merge.
`www.useburrow.app` redirects to the production apex.

## Pipeline

1. **CI** runs website lint/types/build, Go vet/build/tests, OpenTofu formatting
   and validation, workflow lint, and a Docker build.
2. **CI Gate** aggregates every job. A downstream PR must come from the owner's
   frozen `promote/production/vX.Y.Z` ref, have Release Please provenance and match
   the tag's tree. Independent source CI is required; PR/self evidence is rejected.
3. **Deploy** waits for complete production CI, resolves the tag source separately
   from the production merge SHA and rejects stale production runs. It builds
   GoReleaser artifacts, checks every npm tarball and existing GitHub asset before
   publication, and preserves the Release Please release body. Conflicting bytes
   under an existing version fail; identical artifacts are reused on retry.
4. It publishes packages and archives, then promotes the exact source image in
   GHCR/Docker Hub with `sha-<source>`, version and `latest` tags. Readback verifies
   the source revision and manifest digest. Repeat runs reuse the immutable image.
5. OpenTofu applies production using the image's **digest**, not
   a mutable tag. It owns the complete App Platform specification, Upstash
   database and Cloudflare website/tunnel DNS.
6. App Platform supplies wildcard certificate verification tokens. Deployment
   refreshes their TXT records through OpenTofu. **Infrastructure Maintenance**
   may also be dispatched explicitly on production for certificate reconciliation.
7. Deployment waits for HTTPS health, then creates a real WSS/yamux tunnel and
   checks HTTP body, headers, query strings, and API-like paths end to end.

The npm registry serves npm, pnpm, Yarn, and Bun. Native archives cover Linux and
macOS on amd64/arm64. Docker images cover Linux amd64/arm64. There is currently
no Homebrew tap, winget package, or apt repository in this project.

## Select and promote

1. Merge the green Release Please PR into development and wait for its immutable
   `vX.Y.Z` source release and complete development CI.
2. Run **Select Release Candidate** from development with that exact tag. It uses
   the job-scoped `GITHUB_TOKEN`, freezes the tag tree and current production parent, and opens
   a labelled PR assigned to `miniaxolotl`. It never follows later development.
   A run-scoped candidate artifact binds the operator, tooling revision, source,
   tree, frozen head and PR. CI verifies this record for bot-authored promotions.
   Records expire after 90 days; rerun the original selection run to refresh them.
3. Review the identity in the PR and enable auto-merge using **merge**, not squash
   or rebase. Strict `CI Gate`, conversation resolution and no-bypass protection
   apply. Source tags and frozen candidate refs cannot be rewritten or deleted.
4. Verify Deploy, Vercel production readiness, npm versions, GitHub archives and
   the real tunnel smoke test. The branch trees may match with different SHAs.

Rerun a failed Deploy on the same current production commit. Publication is
non-atomic across providers: an error remains visible, and successful immutable
artifacts are reused. Never delete or overwrite an existing version to retry.
After each npm upload, registry integrity is read back up to 60 times, five seconds
apart. Only a missing version is retried; conflicting bytes or provider errors fail
immediately. This accommodates npm's asynchronous processing without republishing.
Native package metadata explicitly preserves the binary's executable bit through
`pnpm pack`. Before any publication, all five tarballs are extracted, every native
binary's mode and version pin is checked, and the packed wrapper must launch the
host binary with the exact release version. This gate also runs on dry-run.
Final packing canonicalizes resolved manifest keys so asynchronous workspace-range
conversion cannot change tarball integrity across retries.
Conflicting base changes require integration into development and a new release;
a rejected/closed candidate is not reopened automatically. Rollbacks use a reviewed
new release/revert, preserving immutable history. Ordinary development pushes do
not open or merge promotions.

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
Release Please and candidate selection use `GITHUB_TOKEN` with contents/PR write
and Actions dispatch permissions. They explicitly dispatch CI on generated refs,
because token-created PRs do not trigger ordinary PR workflows. No PAT is needed.
Backend connection strings and encryption keys must be backed up together;
neither belongs in source control or a normal build artifact.

```bash
# Local validation, without cloud credentials or a state connection
mise exec opentofu@1.12.6 -- bash scripts/infra/check.sh

# Apply via Deploy or Infrastructure Maintenance on the production workflow ref.
# Local apply is an explicit operator repair, not the release path.

# Exercise the real public tunnel path
BURROW_SMOKE_SERVER=brmbl.dev go test ./burrowctl/internal -run TestLiveTunnel -count=1 -v
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
