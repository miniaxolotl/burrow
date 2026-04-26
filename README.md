# burrow

[![npm](https://img.shields.io/npm/v/@miniaxolotl/burrowctl)](https://npmjs.com/package/@miniaxolotl/burrowctl)
[![GHCR](https://img.shields.io/github/v/release/miniaxolotl/burrow?label=ghcr&color=light)](https://github.com/miniaxolotl/burrow/pkgs/container/burrowd)
[![Docker Hub](https://img.shields.io/docker/v/miniaxolotl/burrowd?label=dockerhub&color=light)](https://hub.docker.com/r/miniaxolotl/burrowd)
[![License](https://img.shields.io/npm/l/@miniaxolotl/burrowctl)](LICENSE)

Expose local services to the internet via HTTPS subdomains.

```
https://swiftly-ancient-silent-dragon.burrow.mawa.dev → localhost:3000
```

## Deploy

### Docker

```bash
cp .env.example .env   # edit with registry tokens
pnpm install
pnpm --filter @script/deploy run deploy
```

Registry tokens via env vars:
- `GHCR_REGISTRY=ghcr.io/miniaxolotl` — GitHub Container Registry
- `DOCKERHUB_REGISTRY=miniaxolotl` — Docker Hub

When no registry is set, builds locally without pushing. See `scripts/deploy/README.md` for full options.

### Dokku

See [Server documentation](documentation/server.md#deploy-dokku) for:
- App creation and Redis setup
- Wildcard SSL certificates
- Nginx configuration

### Binaries

Pre-built binaries for Linux and macOS are on the [Releases](https://github.com/miniaxolotl/burrow/releases) page:

```bash
# Linux amd64
curl -fsSL https://github.com/miniaxolotl/burrow/releases/latest/download/burrowd_0.2.0_linux_amd64.tar.gz | tar -xz
sudo mv burrowd /usr/local/bin/

# macOS arm64
curl -fsSL https://github.com/miniaxolotl/burrow/releases/latest/download/burrowd_0.2.0_darwin_arm64.tar.gz | tar -xz
sudo mv burrowd /usr/local/bin/
```

## Server (Docker)

```bash
git clone https://github.com/miniaxolotl/burrow && cd burrow
cp .env.example .env   # set BURROW_SECRET
docker compose up -d
```

## Client (npm)

```bash
npm install -g @miniaxolotl/burrowctl
```

Then:

```bash
burrowctl auth login <token> --server your-server:25701
burrowctl tunnel create --port 3000
```

## Configuration

### Server `.env`

| Variable | Description | Default |
|----------|-------------|---------|
| `BURROW_SECRET` | Authentication secret | (required) |
| `BURROW_DOMAIN` | Domain for tunnel URLs | `mawa.dev` |
| `BURROW_PORT` | Server port | `25701` |
| `BURROW_REDIS_URL` | Redis connection URL | `localhost:6379` |
| `BURROW_TLS` | Generate `https://` URLs | `false` |

### Client environment

| Variable | Description |
|----------|-------------|
| `BURROW_SERVER` | Server address (`host:port`) |
| `BURROW_TOKEN` | Authentication token |
| `BURROW_SECRET` | Shared secret (auto-generates token) |
| `BURROW_TLS` | Use TLS (`wss://`) |

## TUI

| Key | Action |
|-----|--------|
| `n` | New tunnel |
| `c` | Copy URL |
| `o` | Open in browser |
| `d` | Close tunnel |
| `l` | View request logs |
| `q` | Quit |

## Docs

- [Client CLI reference](documentation/client.md)
- [Server setup & deployment](documentation/server.md)

## License

MIT
