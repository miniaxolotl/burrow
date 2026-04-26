# Deploy & Release

## Release Flow

1. Merge features into `development`
2. Release-please creates a PR with version bumps — merge it
3. Release-please creates a `v0.x.x` tag, triggering the release workflow:
   - **GoReleaser** builds platform binaries (linux-x64, linux-arm64, darwin-x64, darwin-arm64)
   - **npm publish** publishes `@miniaxolotl/burrowctl` + 4 platform packages
4. Merge `development` → `production` to deploy

## CI/CD Workflows

| Workflow | Trigger | Action |
|----------|---------|--------|
| `ci.yml` | PRs, pushes to `development`/`production` | Lint, build, test |
| `release.yml` | Push to `development` | Release-please PR |
| `release.yml` | Push `v*` tag | GoReleaser build + npm publish |
| `deploy.yml` | Push `v*` tag | Docker push to GHCR + Docker Hub |

## Docker Images

Both GHCR and Docker Hub receive identical tags on every release:

```
ghcr.io/miniaxolotl/burrowd:latest
ghcr.io/miniaxolotl/burrowd:v0.x.x
miniaxolotl/burrowd:latest
miniaxolotl/burrowd:v0.x.x
```

## Local Deploy (Docker)

Build and push images manually:

```bash
# Build locally (no push)
pnpm --filter @script/deploy run deploy

# Push to GHCR
GHCR_REGISTRY=ghcr.io/miniaxolotl pnpm --filter @script/deploy run deploy

# Push to Docker Hub
DOCKERHUB_REGISTRY=miniaxolotl pnpm --filter @script/deploy run deploy

# Push to both
GHCR_REGISTRY=ghcr.io/miniaxolotl DOCKERHUB_REGISTRY=miniaxolotl pnpm --filter @script/deploy run deploy
```

Tags: `latest` + version from `packages/burrowctl/package.json`.

## Local Release (npm)

Build binaries and publish to npm:

```bash
pnpm --filter @script/release run release
pnpm --filter @script/release run release -- --dry-run  # skip publish
```

Publishes:
- `@miniaxolotl/burrowctl` (main package with `optionalDependencies`)
- `@miniaxolotl/burrowctl-linux-x64`
- `@miniaxolotl/burrowctl-linux-arm64`
- `@miniaxolotl/burrowctl-darwin-x64`
- `@miniaxolotl/burrowctl-darwin-arm64`

## Dokku Deployment

### Prerequisites

- Dokku 0.27+ on your server
- Redis plugin: `dokku plugin:install https://github.com/dokku/dokku-redis.git redis`
- Wildcard DNS configured (`*.burrow.yourdomain.com` → server)

### Setup

```bash
dokku apps:create burrowd
dokku redis:create burrowd
dokku redis:link burrowd burrowd
dokku config:set burrowd BURROW_SECRET=your-secret BURROW_DOMAIN=burrow.yourdomain.com BURROW_PORT=25701
```

### SSL (Wildcard Certificates)

HTTP-01 challenge fails on wildcards. Use DNS-01:

```bash
sudo certbot certonly --dns-cloudflare -d "burrow.yourdomain.com" -d "*.burrow.yourdomain.com"
dokku certs:add burrowd /etc/letsencrypt/live/burrow.yourdomain.com/fullchain.pem /etc/letsencrypt/live/burrow.yourdomain.com/privkey.pem
```

### Deploy via Git

```bash
git remote add dokku dokku@your-server:dokku/burrowd
git push dokku production:master
```

### Nginx

The `nginx.conf.sigil` template handles WebSocket upgrades and SSL. Dokku auto-generates the config from it.
