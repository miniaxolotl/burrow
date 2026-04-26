# Deploy Guide

This guide covers deploying burrowd server in production.

## Deployment Methods

### 1. Docker Compose (Recommended)

```bash
cp .env.example .env
# Edit .env with your BURROW_SECRET
docker compose up -d
```

This starts burrowd on port 25701 with Redis.

### 2. Dokku

See [Server documentation](documentation/server.md#deploy-dokku) for:
- App creation and Redis setup
- Wildcard SSL certificates
- Nginx configuration

### 3. Binaries

Pre-built binaries for Linux (amd64, arm64) and macOS (amd64, arm64) are available on the [Releases](https://github.com/miniaxolotl/burrow/releases) page:

```bash
# Linux amd64
curl -fsSL https://github.com/miniaxolotl/burrow/releases/latest/download/burrowd_0.2.0_linux_amd64.tar.gz | tar -xz
sudo mv burrowd /usr/local/bin/

# macOS arm64
curl -fsSL https://github.com/miniaxolotl/burrow/releases/latest/download/burrowd_0.2.0_darwin_arm64.tar.gz | tar -xz
sudo mv burrowd /usr/local/bin/
```

### 4. Install Script

```bash
curl -fsSL https://raw.githubusercontent.com/miniaxolotl/burrow/production/scripts/install.sh | sh
```

Local install (no sudo):
```bash
curl -fsSL https://raw.githubusercontent.com/miniaxolotl/burrow/production/scripts/install.sh | sh -s -- --local
```

## Docker Registry

Images are published to GHCR and Docker Hub:

| Registry | Image |
|----------|-------|
| GHCR | `ghcr.io/miniaxolotl/burrowd` |
| Docker Hub | `miniaxolotl/burrowd` |

Tags: `latest`, `v{major}.{minor}.{patch}`, `{major}.{minor}.{patch}`

## Versioning

Burrow uses [release-please](https://github.com/googleapis/release-please) for automatic version bumping. Merge a PR to `production` with conventional commits (`feat:`, `fix:`, `chore:`) and release-please creates a release PR. Merging it tags the release and triggers the release + deploy workflows.

### Manual Release

```bash
pnpm --filter @script/release run release -- --dry-run   # dry run
pnpm --filter @script/release run release                # actually release
```

## Updating

```bash
# Pull latest image
docker pull ghcr.io/miniaxolotl/burrowd:latest

# Restart
docker compose down
docker compose up -d
```

## Support

- Issues: https://github.com/miniaxolotl/burrow/issues
- Docker Hub: https://hub.docker.com/r/miniaxolotl/burrowd
- GHCR: https://github.com/miniaxolotl/burrow/pkgs/container/burrowd
