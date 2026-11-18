# Deploy & release

## Persistent attachment compatibility floor

After persistent-mode IDs are enabled, pre-fencing server images are not safe
rollback targets. Every `scripts/infra/apply.sh` path now pulls the exact immutable
GHCR digest and requires `io.bramble.burrow.attachment-state=1` before planning or
applying infrastructure. Missing/newer/older labels and registry failures fail
closed. The existing single-instance and current-ref gates remain required.
Use a verified compatible last-good digest or a reviewed forward fix. Do not erase
owner mode/generation records or tombstones to permit downgrade. Older deployment
scripts/manual provider mutations are not supported bypasses; preserve this guard
when reviewing any release or rollback. This is source policy, not evidence that
a deployment has occurred.

Follow [shared release/naming conventions](release-conventions.md). Release Please
uses the upstream Node strategy for the root/npm/Go version mappings, patch-first
automatic bumps and explicit `Release-As` minor/major/breaking decisions. Existing
versions and tags are retained; generated release PRs own version/changelog edits.

Shared workflow names, action pins, runtime and evidence conventions follow the
[cross-repository CI/CD baseline](pipeline-conventions.md).

The [Burrow hardening review](cicd-hardening-review.md) separates implemented
admission/publication/rollout from planned selection, drain continuity and automatic
last-good recovery. Continue using this operator path until replacements are
reviewed and separately activated.

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
2. **CI Gate** verifies classification, admission and exact selected-job results;
   unknown/empty/unreadable diffs select all. Development PRs select dependency
   families, while source push/manual validation stays complete. Frozen promotion
   PRs reuse independent source validation through authenticated admission.
   A downstream PR must come from the owner's
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
7. Deployment first waits for the provider's active single-instance generation
   to match the exact requested digest, with no deployment still in progress.
   A healthy previous generation cannot satisfy this five-minute bounded gate.
   It then checks HTTPS health and a real WSS/yamux tunnel: HTTP body, headers,
   query strings and API-like paths must all match end to end. Synthetic smoke
   requests may retry edge 502/503/504 responses within their existing 90-second
   caller deadline while routes converge. Other statuses, wrong successful bodies
   and deadline expiry still fail; this is not business-request retry policy.

The npm registry serves npm, pnpm, Yarn, and Bun. Native archives cover Linux and
macOS on amd64/arm64. Docker images cover Linux amd64/arm64. There is currently
no Homebrew tap, winget package, or apt repository in this project.

### Compare candidate binaries (report only)

Use `scripts/release/replay-binaries.sh BASE CANDIDATE TARGET` to check whether
the compiled `burrowd` and `burrowctl` bytes changed independently. Run it with
Git, Bash, `sha256sum` and a Go toolchain that builds both reviewed commits.
It builds clean, detached Git worktrees so tracked `export-ignore` files are
included and uncommitted edits are excluded. Both builds use the same local
compiler and flags, normalize the marked product version, and disable VCS
stamping. The report records the compiler, target, CPU level, both SHA-256s
and changed paths. It does not execute the binaries or require release credentials.

The build starts with only `HOME` and `PATH` from the caller and ignores saved
Go settings. It pins `CGO_ENABLED=0` and either `GOAMD64=v1` or `GOARM64=v8.0`.
These match `go version -m` on the checksum-verified v0.3.23 Linux CLI archives;
recheck the levels when the release configuration changes.

```bash
bash scripts/release/replay-binaries.sh 'a57f87c^' a57f87c linux/amd64 # website #106
bash scripts/release/replay-binaries.sh 'c36f277^' c36f277 linux/amd64 # server fix #81
bash scripts/release/replay-binaries.sh '6c0c5ba^' 6c0c5ba linux/amd64 # protocol #97
```

| Replay (Go 1.26.10, Linux amd64) | `burrowd` | `burrowctl` | Delivery inputs to review                                                                                             |
| -------------------------------- | --------- | ----------- | --------------------------------------------------------------------------------------------------------------------- |
| Website #106                     | Identical | Identical   | Vercel site sources changed.                                                                                          |
| Server #81                       | Different | Identical   | Server image, archives and tunnel; `documentation/server.md` also feeds the site. The changed client files are tests. |
| Protocol #97                     | Different | Different   | Server, client archives and pinned native family; verify mixed-version compatibility.                                 |

For a delivery review, use the source of the last **verified delivery of that
target**, not necessarily the last product tag. Repeat for Linux and Darwin on
amd64 and arm64. Review the full diff too: wrapper installation, Docker base,
build flags, archive contents, and the site (including rendered reference docs)
can change independently of these bytes. Neither paths nor a different hash
authorize an npm bump. Identify what changes for an installer and inspect the
packed wrapper's exact native pins against the registry.

In particular, the v0.3.22 and v0.3.23 **published** wrappers differ only in
their version and four exact pins. The v0.3.23 Linux amd64 server and client
archives each also contain `CHANGELOG.md`, `LICENSE` and `README.md`. A replay
digest is not a release checksum: verify the final stamped archives against
`checksums.txt`. Before splitting delivery, preserve a complete checksum list
for the selected archives and the last verified download links for the rest.
Server changes still need image-digest and real HTTPS tunnel verification;
protocol changes also need an installed client against the candidate server.
The coordinated production pipeline above remains in effect.

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
Extraction preserves archive permissions, making retries independent of the
operator's umask as well as the runner's.
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
for independent exact-head evidence. Native bot-PR runs can require maintainer
approval; review/approve those runs and verify the actual PR checks before merging.
No PAT is needed. The retained `staging` environment only admits the `staging`
branch; production delivery uses the production-only environment.
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

The smoke test uses a fresh in-memory owner credential, checks apex health between
tunnel requests, and explicitly revokes its own tunnel at completion. It never
loads a globally installed client's configuration.

### Bounded live load testing

Run only with authorization for the target service. This is intentionally **not**
part of deployment smoke testing or normal CI:

```bash
BURROW_LOAD_SERVER=brmbl.dev go test ./burrowctl/internal -run '^TestLiveTunnelLoad$' -count=1 -v -timeout 4m
```

The fixture uses a private temporary HOME/config, fresh in-memory owner and unique
response marker. It exposes only synthetic loopback services through three new
tunnels. It never invokes or edits the global CLI or another owner's tunnels.
Run from a separate machine/temporary Droplet for an independent test location;
copy only a compiled test binary, not local credentials or provider secrets.

The fixed ramp is 5, 15, 30, 60 and 100 requests/second for ten seconds each, with
at most 32 in flight and 2,100 measured small requests overall. Measured requests
are never retried. Incorrect status/marker/path/body, a request failure or observed
latency over two seconds stops the ramp. The test also checks management routing,
64 KiB/1 MiB/4 MiB upload-and-echo integrity, the 200-entry history bound, response
usage retained after history eviction, and rejection of a transport message over
1 MiB. Cleanup revokes only its own IDs and verifies its owner has no active
tunnels; persistent revoked ownership tombstones remain by design.

Capture actual completed requests, achieved throughput, errors and latency
percentiles. A short successful ramp establishes only the workload tested, not
maximum capacity, a sustained-load SLO or deployment continuity. Stop on baseline
correctness failures instead of interpreting them as saturation. Destroy any
temporary Droplet and SSH key and verify their absence, allowing for provider
readback delay. [Investigation and reproduction](request-isolation.md).

For separately authorized heavier testing, the normal smoke/load flags cannot
enable the heavy profile. Both an explicit hostname and its measured-request
ceiling acknowledgement are required:

```bash
BURROW_HEAVY_LOAD_SERVER=brmbl.dev BURROW_HEAVY_LOAD_CONFIRM=52000 go test ./burrowctl/internal -run '^TestLiveTunnelHeavyLoad$' -count=1 -v -timeout 4m
```

This fixed profile plans 100/250/500/750/1,000 RPS, 20 seconds per stage, at most
128 in flight and 52,000 measured 1-KiB echo requests, plus bounded control and
payload probes. It retains fresh owner/config isolation, integrity checks and
first-error/observed-latency safety stops, and checks apex health before stages.
Capture generator resource use separately from production-server resource use.
Do not increase ceilings or relax guards to hide a failed stage. The
[5 November heavy run](heavy-load-2026-11-05.md) stopped at its 250-RPS latency guard;
the higher planned stages were not executed.

### Deployment continuity limits

App Platform runs **one instance per environment** because active WebSocket and
yamux sessions live in Go memory. During a deployment, clients reconnect using
their existing reconnect behavior. Horizontal scaling requires connection-aware
routing and is not enabled by changing the replica count alone.

Current server shutdown closes existing tunnel connections before shutting down
HTTP. Reconnect support is not an in-flight continuity or zero-downtime guarantee;
the hardening review requires explicit drain/reconnect failure tests and hosted
acceptance before making such a claim.

The [tunnel operations plan](tunnel-operations-plan.md) defines bounded
admission/draining, optional client notices and separate readiness. It preserves
the single-instance and image-generation gates. No readiness endpoint, provider
health-path change or drain guarantee is implemented by that planning PR; hosted
termination-order acceptance requires separate authorization.

## Local Docker

```bash
cp .env.example .env
docker compose up -d
curl http://localhost:25701/health
```

For self-hosted HTTPS, set `BURROW_DOMAIN` to your tunnel domain and
`BURROW_TLS=true`, and terminate TLS in front of port 25701. Upstash connections
use `rediss://`; the container includes public CA certificates.

# Site-only delivery of v0.3.24

Selection and promotion admission check the production parent and the reviewed #109/#110 product tree in `scripts/release/policy.json`. Release Please updates the private product version; the wrapper, its four native packages, Go version stamps, download archives and tunnel image stay at their last verified delivery. The production merge triggers the existing Vercel Git integration. The artifact publisher is skipped and its dependent infrastructure rollout cannot run.

Verify the Vercel production alias resolves to the deployment for the **production merge SHA**, fetch the live site, read back the active DigitalOcean deployment/digest and npm pins, and check the previous release assets. A new tag or changed baseline needs a newly reviewed delivery decision; do not use the legacy full publisher to recover a missing receipt.
