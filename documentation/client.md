# burrowctl

Share local services over HTTPS. No account, login, or manual credential setup.

```bash
npm install -g @miniaxolotl/burrowctl
burrowctl tunnel 3000
```

Native Linux/macOS binaries are also available from [GitHub Releases](https://github.com/miniaxolotl/burrow/releases).

## Commands

```bash
burrowctl tunnel 3000 8080          # Share ports and open the interactive manager
burrowctl list                     # Your active tunnels and count
burrowctl logs <id> --follow        # Your tunnel's request logs
burrowctl close <id>                # Close your tunnel permanently
```

`logs --tail N` limits the output. All commands support `--server <host[:port]>`, for example:

```bash
burrowctl tunnel 3000 --server staging.brmbl.dev
```

There are no privileged users, global lists, or admin commands. Each command sees only the tunnels belonging to its owner credential. `list` works from a second terminal using the same server and local configuration.

## Automatic ownership

The first tunnel command generates a random 256-bit owner credential for that server. It is stored under `~/.config/burrow/owners/<server-hash>.token`, with directory permissions 0700 and file permissions 0600. Concurrent first launches share one identity. Different servers receive different credentials; returning to a server reuses its identity.

Keep this directory private and back it up to retain access to your sessions. Sharing a tunnel URL does not share its credential or grant access to its logs, counts, reconnection, or closure. The exposed application's contents remain public unless that application requires authentication.

## Configuration

Default settings live in `~/.config/burrow/client.json`. Flags and environment variables override them.

| Flag       | Environment     | Purpose                                          |
| ---------- | --------------- | ------------------------------------------------ |
| `--server` | `BURROW_SERVER` | Server address; defaults to `brmbl.dev`          |
| `--token`  | `BURROW_TOKEN`  | Optional explicit owner credential               |
| `--domain` | `BURROW_DOMAIN` | Fallback domain if an older server omits the URL |

A saved explicit token is used only for its configured server, never forwarded to a different `--server` override. Management requests reject cross-origin redirects. There is no shared server secret or credential with elevated privileges.

TLS is required for remote servers. `localhost`, `127.0.0.1`, and `::1` use HTTP for local development.

## Interactive controls

| Key            | Action                         |
| -------------- | ------------------------------ |
| `n`            | Add a port                     |
| `c` / `o`      | Copy / open the selected URL   |
| `d`            | Close the selected tunnel      |
| `l`            | View logs / return to the list |
| `s`            | Change sorting                 |
| `f` / `r`      | Follow / refresh logs          |
| `?`            | Help                           |
| `q` / `Ctrl+C` | Exit                           |

## Reconnection and upgrades

Transient connection failures retry the same URL with exponential backoff while the CLI is running. Request logs survive short disconnects for 20 seconds. Ownership stays reserved after connection records expire, so another owner cannot claim a previously shared URL. Explicit closure revokes the ID; create a new tunnel for a fresh URL. Authorization failures, ownership conflicts, and revocation stop automatic reconnection.

Upgrade the client and server together for the owner-only API. Older tokenless clients must upgrade and create new tunnels; legacy anonymous IDs are retired. Existing explicit tokens can still identify their owner, but never grant privileged access. Use the four commands above in place of the old nested `tunnel` subcommands. The old `auth` commands and `--secret` flag are removed.

See the [server API](server.md) and [deployment guide](deploy.md).
