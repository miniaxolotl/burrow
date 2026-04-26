# burrow

[![npm](https://img.shields.io/npm/v/@miniaxolotl/burrowctl)](https://npmjs.com/package/@miniaxolotl/burrowctl)
[![GHCR](https://img.shields.io/github/v/release/miniaxolotl/burrow?label=ghcr&color=light)](https://github.com/miniaxolotl/burrow/pkgs/container/burrowd)
[![Docker Hub](https://img.shields.io/docker/v/miniaxolotl/burrowd?label=dockerhub&color=light)](https://hub.docker.com/r/miniaxolotl/burrowd)
[![License](https://img.shields.io/npm/l/@miniaxolotl/burrowctl)](LICENSE)

Expose local services to the internet via HTTPS subdomains.

```
https://swiftly-ancient-silent-dragon.burrow.mawa.dev → localhost:3000
```

## Quick Start

### 1. Start the server

```bash
docker run -d --name burrowd \
  -p 25701:25701 \
  -e BURROW_SECRET=your-secret \
  -e BURROW_DOMAIN=your-domain.com \
  ghcr.io/miniaxolotl/burrowd:latest
```

Or with Docker Compose:

```bash
git clone https://github.com/miniaxolotl/burrow && cd burrow
cp .env.example .env   # set BURROW_SECRET
docker compose up -d
```

### 2. Install the client

**Global install (recommended):**

```bash
npm install -g @miniaxolotl/burrowctl
pnpm add -g @miniaxolotl/burrowctl
bun add -g @miniaxolotl/burrowctl
```

**One-time use with npx:**

```bash
npx @miniaxolotl/burrowctl auth login <token> --server your-server:25701
npx @miniaxolotl/burrowctl tunnel create --port 3000
```

### 3. Authenticate & create a tunnel

```bash
burrowctl auth login <token> --server your-server:25701
burrowctl tunnel create --port 3000
```

A TUI opens showing your tunnel URL, live status, and request logs.

## Configuration

### Server (`.env`)

| Variable | Description | Default |
|---|---|---|
| `BURROW_SECRET` | Auth secret for token validation | (required) |
| `BURROW_DOMAIN` | Domain for tunnel URLs | `burrow.mawa.dev` |
| `BURROW_PORT` | Server listen port | `25701` |
| `BURROW_REDIS_URL` | Redis connection URL | `redis://redis:6379` |
| `BURROW_TLS` | Generate `https://` URLs | `false` |

### Client

| Variable | Description |
|---|---|
| `BURROW_SERVER` | Server address (`host:port`) |
| `BURROW_TOKEN` | Auth token (stored by `auth login`) |
| `BURROW_SECRET` | Shared secret (auto-generates token) |
| `BURROW_TLS` | Use TLS (`wss://` / `https://`) |

Flags can also be passed directly: `--server`, `--token`, `--secret`, `--domain`, `--tls`.

## TUI

| Key | Action |
|---|---|
| `n` | New tunnel |
| `c` | Copy URL |
| `o` | Open in browser |
| `d` | Close tunnel |
| `l` | Request logs |
| `s` | Cycle sort (port / latency / reconnects) |
| `g` / `G` | Jump to top / bottom |
| `?` | Help |
| `q` | Quit |

## Architecture

- **burrowd** — Go tunnel server (Docker / binary)
- **burrowctl** — Go CLI client distributed as an npm package with pre-built binaries
- **Redis** — Tunnel state persistence

## Docs

- [Deploy & release guide](documentation/deploy/deploy.md)
- [Server setup](documentation/server.md)
- [Client reference](documentation/client.md)

## License

MIT
