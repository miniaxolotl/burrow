# burrowctl

CLI client that creates tunnels from local ports to the burrow server, exposing them as public URLs.

## Install

```bash
npm install -g @miniaxolotl/burrowctl
```

Or download binaries from [GitHub Releases](https://github.com/miniaxolotl/burrow/releases) (Linux / macOS, amd64 + arm64).

## Quick Start

No account or token required to use the public server:

```bash
burrowctl tunnel create --port 3000
```

Connects to `burrow.mawa.dev` over TLS by default.

## Self-hosting

Point the client at your own server:

```bash
burrowctl auth set-server your-server.example.com
burrowctl auth login <token>            # token generated from your BURROW_SECRET
burrowctl tunnel create --port 3000
```

## Configuration

Settings are stored in `~/.config/burrow/client.json` after first run.

| Variable | Description | Default |
| -------- | ----------- | ------- |
| `BURROW_SERVER` | Server address | `burrow.mawa.dev` |
| `BURROW_TOKEN` | Auth token | — |
| `BURROW_SECRET` | Shared secret (generates a token on the fly) | — |
| `BURROW_TLS` | Force TLS on/off (`true`/`false`) | auto (on for non-localhost) |
| `BURROW_DOMAIN` | Fallback domain for URL display | `burrow.mawa.dev` |

## Commands

### Auth

```bash
burrowctl auth set-server <address>          # Set server (e.g. your-server.example.com)
burrowctl auth login <token> [--server addr] # Store token (and optionally set server)
burrowctl auth logout                         # Remove stored token
burrowctl auth status                         # Show current token and server
```

### Tunnels

```bash
burrowctl tunnel create --port 3000 --port 8080    # Open tunnels + launch TUI
burrowctl tunnel close <id>                         # Close a tunnel by ID
burrowctl tunnel inspect <id> [--follow] [--tail N] # View request logs
```

### Admin (requires token)

```bash
burrowctl tunnel list      # List all active tunnels on the server
burrowctl tunnel status    # Same as list with a separator
```

## Access control

| Operation | Anonymous | With token |
| --------- | --------- | ---------- |
| Create tunnel | ✅ | ✅ |
| Close own tunnel | ✅ (anonymous) | ✅ |
| View own tunnel logs | ✅ (anonymous) | ✅ |
| Close any tunnel | ✗ | ✅ (admin) |
| List all tunnels | ✗ | ✅ (admin) |

Tunnels created with a token are owned by that token — only the same token (or an admin) can close them or view their logs. Tunnels created without a token are anonymous and can be managed by anyone who knows the tunnel ID.

## TUI Controls

| Key | Action |
| --- | ------ |
| `n` | New tunnel |
| `c` | Copy URL |
| `o` | Open URL in browser |
| `d` | Close tunnel |
| `l` | View logs / return to list |
| `s` | Cycle sort (port → latency → reconnects) |
| `f` | Toggle auto-follow in log view |
| `r` | Refresh logs |
| `?` | Help overlay |
| `g`/`G` | Top/bottom |
| `PgUp`/`PgDn` | Page up/down |
| `↑↓` / `j k` | Navigate |
| `q` / `Ctrl+C` | Quit |

## Reconnection

Exponential backoff (1s → 30s cap). After 5 consecutive failures a new tunnel ID is generated. The server holds tunnel registrations for 20 s after disconnect so a quick reconnect preserves the URL.

## Documentation

- [Server reference](server.md)
- [Deploy & release](deploy.md)
