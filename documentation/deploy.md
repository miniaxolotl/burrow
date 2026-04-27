# Deploy & Release

## Releases

Burrow uses [release-please](https://github.com/googleapis/release-please) for automated versioning.

### Workflow

1. Push conventional commits to `development`
2. Release-please creates a release PR
3. Merge → GitHub release + `v*` tag created
4. Deploy workflow triggers on tag → pushes Docker images

### Conventional Commits

| Prefix | Effect |
| ------ | ------ |
| `feat:` | Minor bump |
| `fix:` | Patch bump |
| `BREAKING CHANGE:` | Major bump |
| `docs:`, `chore:`, `refactor:` | No bump |

### Config

- `release-please-config.json` — single package (`.`) with `simple` release type
- `.release-please-manifest.json` — current version
- Updates `package.json`, platform `package.json` files, `burrowd/version.go`, `burrowctl/version.go`

## Deploy

### Docker Compose

```bash
cp .env.example .env
docker compose up -d
```

### Docker Run

```bash
docker run -p 25701:25701 \
  -e BURROW_SECRET=your-secret \
  -e BURROW_DOMAIN=burrow.example.com \
  -e BURROW_REDIS_URL=redis://redis:6379 \
  ghcr.io/miniaxolotl/burrowd:latest
```

### Dokku

```bash
dokku apps:create burrowd
dokku redis:create burrowd
dokku config:set burrowd BURROW_SECRET=secret BURROW_DOMAIN=burrow.example.com
git remote add dokku dokku@your-server:dokku/burrowd
git push dokku production:master
```

### Wildcard SSL

HTTP-01 challenge doesn't work for wildcards. Use DNS challenge:

```bash
certbot certonly --dns-cloudflare -d "burrow.example.com" -d "*.burrow.example.com"
dokku certs:add burrowd /etc/letsencrypt/live/burrow.example.com/fullchain.pem /etc/letsencrypt/live/burrow.example.com/privkey.pem
```

## CI/CD

| Workflow | Trigger | Action |
| -------- | ------- | ------ |
| `ci.yml` | Push/PR | Lint, build, test |
| `release.yml` | Push to `development` or `production` | Release PR on dev → npm publish on prod branch |
| `deploy.yml` | `v*` tag | Push Docker to GHCR + Docker Hub |

## Updating

```bash
# Docker
docker pull ghcr.io/miniaxolotl/burrowd:latest
docker compose down && docker compose up -d

# npm
npm update -g @miniaxolotl/burrowctl
```

## Troubleshooting

```bash
# Server health
curl http://localhost:25701/health

# Docker logs
docker compose logs -f burrowd

# Verify Redis
docker compose exec redis redis-cli ping
```
