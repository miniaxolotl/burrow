# burrow

[![npm](https://img.shields.io/npm/v/@miniaxolotl/burrowctl)](https://npmjs.com/package/@miniaxolotl/burrowctl)
[![GHCR](https://img.shields.io/github/v/release/miniaxolotl/burrow?label=ghcr&color=light)](https://github.com/miniaxolotl/burrow/pkgs/container/burrowd)
[![Docker Hub](https://img.shields.io/docker/v/miniaxolotl/burrowd?label=dockerhub&color=light)](https://hub.docker.com/r/miniaxolotl/burrowd)
[![License](https://img.shields.io/npm/l/@miniaxolotl/burrowctl)](LICENSE)

Expose local services to the internet via HTTPS subdomains.

```
https://swiftly-ancient-silent-dragon.burrow.mawa.dev → localhost:3000
```

## Server

```bash
git clone https://github.com/miniaxolotl/burrow && cd burrow
cp .env.example .env   # set BURROW_SECRET
docker compose up -d
```

## Client

```bash
# npm
npm install -g @miniaxolotl/burrowctl

# pnpm
pnpm add -g @miniaxolotl/burrowctl

# bun
bun add -g @miniaxolotl/burrowctl
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

- [Deployment guide](documentation/deploy/deploy.md)
- [Client CLI reference](documentation/client.md)
- [Server setup & deployment](documentation/server.md)

## License

MIT
