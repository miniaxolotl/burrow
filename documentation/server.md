# burrowd

Owner-isolated HTTP/WebSocket tunneling server. Every credential is a regular owner; there is no privileged access model.

## Run

```bash
cp .env.example .env              # Set BURROW_DOMAIN
docker compose up -d
```

Or run the binary with Redis available:

```bash
burrowd serve --domain localhost --port 25701 --redis-url localhost:6379
burrowd stop
```

| Variable           | Purpose                                | Default          |
| ------------------ | -------------------------------------- | ---------------- |
| `BURROW_DOMAIN`    | Tunnel URL domain                      | `localhost`      |
| `BURROW_PORT`      | Listen port (`PORT` fallback)          | `25701`          |
| `BURROW_REDIS_URL` | Redis URL (`REDIS_URL` fallback)       | `localhost:6379` |
| `BURROW_TLS`       | Generate HTTPS URLs behind a TLS proxy | `false`          |

Remote deployments require TLS termination and a wildcard certificate. Production uses `brmbl.dev`; staging uses `staging.brmbl.dev`. See [deployment](deploy.md).

## Five API routes

All management requests require `X-Tunnel-Token: <owner-credential>`. The CLI creates this credential automatically; no account is needed. URL query credentials are rejected.

| Method    | Route                                         | Access / result                                |
| --------- | --------------------------------------------- | ---------------------------------------------- |
| GET       | `/health`                                     | Public; only `{"status":"ok"}`                 |
| GET       | `/tunnels`                                    | Owner's connected tunnels and count            |
| WebSocket | `/tunnels/connect?tunnel_id=<id>&port=<port>` | Create or reconnect an owned tunnel            |
| GET       | `/tunnels/<id>/logs`                          | Owner's request history, up to 200 entries     |
| DELETE    | `/tunnels/<id>`                               | Owner-only closure and permanent ID revocation |

The list response is `{"tunnels":[...],"count":N}`. Both fields use the same owner-filtered snapshot. Disconnected grace-period records are not counted. There is no global count/list endpoint, no `all` scope, and no uptime disclosure. Former admin tokens have no special meaning. Private responses use `Cache-Control: no-store` and never contain owner credentials or hashes.

Traffic to `https://<id>.<domain>` goes to the exposed application. Its authentication is independent of Burrow management access. A shareable URL is never proof of ownership.

## Ownership lifecycle

- Redis atomically binds each ID to an owner hash. A different owner cannot replace the connection, claim its ID, read its logs, or close it.
- Each connection has a generation identifier. Old disconnect timers and keepalive work cannot remove or renew a newer connection.
- Disconnects retain connection metadata/logs for 20 seconds. Active records have a refreshed 24-hour TTL.
- `tunnel-owner:<id>` reservations retain the owner hash and closure state without a TTL, preventing old URLs from becoming another user's tunnel. Preserve these records and use Redis **noeviction**; storage grows with allocated IDs.
- Closing revokes the ID and clears its logs. The client stops reconnecting rather than silently recreating the closed session.

## Upgrading

Coordinate client/server updates: the consolidated routes and list response replace the previous API. Missing owner credentials are rejected; the updated CLI creates them automatically. Startup reserves existing token-bearing records for their recorded owners and permanently retires existing tokenless IDs before serving requests. Old anonymous clients must upgrade and create new URLs. Historical IDs already absent from Redis at upgrade time cannot be recovered or reserved from missing data.

`BURROW_SECRET`, HMAC admin authentication, and the old `burrowd auth` / `burrowd tunnel` commands are removed. There is no application-level override to access another owner's data. The server process and hosting infrastructure still handle the proxied HTTP traffic; this is owner isolation, not end-to-end encryption from the host.
