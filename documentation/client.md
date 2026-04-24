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
| `BURROW_SECRET` | Shared secret (auto-generates token if token not set) |
| `BURROW_TLS` | Use TLS (`wss://` and `https://`) when connecting to the server |
| `BURROW_DOMAIN` | Fallback domain for tunnel URL display |

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
- `--secret` - Shared secret (auto-generates token)
- `--tls` - Use TLS (`wss://` and `https://`) when connecting to the server
- `--domain` - Fallback domain for tunnel URL display

**Example:**
```bash
burrowctl tunnel create --port 3000 --port 8080 --server burrow.example.com:25701 --token my-token
```

**Output:**
```
Creating tunnels to burrow.example.com:25701...

Tunnels established:
    https://swiftly-ancient-silent-dragon.burrow.mawa.dev -> localhost:3000
    https://darkly-shadow-umbral-wraith.burrow.mawa.dev -> localhost:8080

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
TUNNEL ID                               PORT    URL
swiftly-ancient-silent-dragon          3000    https://swiftly-ancient-silent-dragon.burrow.mawa.dev
darkly-shadow-umbral-wraith            8080    https://darkly-shadow-umbral-wraith.burrow.mawa.dev
```

### `burrowctl tunnel status`

Show connection status and latency for all tunnels.

```bash
burrowctl tunnel status
```

**Example output:**
```
TUNNEL ID                               PORT    LATENCY    RECONNECTS
swiftly-ancient-silent-dragon          3000    12ms       0
darkly-shadow-umbral-wraith            8080    8ms        1
```

### `burrowctl tunnel inspect`

View detailed tunnel traffic logs, requests, and responses.

```bash
burrowctl tunnel inspect {tunnel_id}
```

**Example:**
```bash
burrowctl tunnel inspect swiftly-ancient-silent-dragon
```

**Flags:**
- `--tail` - Number of recent entries to show (default: 50)
- `--follow` - Stream logs in real-time (like `tail -f`)

**Example output:**
```
[10:30:15] Incoming request:
  Method: GET
  URL: /
  Headers: Host=swiftly-ancient-silent-dragon.burrow.mawa.dev

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
{adverb}-{adjective}-{adjective}-{noun}
```

Examples:
- `swiftly-ancient-silent-dragon`
- `darkly-shadow-umbral-wraith`
- `keenly-mighty-radiant-phoenix`
- `ghostly-silent-ethereal-void`

Generated with `crypto/rand` for uniqueness. Word lists:
- **Adverbs** (100): arcaneily, blindly, boldly, brightly, calmly, chaotically, clearly, coldly, covertly, cruelly, cryptically, darkly, dauntlessly, deeply, deftly, dimly, distantly, divinely, dreadfully, dryly, eerily, eldritchly, endlessly, eternally, evilly, faintly, fearlessly, fiercely, firmly, forebodingly, freely, frostily, fully, ghostly, ghoulishly, gravelely, grimly, hauntingly, harshly, heavily, hellishly, hollowly, icily, infernally, keenly, lethally, lightly, liminally, lowly, magically, malevolently, menacingly, mercilessly, mutely, mystically, nimbly, nobly, obscurely, ominously, openly, perilously, phantomly, proudly, quietly, rapidly, rarely, relentlessly, roughly, ruinously, savagely, sharply, silently, sinisterly, slowly, softly, solemnly, solidly, spectrally, starkly, stealthily, sternly, stolidly, strongly, subtly, swiftly, terribly, thinly, treacherously, truly, undyingly, unholy, vastly, vengefully, vividly, voraciously, wickedly, wildly, wisely, wrathfully, wryly
- **Adjectives** (144): abyssal, accursed, ancient, arcane, ashen, astral, banished, battered, bewitched, bleak, blighted, bloodied, bold, bonded, brave, broken, burning, celestial, chaotic, charmed, chromatic, cold, corrupted, crimson, cryptic, cursed, dark, dead, deathly, defiled, demonic, destined, diabolical, distant, divine, doomed, draconic, dread, druidic, dry, dwarven, dying, elder, eldritch, elven, empty, enchanted, ethereal, exalted, fallen, feral, fierce, fiendish, flaming, forbidden, forgotten, forsaken, foul, frozen, furtive, ghostly, gilded, glowing, grim, hallowed, haunted, hellish, heretical, hidden, hollow, holy, hungry, icy, infernal, iron, jade, keen, legendary, lethal, liquid, lost, luminous, lurking, mad, malevolent, mighty, molten, moonlit, mournful, murky, mystic, necrotic, noble, obscure, ominous, pale, petrified, phantom, plagued, potent, primal, profane, quick, radiant, raging, ruined, runic, sacred, savage, scarlet, scorched, sepulchral, shadow, shattered, silent, silver, sinister, skeletal, smoldering, spectral, stark, still, stone, storming, sunken, swift, tainted, terrible, twilight, twisted, umbral, uncanny, unholy, unseen, veiled, vengeful, vivid, volatile, wandering, wicked, wild, withered, wrathful, wretched
- **Nouns** (139): altar, amulet, anvil, arch, archmage, artefact, assassin, axe, banshee, basilisk, beacon, behemoth, blade, blight, bones, bramble, catacomb, centaur, chains, chimera, cipher, citadel, crypt, curse, cyclops, dagger, demon, depths, dirge, dragon, druid, dungeon, effigy, ember, enchantment, exile, familiar, fiend, forge, fortress, gargoyle, gate, ghost, ghoul, giant, goblin, golem, grave, grimoire, guardian, harbinger, haven, helm, heretic, hex, hydra, idol, illusion, inferno, isle, jailer, kraken, labyrinth, lair, lance, leviathan, lich, longbow, manticore, mausoleum, maze, minotaur, mithril, monolith, moon, necromancer, nexus, nightmare, nymph, obsidian, ogre, oracle, orc, overlord, paladin, phantom, phoenix, plague, portal, prism, prophet, quill, ravine, reaper, relic, revenant, rune, sanctum, sarcophagus, scroll, sentinel, serpent, shade, shard, shrine, siege, skeleton, skull, specter, spell, spire, staff, stalker, stronghold, sword, talisman, throne, tomb, tome, tower, troll, unicorn, urn, vampire, vault, vestige, void, vortex, warden, warlock, wasteland, witch, wizard, wraith, wyvern, xorn, zealot, zephyr, zombie

## Reconnection Behavior

The client reconnects on connection loss with exponential backoff (1s doubling to 30s cap).

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

### Using Environment Variables

```bash
export BURROW_SERVER=burrow.example.com:25701
export BURROW_TOKEN=my-secret-token

# Now just specify ports
burrowctl tunnel create --port 3000 --port 8080
```

### Monitoring

```bash
# Monitor tunnel traffic in real-time
burrowctl tunnel inspect swiftly-ancient-silent-dragon --follow

# Check connection health
burrowctl tunnel status
```

### Cleanup

```bash
# Close specific tunnel
burrowctl tunnel close swiftly-ancient-silent-dragon
```
