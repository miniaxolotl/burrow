# Burrowd - Server Documentation

## Overview

`burrowd` is the server component that handles tunnel requests from clients, manages tunnel lifecycle, and proxies incoming HTTP/TCP traffic to connected clients.

## Configuration

Configuration is loaded from (in order of precedence):
1. Command-line flags
2. Environment variables (`.env`)
3. Default values

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BURROW_SECRET` | Authentication secret | (required) |
| `BURROW_DOMAIN` | Domain for tunnel URLs | `mawa.dev` |
| `BURROW_PORT` | Server port | `25701` |
| `BURROW_REDIS_URL` | Redis connection URL | `localhost:6379` |
| `BURROW_TLS` | Generate `https://` tunnel URLs | `false` |

Also accepts `PORT` and `REDIS_URL` as fallbacks (Dokku convention).

### .env Example
```
BURROW_REDIS_URL=redis://redis:6379
BURROW_SECRET=your-secret-key
BURROW_DOMAIN=mawa.dev
BURROW_PORT=25701
BURROW_TLS=true
```

## Commands

### `burrowd serve`

Starts the tunnel server.

```bash
burrowd serve --secret mysecret --domain mawa.dev --port 25701
```

**Flags:**
- `--port` - Port to listen on (default: `25701`)
- `--domain` - Domain for tunnel URLs (default: `mawa.dev`)
- `--redis-url` - Redis connection URL (default: `localhost:6379`)
- `--secret` - Authentication secret (required)
- `--tls` - Generate `https://` tunnel URLs (set when server is behind HTTPS proxy)

### `burrowd auth login`

Authenticate with a token. Validates against the configured secret and stores the token.

```bash
burrowd auth login {token}
```

**Behavior:**
- Validates token HMAC against `BURROW_SECRET`
- Stores the token as plaintext in `~/.burrow/token` (0600 permissions)

### `burrowd auth status`

Check current authentication status. Shows masked token if `~/.burrow/token` exists.

### `burrowd auth logout`

Remove stored authentication token.

### `burrowd tunnel list`

List all active tunnels by querying the running server's `GET /tunnels` API.

```bash
burrowd tunnel list
burrowd tunnel list --server burrow.example.com:25701 --token mytoken
```

**Requires:** `burrowd serve` must be running. Use `--server` and `--token` (or `BURROW_SERVER` / `BURROW_TOKEN` env vars) to authenticate.

### `burrowd tunnel revoke`

Force-close a specific tunnel by calling the server's `DELETE /tunnel/{id}` API. Closes both the Redis record and the live yamux session.

```bash
burrowd tunnel revoke {tunnel_id}
```

**Requires:** `burrowd serve` must be running. Use `--server` and `--token` (or `BURROW_SERVER` / `BURROW_TOKEN` env vars) to authenticate.

### `burrowd stop`

Stop the server gracefully via PID file.

```bash
burrowd stop
```

## Architecture

```
┌──────────────────┐         ┌──────────────────┐         ┌──────────────────┐
│    burrowctl     │◄────────│     burrowd      │◄────────│    End User      │
│    (client)      │   WS    │    (server)      │  HTTP   │   (browser)      │
└──────────────────┘         └──────────────────┘         └──────────────────┘
        │                            │                            │
        │ WebSocket + token auth     │ subdomain routing          │
        │ ─────────────────────────►│ swiftly-silent-dragon.burrow.mawa.dev │
        │                            │                             │
        │                            │ ┌───────────────────────────┴──► localhost:8080
        │◄───────────────────────── │ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─► │
        │    proxied response       │    HTTP request               │
        └────────────────────────────┘
```

## Tunnel URL Format

Tunnels are assigned human-readable IDs and served as subdomains:

```
https://{tunnel_id}.{domain}
```

The server extracts the tunnel ID by matching the host against the configured domain suffix. This supports any domain depth:

- Domain `tryburrow.dev`: `https://swiftly-ancient-silent-dragon.tryburrow.dev`
- Domain `burrow.mawa.dev`: `https://swiftly-ancient-silent-dragon.burrow.mawa.dev`

## API Endpoints

### Health Check

```bash
curl http://localhost:25701/health
```

**Response:**
```json
{"status":"ok","tunnels":0,"uptime":"1h30m"}
```

### WebSocket: `/tunnel/ws`

- **Auth**: Token via `X-Tunnel-Token` header or `token` query param
- **Protocol**: yamux for stream multiplexing

### Tunnel Logs: `/logs/{tunnel_id}`

- **Auth**: Token via `X-Tunnel-Token` header
- **Returns**: JSON array of request log entries (timestamp, method, path, size, duration)
- **Max**: 200 entries per tunnel (circular buffer, in-memory)

## Request Logging

Both HTTP and WebSocket requests are logged in-memory per tunnel. Each entry captures:
- Timestamp, method, path, response size, duration

Logs are exposed via `GET /logs/{tunnel_id}` and viewed in the TUI (`l` key).

## Deployment (Dokku)

### Nginx Configuration

The `nginx.conf.sigil` template is used by Dokku to generate per-app nginx configs:
- Uses `.DOKKU_APP_WEB_LISTENERS` for upstream generation (reliable for Dockerfile deploys)
- Supports WebSocket upgrades via `map $http_upgrade $connection_upgrade`
- SSL block is conditionally rendered when `.SSL_INUSE` is set

### SSL with Wildcard Domains

`dokku letsencrypt:enable` uses HTTP-01 challenge which fails on wildcard domains. Instead:

1. Install `certbot` + `certbot-dns-cloudflare` on the Dokku host
2. Obtain wildcard cert: `certbot certonly --dns-cloudflare -d "burrow.mawa.dev" -d "*.burrow.mawa.dev"`
3. Install via dokku: `dokku certs:add <app> /etc/letsencrypt/live/burrow.mawa.dev/fullchain.pem /etc/letsencrypt/live/burrow.mawa.dev/privkey.pem`

### App Configuration

| App | Domain | Port | SSL |
|-----|--------|------|-----|
| burrowd | burrow.mawa.dev | 25701 | Wildcard cert |
| burrowd-preview | burrow-preview.mawa.dev | 25702 | Wildcard cert |

## Docker Compose

```bash
docker compose up -d
```

This starts:
- `burrowd` server on port 25701
- `redis` on port 6379
