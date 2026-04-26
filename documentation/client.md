# burrowctl

CLI client that creates tunnels from local ports to the burrow server, exposing them as public URLs.

## Install

```bash
# npm
npm install -g @miniaxolotl/burrowctl

# Or download binaries from GitHub Releases
# Linux/macOS: amd64, arm64
```

## Quick Start

```bash
burrowctl auth login <token> --server burrow.example.com:25701
burrowctl tunnel create --port 3000
```

## Configuration

| Variable | Description |
| -------- | ----------- |
| `BURROW_SERVER` | Server address (host:port) |
| `BURROW_TOKEN` | Auth token |
| `BURROW_SECRET` | Shared secret (auto-generates token) |
| `BURROW_TLS` | Use TLS (`wss://` / `https://`) |
| `BURROW_DOMAIN` | Fallback domain for URL display |

Config stored in `~/.config/burrow/client.json` after first `auth login`.

## Commands

```bash
burrowctl tunnel create --port 3000 --port 8080   # Create tunnels + open TUI
burrowctl tunnel list                              # List active tunnels
burrowctl tunnel status                            # Connection status
burrowctl tunnel inspect {id} [--follow] [--tail N] # View logs
burrowctl tunnel close {id}                        # Close a tunnel
burrowctl auth login <token> [--server addr]       # Store token
burrowctl auth logout                              # Remove token
burrowctl auth status                              # Check auth status
```

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

Exponential backoff (1s → 30s cap). After 5 failures, a new tunnel ID is generated. Server keeps tunnels for 20s after disconnect to allow reconnection without URL change.

## Documentation

- [Server reference](server.md)
- [Deploy & release](deploy.md)
