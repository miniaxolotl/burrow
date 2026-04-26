# burrowd

Tunnel server that manages WebSocket connections from clients, proxies HTTP/TCP traffic, and assigns public subdomain URLs.

## Quick Start

```bash
cp .env.example .env
docker compose up -d
```

## Configuration

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `BURROW_SECRET` | Auth secret (required) | — |
| `BURROW_DOMAIN` | Domain for tunnel URLs | `localhost` |
| `BURROW_PORT` | Server port | `25701` |
| `BURROW_REDIS_URL` | Redis connection URL | `redis://redis:6379` |
| `BURROW_TLS` | Generate `https://` URLs | `false` |

Also accepts `PORT` and `REDIS_URL` as fallbacks.

## Commands

```bash
burrowd serve --secret mysecret --domain burrow.example.com --port 25701
burrowd stop                          # Graceful stop via PID file
burrowd auth login {token}            # Validate and store token
burrowd auth status                   # Check auth status
burrowd auth logout                   # Remove stored token
burrowd tunnel list                   # List active tunnels
burrowd tunnel revoke {id}            # Force-close a tunnel
```

## API

| Endpoint | Method | Description |
| -------- | ------ | ----------- |
| `/health` | GET | Server health and tunnel count |
| `/tunnel/ws` | WebSocket | Client tunnel connection (yamux) |
| `/tunnels` | GET | List all active tunnels |
| `/tunnel/{id}` | DELETE | Revoke a tunnel |
| `/logs/{id}` | GET | Request logs for a tunnel (max 200) |

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
