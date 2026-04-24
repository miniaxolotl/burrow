# Burrowctl - Client Documentation

## Overview

`burrowctl` is the client CLI that creates tunnels from local ports to the burrow server, exposing them as public URLs.

## Configuration

Configuration is loaded from (in order of precedence):
1. Command-line flags
2. Environment variables
3. Stored auth token
4. Default values

### Environment Variables

| Variable | Description |
|----------|-------------|
| `BURROW_SERVER` | Server address (host:port) |
| `BURROW_TOKEN` | Authentication token |
| `BURROW_PORT` | Default server port |
| `BURROW_HOSTNAME` | Default server hostname |

### Config File

Default location: `~/.burrowctl/config.yaml`

```yaml
server: localhost:25701
token: your-auth-token
```

## Commands

### `burrowctl tunnel create`

Create tunnels for one or more local ports.

```bash
burrowctl tunnel create --port 3000 --port 8080
```

**Flags:**
- `--port` - Local port to tunnel (can be specified multiple times, required)
- `--server` - Burrow server address (default: `localhost:25701`)
- `--token` - Authentication token

**Example:**
```bash
burrowctl tunnel create --port 3000 --port 8080 --server burrow.example.com:25701 --token my-token
```

**Output:**
```
Creating tunnels to burrow.example.com:25701...

Tunnels established:
  https://arcane-dragon-xorn.inkspire.app -> localhost:3000
  https://shadow-lich-umbra.inkspire.app -> localhost:8080

Press Ctrl+C to close tunnels
```

**Behavior:**
- Connects to the burrow server
- Authenticates with token
- For each port, opens a WebSocket connection
- Generates a human-readable tunnel ID
- Prints the public URL(s)
- Maintains connections and reconnects automatically

### `burrowctl tunnel list`

List active tunnels managed by this client.

```bash
burrowctl tunnel list
```

**Example output:**
```
TUNNEL ID              PORT    URL                                           STATUS
arcane-dragon-xorn     3000    https://arcane-dragon-xorn.inkspire.app       active
shadow-lich-umbra      8080    https://shadow-lich-umbra.inkspire.app       active
```

### `burrowctl tunnel status`

Show connection status and latency for all tunnels.

```bash
burrowctl tunnel status
```

**Example output:**
```
TUNNEL ID              PORT    LATENCY    RECONNECTS
arcane-dragon-xorn     3000    12ms       0
shadow-lich-umbra      8080    8ms        1
```

### `burrowctl tunnel inspect`

View detailed tunnel traffic logs, requests, and responses.

```bash
burrowctl tunnel inspect {tunnel_id}
```

**Example:**
```bash
burrowctl tunnel inspect arcane-dragon-xorn
```

**Flags:**
- `--tail` - Number of recent entries to show (default: 50)
- `--follow` - Stream logs in real-time (like `tail -f`)

**Example output:**
```
[10:30:15] Incoming request:
  Method: GET
  URL: /
  Headers: Host=arcane-dragon-xorn.inkspire.app

[10:30:15] Outgoing response:
  Status: 200 OK
  Headers: Content-Type=text/html
  Body: 52 bytes
```

### `burrowctl tunnel close`

Close a specific tunnel.

```bash
burrowctl tunnel close {tunnel_id}
```

## Authentication

Before creating tunnels, authenticate with the server:

```bash
export BURROW_TOKEN=my-secret-token
burrowctl tunnel create --port 3000
```

Or inline:
```bash
burrowctl tunnel create --port 3000 --token my-secret-token
```

## Tunnel ID Generation

Tunnel IDs are generated using a fantasy/D&D-themed format:

```
{adjective}-{noun}-{creature}
```

Examples:
- `arcane-dragon-xorn`
- `shadow-lich-umbra`
- `ethereal-phoenix-void`
- `mystic-wyrm-zephyr`

Fantasy word lists:
- **Adjectives**: arcane, ancient, astral, bold, brave, chaotic, cryptic, dark, elder, ethereal, fierce, frozen, hidden, icy, jade, keen, liquid, mystic, noble, obscure, potent, quick, radiant, shadow, swift, twilight, uncanny, vivid, wandering, wild
- **Nouns**: amulet, basilisk, cipher, dragon, ember, fortress, gargoyle, helm, illusion, kraken, lich, mithril, nymph, oracle, phoenix, quest, rune, specter, talisman, umbral, void, wyrm, zephyr, amethyst, bramble, crypt, druid, forge, grimoire, haven, isle, knave, lava, moon, nexus, obsidian, prism, quill, shadow, tome, umbra, vestige, warden, xorn, zinc

## Reconnection Behavior

The client automatically reconnects on connection loss with exponential backoff:

1. First retry: 1 second
2. Second retry: 2 seconds
3. Maximum retry: 60 seconds

During reconnection, the tunnel URL remains active (for a limited time).

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

### Using Environment Variables

```bash
export BURROW_SERVER=burrow.example.com:25701
export BURROW_TOKEN=my-secret-token
export BURROW_PORT=25701
export BURROW_HOSTNAME=burrow.example.com

# Now just specify ports
burrowctl tunnel create --port 3000 --port 8080
```

### Monitoring

```bash
# Monitor tunnel traffic in real-time
burrowctl tunnel inspect arcane-dragon-xorn --follow

# Check connection health
burrowctl tunnel status
```

### Cleanup

```bash
# Close specific tunnel
burrowctl tunnel close arcane-dragon-xorn
```
