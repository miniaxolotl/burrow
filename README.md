# burrow

[![npm](https://img.shields.io/npm/v/@miniaxolotl/burrowctl)](https://npmjs.com/package/@miniaxolotl/burrowctl)
[![GHCR](https://img.shields.io/github/v/release/miniaxolotl/burrow?label=ghcr&color=light)](https://github.com/miniaxolotl/burrow/pkgs/container/burrowd)
[![Docker Hub](https://img.shields.io/docker/v/miniaxolotl/burrowd?label=dockerhub&color=light)](https://hub.docker.com/r/miniaxolotl/burrowd)
[![License](https://img.shields.io/github/license/miniaxolotl/burrow)](LICENSE)

Expose local services to the internet via HTTPS subdomains.

## Client

```bash
npm i -g @miniaxolotl/burrowctl
burrowctl tunnel create --port 3000
```

Connects to `burrow.mawa.dev` by default. No account or token required.

## Self-hosting

```bash
cp .env.example .env   # set BURROW_SECRET and BURROW_DOMAIN
docker compose up -d
```

Then point your client at it:

```bash
burrowctl auth set-server your-server.example.com
burrowctl auth login <token>
burrowctl tunnel create --port 3000
```

### Tunnel Manager

<img src=".assets/ui_main.png" width="280"> <img src=".assets/ui_logs.png" width="280"> <img src=".assets/ui_help.png" width="280">

## Docs

- [Deploy & release](documentation/deploy.md)
- [Server reference](documentation/server.md)
- [Client reference](documentation/client.md)

MIT
