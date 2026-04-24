# burrow

Expose local services to the internet via HTTPS subdomains.

Tunnel URLs look like: `https://mighty-arcane-dragon.inkspire.one`

## Quick Start (Docker Compose)

```bash
cp .env.example .env
# Edit .env — set BURROW_SECRET

docker compose up -d
```

Then on your local machine:

```bash
BURROW_SERVER=your-server.example.com BURROW_SECRET=xxx BURROW_TLS=true \
  burrowctl tunnel create --port 3000
```

## Local Development (no Docker)

```bash
# Start Redis
docker compose up -d redis

# Build
go build -o bin/burrowd ./burrowd
go build -o bin/burrowctl ./burrowctl

# Run server
./bin/burrowd serve --secret dev --domain localhost

# Create tunnel (separate terminal)
./bin/burrowctl tunnel create --server localhost:25701 --secret dev --port 3000
```

## Configuration

**Server (`burrowd`)**

| Variable | Default | Description |
|----------|---------|-------------|
| `BURROW_SECRET` | required | Shared auth secret |
| `BURROW_DOMAIN` | `inkspire.one` | Tunnel subdomain base |
| `BURROW_PORT` | `25701` | Listen port |
| `BURROW_REDIS_URL` | `redis://localhost:6379` | Redis URL |
| `BURROW_TLS` | `false` | Generate `https://` tunnel URLs (set `true` when behind HTTPS proxy) |

Also accepts `PORT` and `REDIS_URL` as fallbacks (Dokku convention).

**Client (`burrowctl`)**

| Variable | Default | Description |
|----------|---------|-------------|
| `BURROW_SERVER` | `localhost:25701` | Server address |
| `BURROW_SECRET` | — | Shared secret (auto-generates token) |
| `BURROW_TOKEN` | — | Pre-generated token (alternative to secret) |
| `BURROW_TLS` | `false` | Use `wss://` and `https://` when connecting to the server |
| `BURROW_DOMAIN` | `inkspire.one` | Fallback domain for tunnel URL display |

## API

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/health` | GET | No | Status, uptime, active tunnel count |
| `/tunnels` | GET | Yes | List active tunnels |
| `/tunnel/ws` | GET | Yes | WebSocket — establish tunnel |
| `/tunnel/{id}` | DELETE | Yes | Close tunnel |

## Deployment (Dokku)

```bash
dokku apps:create burrowd
dokku plugin:install https://github.com/dokku/dokku-redis.git
dokku redis:create burrowd-redis && dokku redis:link burrowd-redis burrowd
dokku domains:set burrowd inkspire.one '*.inkspire.one'
dokku config:set burrowd BURROW_SECRET=xxx BURROW_DOMAIN=inkspire.one BURROW_TLS=true
git push dokku production:main
dokku letsencrypt:enable burrowd
```

## License

MIT
