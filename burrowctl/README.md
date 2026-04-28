# burrowctl

[![npm](https://img.shields.io/npm/v/@miniaxolotl/burrowctl)](https://npmjs.com/package/@miniaxolotl/burrowctl)
[![License](https://img.shields.io/github/license/miniaxolotl/burrow)](LICENSE)

CLI client for [burrow](https://github.com/miniaxolotl/burrow). Expose local services to the internet via HTTPS subdomains.

## Install

```bash
npm i -g @miniaxolotl/burrowctl
```

Requires Go 1.21+ if building from source:
```bash
go install github.com/miniaxolotl/burrow/burrowctl@latest
```

## Quick start

```bash
burrowctl auth login <token>
burrowctl tunnel create --port 3000
```

No account required for the default server. Connect to your local service at `https://<tunnel-id>.burrow.mawa.dev`.

## Commands

### Authentication

```
burrowctl auth login <token>       Store an authentication token
burrowctl auth logout              Remove stored token
burrowctl auth status              Check authentication status
burrowctl auth set-server <addr>   Set burrow server address
```

### Tunnels

```
burrowctl tunnel create --port <port> [--port <port> ...]
                                   Create tunnels for specified ports

burrowctl tunnel list              List active tunnels
burrowctl tunnel status            Show tunnel connection status
burrowctl tunnel inspect <id>      View tunnel traffic logs
                                   Flags: --tail N, --follow
burrowctl tunnel close <id>        Close a specific tunnel
```

### Flags

```
--server <address>   Burrow server address (default: burrow.mawa.dev)
--token <token>      Authentication token
--secret <secret>    Shared secret (generates a token via HMAC-SHA256)
--domain <domain>    Domain for tunnel URLs (default: burrow.mawa.dev)
--port <port>        Local port to tunnel (can be specified multiple times)
```

### Environment variables

All flags map to `BURROW_*` env vars. E.g., `--token` → `BURROW_TOKEN`.

### TLS

TLS is auto-enabled for any server other than `localhost`, `127.0.0.1`, or `::1`. There is no configuration option to disable TLS for remote servers.

## Examples

Create a tunnel to local port 3000:
```bash
burrowctl tunnel create --port 3000
```

Create multiple tunnels:
```bash
burrowctl tunnel create --port 3000 --port 8080 --port 5432
```

View logs for a tunnel:
```bash
burrowctl tunnel inspect <tunnel-id> --tail 20 --follow
```

Use a custom server:
```bash
burrowctl auth set-server burrow.example.com
burrowctl auth login <token>
burrowctl tunnel create --port 3000
```

## Config

Config is stored at `~/.config/burrow/client.json`.

```json
{
  "server": "burrow.mawa.dev",
  "token": "",
  "secret": "",
  "domain": "burrow.mawa.dev"
}
```