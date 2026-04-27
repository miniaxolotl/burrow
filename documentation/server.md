# burrowd

Tunnel server that manages WebSocket connections from clients, proxies HTTP/TCP traffic, and assigns public subdomain URLs.

## Quick Start

```bash
cp .env.example .env   # set BURROW_SECRET and BURROW_DOMAIN
docker compose up -d
```

## Configuration

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `BURROW_SECRET` | Admin secret — if set, admin endpoints require a valid token | — |
| `BURROW_DOMAIN` | Domain for tunnel URLs | `localhost` |
| `BURROW_PORT` | Server port | `25701` |
| `BURROW_REDIS_URL` | Redis connection URL | `redis://redis:6379` |

Also accepts `PORT` and `REDIS_URL` as fallbacks.

`BURROW_SECRET` is optional. Without it the server runs in open mode: anyone can create tunnels and all admin endpoints are disabled.

## Auth model

Tunnel **creation** is always open — no token required.

Admin operations (listing all tunnels, revoking any tunnel) require a token generated from `BURROW_SECRET`:

```bash
# Generate a token with burrowd:
burrowd auth login <token>

# Or with burrowctl (client-side):
burrowctl auth login <token> --server your-server.example.com
```

Per-tunnel operations (close, logs) are available to the owner token that created the tunnel, or to any admin token. Anonymous tunnels (created without a token) can be managed by anyone with the tunnel ID.

## Commands

```bash
burrowd serve --secret mysecret --domain burrow.example.com --port 25701
burrowd stop                          # Graceful stop via PID file
burrowd auth login <token>            # Validate and store token locally
burrowd auth status                   # Check stored auth status
burrowd auth logout                   # Remove stored token
burrowd tunnel list                   # List active tunnels
burrowd tunnel revoke <id>            # Force-close a tunnel
```

## API

| Endpoint | Method | Auth | Description |
| -------- | ------ | ---- | ----------- |
| `/health` | GET | none | Server health and tunnel count |
| `/tunnel/ws` | WebSocket | none | Client tunnel connection |
| `/tunnel/<id>` | DELETE | owner or admin | Close a tunnel |
| `/logs/<id>` | GET | owner or admin | Request logs for a tunnel (max 200) |
| `/tunnels` | GET | admin | List all active tunnels |

## Tunnel URLs

```
https://{tunnel_id}.{domain}
```

IDs are generated as `{adverb}-{adjective}-{adjective}-{noun}` (e.g. `swiftly-ancient-silent-dragon`). ~288M combinations.

## Architecture

```
┌──────────────┐         ┌──────────────┐         ┌──────────────┐
│  burrowctl   │◄────────│   burrowd    │◄────────│  End User    │
│  (client)    │   WS    │   (server)   │  HTTP   │  (browser)   │
└──────────────┘         └──────────────┘         └──────────────┘
```

## Deployment

### Docker Compose

```bash
cp .env.example .env
docker compose up -d
```

### Dokku

```bash
dokku apps:create burrowd
dokku redis:create burrowd
dokku config:set burrowd BURROW_SECRET=your-secret BURROW_DOMAIN=burrow.example.com
git remote add dokku dokku@your-server:dokku/burrowd
git push dokku production:master
```

Wildcard SSL via certbot + DNS challenge (HTTP-01 doesn't work for wildcards).

## Documentation

- [Client reference](client.md)
- [Deploy & release](deploy.md)
