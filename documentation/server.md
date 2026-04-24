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

**Example:**
```bash
docker compose up -d
# Or manually:
burrowd serve --port 25701 --domain mawa.dev --redis-url redis://redis:6379 --secret my-secret-key
```

### `burrowd auth login`

Authenticate with a token to access tunnel services.

```bash
burrowd auth login {token}
```

**Example:**
```bash
burrowd auth login my-secret-token
```

**Behavior:**
- Stores the token locally (encrypted in config)
- Validates token against configured secret
- Returns error if token is invalid

### `burrowd auth status`

Check current authentication status.

```bash
burrowd auth status
```

### `burrowd auth logout`

Remove stored authentication token.

```bash
burrowd auth logout
```

### `burrowd tunnel list`

List all active tunnels on this server.

```bash
burrowd tunnel list
```

### `burrowd tunnel revoke`

Force-close a specific tunnel.

```bash
burrowd tunnel revoke {tunnel_id}
```

### `burrowd stop`

Stop the server gracefully.

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
        │ ─────────────────────────►│ swiftly-silent-dragon.mawa.dev │
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

Examples:
- Domain `tryburrow.dev`: `https://swiftly-ancient-silent-dragon.tryburrow.dev`
- Domain `burrow.mawa.dev`: `https://swiftly-ancient-silent-dragon.burrow.mawa.dev`
- Domain `burrow-preview.mawa.dev`: `https://swiftly-ancient-silent-dragon.burrow-preview.mawa.dev`

## API Endpoints

### Health Check

```bash
curl http://localhost:25701/health
```

**Response:**
```json
{
  "status": "ok",
  "tunnels": 0,
  "uptime": "1h30m"
}
```

### WebSocket: `/tunnel/ws`

- **Auth**: Token via `X-Tunnel-Token` header or `token` query param
- **Protocol**: yamux for stream multiplexing

## Docker Compose

```bash
docker compose up -d
```

This starts:
- `burrowd` server on port 25701
- `redis` on port 6379
