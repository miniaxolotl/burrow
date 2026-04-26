# burrow

[![npm](https://img.shields.io/npm/v/@miniaxolotl/burrowctl)](https://npmjs.com/package/@miniaxolotl/burrowctl)
[![GHCR](https://img.shields.io/github/v/release/miniaxolotl/burrow?label=ghcr&color=light)](https://github.com/miniaxolotl/burrow/pkgs/container/burrowd)
[![Docker Hub](https://img.shields.io/docker/v/miniaxolotl/burrowd?label=dockerhub&color=light)](https://hub.docker.com/r/miniaxolotl/burrowd)

Expose local services to the internet via HTTPS subdomains.

## Server

```bash
cp .env.example .env && docker compose up -d
```

## Client

```bash
npm i -g @miniaxolotl/burrowctl
burrowctl auth login <token> --server your-server:25701
burrowctl tunnel create --port 3000
```

## Docs

- [Deploy & release](documentation/deploy.md)
- [Server setup](documentation/server.md)
- [Client reference](documentation/client.md)

MIT
