# Deploy & Release Guide

## Server Deployment

### Docker (Recommended)

```bash
docker run -d --name burrowd \
  -p 25701:25701 \
  -e BURROW_SECRET=your-secret \
  -e BURROW_DOMAIN=your-domain.com \
  ghcr.io/miniaxolotl/burrowd:latest
```

### Docker Compose

```bash
cp .env.example .env
# Edit .env — set BURROW_SECRET
docker compose up -d
```

### Docker Images

| Registry | Image |
|----------|-------|
| GHCR | `ghcr.io/miniaxolotl/burrowd` |
| Docker Hub | `miniaxolotl/burrowd` |

Tags: `latest`, `v{major}.{minor}.{patch}`

### Dokku

See [Server documentation](../server.md#deploy-dokku) for app creation, Redis setup, and SSL configuration.

### Binaries

Download from [Releases](https://github.com/miniaxolotl/burrow/releases):

```bash
curl -fsSL https://github.com/miniaxolotl/burrow/releases/latest/download/burrowd_<version>_linux_amd64.tar.gz | tar -xz
sudo mv burrowd /usr/local/bin/
```

## Versioning

Burrow uses [release-please](https://github.com/googleapis/release-please) for automatic version management.

### How It Works

1. Developers write conventional commit messages:
   - `feat:` → minor version bump (0.2.0 → 0.3.0)
   - `fix:` → patch version bump (0.2.0 → 0.2.1)
   - `chore:`, `docs:`, `ci:` → no version bump

2. On every push to `production`, release-please creates or updates a release PR

3. Merging the release PR:
   - Creates a git tag (`v0.2.3`)
   - Creates a GitHub Release
   - Updates all version files automatically

### Release-Please Configuration

Tracked files (all bumped together):
- `package.json` (root)
- `packages/burrowctl/package.json`
- `packages/burrowctl-*/package.json` (4 platform packages)
- `burrowd/version.go`
- `burrowctl/version.go`

Config: `release-please-config.json`

## Release Workflow

When a tag `v*` is pushed, `.github/workflows/release.yml` runs:

1. **Builds burrowctl binaries** with goreleaser (linux/darwin × amd64/arm64)
2. **Publishes to npm** — 5 packages:
   - `@miniaxolotl/burrowctl` (main package with `bin/burrowctl.js`)
   - `@miniaxolotl/burrowctl-linux-x64`
   - `@miniaxolotl/burrowctl-linux-arm64`
   - `@miniaxolotl/burrowctl-darwin-x64`
   - `@miniaxolotl/burrowctl-darwin-arm64`

## Deploy Workflow

When a tag `v*` is pushed, `.github/workflows/deploy.yml` runs:

1. **Builds multi-platform Docker image** (linux/amd64, linux/arm64)
2. **Pushes to GHCR** — `ghcr.io/miniaxolotl/burrowd`
3. **Pushes to Docker Hub** — `miniaxolotl/burrowd` (if credentials configured)

## Manual Release & Deploy

### Release (npm + GitHub binaries)

```bash
pnpm run release              # build + publish to npm
pnpm run release -- --dry-run # dry run (no publish)
```

Requires: Go (for goreleaser), npm auth token

### Deploy (Docker images)

```bash
GHCR_REGISTRY=ghcr.io/miniaxolotl \
DOCKERHUB_REGISTRY=miniaxolotl \
TAG=v0.2.3 \
pnpm run deploy
```

Requires: Docker with buildx, GH_TOKEN or GITHUB_TOKEN, DOCKERHUB_TOKEN (optional)

### Full Release Cycle

```bash
# 1. Bump versions (release-please handles this automatically)
#    Or manually edit version files

# 2. Commit and push to production
git add . && git commit -m "chore: release v0.2.3"
git push origin production

# 3. Merge the release-please PR (creates tag + GitHub release)

# 4. Workflows trigger automatically:
#    - release.yml  → goreleaser binaries + npm publish
#    - deploy.yml   → Docker images to GHCR + Docker Hub
```

## Updating

```bash
# Server
docker pull ghcr.io/miniaxolotl/burrowd:latest
docker compose down && docker compose up -d

# Client
npm update -g @miniaxolotl/burrowctl
```

## Support

- Issues: https://github.com/miniaxolotl/burrow/issues
- Docker Hub: https://hub.docker.com/r/miniaxolotl/burrowd
- GHCR: https://github.com/miniaxolotl/burrow/pkgs/container/burrowd
