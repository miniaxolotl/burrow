# Burrowctl - Client Documentation

## Overview

`burrowctl` is the client CLI that creates tunnels from local ports to the burrow server, exposing them as public URLs.

## Configuration

Configuration is loaded from (in order of precedence):
1. Command-line flags
2. Environment variables
3. `~/.config/burrow/client.json` (written by `burrowctl auth login`)
4. `~/.burrow/token` (legacy fallback, written by `burrowd auth login`)
5. Generate from `--secret` / `BURROW_SECRET`

### Environment Variables

| Variable | Description |
|----------|-------------|
| `BURROW_SERVER` | Server address (host:port) |
| `BURROW_TOKEN` | Authentication token |
| `BURROW_SECRET` | Shared secret (auto-generates token if token not set) |
| `BURROW_TLS` | Use TLS (`wss://` and `https://`) when connecting to the server |
| `BURROW_DOMAIN` | Fallback domain for tunnel URL display |

### Config File

`~/.config/burrow/client.json` — created on first run. Fields: `server`, `token`, `secret`, `domain`, `tls`.

## Commands

### `burrowctl tunnel create`

Create tunnels for one or more local ports. After tunnels are established, launches the interactive TUI.

```bash
burrowctl tunnel create --port 3000 --port 8080
```

**Flags:**
- `--port` - Local port to tunnel (can be specified multiple times, required)
- `--server` - Burrow server address (default: `localhost:25701`)
- `--token` - Authentication token
- `--secret` - Shared secret (auto-generates token)
- `--tls` - Use TLS (`wss://` and `https://`) when connecting to the server
- `--domain` - Fallback domain for tunnel URL display

**Behavior:**
- Connects to the burrow server via WebSocket
- Authenticates with token
- For each port, opens a tunnel and registers it
- Launches the interactive TUI — all tunnel management happens there
- TLS is auto-enabled for non-localhost servers (no `--tls` flag needed for production)
- Press `q` or `Ctrl+C` in the TUI to close all tunnels and exit

**TUI controls:**

| Key | Action |
|-----|--------|
| `n` | New tunnel (prompts for port) |
| `c` | Copy selected tunnel URL to clipboard |
| `o` | Open selected tunnel URL in browser |
| `d` | Close (delete) selected tunnel |
| `l` | View request logs for selected tunnel |
| `↑↓` / `j k` | Navigate tunnel list |
| `esc` | Back (from log view or port input) |
| `q` / `Ctrl+C` | Quit |

**TUI features:**
- Real-time tunnel list updated every second (tunnel ID, port, latency, reconnect count)
- Latency tracked in real-time via `/health` ping every second
- Spinner during tunnel creation
- Request log view: time, method, path, response size, duration
- Tunnel list sorted by port (stable order across refreshes)
- Cursor tracks selected tunnel ID across list refreshes
- Dark background with keyboard-driven navigation

### `burrowctl tunnel list`

List active tunnels managed by this client.

```bash
burrowctl tunnel list
```

**Example output:**
```
TUNNEL ID                       PORT     URL                                             STATUS
swiftly-ancient-silent-dragon  3000     https://swiftly-ancient-silent-dragon.burrow.mawa.dev   active
darkly-shadow-umbral-wraith    8080     https://darkly-shadow-umbral-wraith.burrow.mawa.dev     active
```

### `burrowctl tunnel status`

Show connection status for all tunnels. Same output as `list` but with a separator line beneath the header.

```bash
burrowctl tunnel status
```

**Note:** Neither `list` nor `status` shows latency or reconnect count — those are only visible in the TUI.

### `burrowctl tunnel inspect`

View tunnel traffic logs. **Note: This CLI command is a stub.** Use the TUI (`l` key) for request logs.

```bash
burrowctl tunnel inspect {tunnel_id}
```

### `burrowctl tunnel close`

Close a specific tunnel.

```bash
burrowctl tunnel close {tunnel_id}
```

### `burrowctl auth login`

Store an authentication token for future use.

```bash
burrowctl auth login your-hmac-token
burrowctl auth login your-hmac-token --server burrow.example.com:25701
```

With `--server`, validates the token against the server after storing. Token saved to `~/.config/burrow/client.json`.

### `burrowctl auth logout`

Remove stored authentication token.

```bash
burrowctl auth logout
```

### `burrowctl auth status`

Show current authentication status.

```bash
burrowctl auth status
```

## Tunnel ID Generation

Tunnel IDs are generated using a fantasy/D&D-themed format:

```
{adverb}-{adjective}-{adjective}-{noun}
```

Examples:
- `swiftly-ancient-silent-dragon`
- `darkly-shadow-umbral-wraith`

Generated with `crypto/rand`. Word lists: 100 adverbs, 144 adjectives, 139 nouns. ~288M combinations.

## Reconnection Behavior

The client reconnects on connection loss with exponential backoff (1s doubling to 30s cap). After 5 consecutive failures, a new tunnel ID is generated.

When the client disconnects, the server keeps the tunnel registered for 20 seconds before removing it. This allows the client to reconnect and restore the tunnel without a URL change.

## Examples

### Basic Usage

```bash
# Tunnel a local web server
burrowctl tunnel create --port 3000

# Tunnel multiple services
burrowctl tunnel create --port 3000 --port 8080 --port 5432

# Use specific server
burrowctl tunnel create --port 3000 --server burrow.example.com:25701 --token my-token
```

### Using Authentication

```bash
# Store token once
burrowctl auth login my-token --server burrow.example.com:25701

# Then just specify ports
burrowctl tunnel create --port 3000 --port 8080
```

### Cleanup

```bash
# Close specific tunnel
burrowctl tunnel close swiftly-ancient-silent-dragon
```
