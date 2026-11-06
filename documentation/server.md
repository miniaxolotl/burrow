# burrowd

## Bounded process shutdown

`burrowd serve --drain-timeout 15s` (`BURROW_DRAIN_TIMEOUT`) stops new public
requests and registrations atomically, returning 503 with `Retry-After: 1` and
`Cache-Control: no-store`. Already admitted requests, including SSE and upgrades,
may finish until the deadline. Zero requests immediate interruption; accepted
values are 0–15 seconds. Cleanup gets at most five more seconds within a 20-second
total cap (or the caller's earlier deadline). Concurrent shutdown callers share
the first shutdown outcome; deadline interruption is reported as an error. Burrow
never automatically resends an interrupted request.

`GET /health` remains content-free liveness. `GET /ready` returns 200 only while
admitting traffic and after a bounded successful Redis read (one-second deadline,
one-second single-flight cache); otherwise it returns content-free 503. Tunnel
subdomains still forward `/health` and `/ready` to their local apps. Hosted health
checks remain unchanged pending authorized provider termination-order evidence.

## Operational telemetry

Official server releases can report content-free connection/forwarding outcomes,
sanitized errors, 10% sampled traces, and operation metrics to the same Burrow
providers. Enable with `BURROW_TELEMETRY_ENABLED=true`; the hosted production
service sets this through its infrastructure definition. Self-hosted servers are
off by default. `DO_NOT_TRACK=1` or `BURROW_TELEMETRY=0` disables collection.

The server uses a random identity under its service account's
`~/.config/burrow/telemetry-id`; it never derives identity from a tunnel owner.
It receives no request object or tunnel content in its telemetry layer. Headers,
URLs, credentials, user IPs, ports, request histories and payloads remain outside
provider reporting. The bounded queue and shutdown behavior match the
[client contract](client.md#telemetry-and-privacy). Server diagnostics are an
operator-controlled setting independent of a connecting client's local choice.

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

## API routes

Owner-scoped management requests require `X-Tunnel-Token: <owner-credential>`.
Health and capability discovery are public exceptions. The CLI creates the
credential automatically; no account is needed. URL query credentials are rejected.

| Method    | Route                                         | Access / result                                |
| --------- | --------------------------------------------- | ---------------------------------------------- |
| GET       | `/health`                                     | Public; only `{"status":"ok"}`                 |
| GET       | `/tunnels`                                    | Owner's connected tunnels and count            |
| WebSocket | `/tunnels/connect?tunnel_id=<id>&port=<port>` | Create or reconnect an owned tunnel            |
| GET       | `/tunnels/<id>/logs`                          | Owner's request history, up to 200 entries     |
| DELETE    | `/tunnels/<id>`                               | Owner-only closure and permanent ID revocation |

Development also provides public `GET /.well-known/burrow-capabilities` for
registration/persistent-attachment negotiation and owner-only
`GET /tunnels/<id>/reservation` for saved-alias reconciliation. These accompany
the [merged workflow implementation](tunnel-workflows-implementation.md), not a
claim that every published server already supports them.

The proposed [tunnel operations plan](tunnel-operations-plan.md) adds bounded
draining, optional lifecycle notices and a separate readiness endpoint. None is
implemented by the planning PR. `/health` currently reports only the fixed response
above, not Redis readiness, tunnel registration or local application health.

The list response is `{"tunnels":[...],"count":N}`. Both fields use the same owner-filtered snapshot. Disconnected grace-period records are not counted. There is no global count/list endpoint, no `all` scope, and no uptime disclosure. Former admin tokens have no special meaning. Private responses use `Cache-Control: no-store` and never contain owner credentials or hashes.

Each log entry has an opaque `id` for deduplication. Entries are appended when requests finish; their `timestamp` records the request start, so timestamps need not be ordered. The CLI follows entry IDs to include slow requests that finish after newer ones.

Traffic to `https://<id>.<domain>` goes to the exposed application. Its authentication is independent of Burrow management access. A shareable URL is never proof of ownership.

## Request isolation and usage bounds

Every HTTP request is routed by its current Host, even when an ingress reuses the
same origin connection for different tunnel hosts or the management API. Ordinary
HTTP stays under Go's request dispatcher; one request uses one yamux stream and a
request-scoped HTTP transport. Only a successfully negotiated protocol upgrade
(such as WebSocket) takes over the connection. HTTP/1 keep-alive, HTTP/2 responses,
streaming bodies and WebSocket upgrades are covered by regression tests.

Request history retains **200 completed requests per tunnel** in memory. Usage
counts bytes read from the local service's response stream, including HTTP
headers; it is not bidirectional bandwidth accounting, a billing meter or a
transfer allowance. Counts survive history eviction but reset when tunnel state
is cleaned up. HTTP request bodies are streamed rather than limited by the
**1 MiB transport WebSocket-message** bound: yamux splits larger HTTP payloads
across messages.

There is currently no application-level requests-per-second, transfer-volume or
per-owner tunnel-count quota. This is not an unlimited-capacity guarantee. Hosted
edge, memory, connection, compute and Redis/provider limits still apply. Production
uses one 0.5 GB App Platform instance; infrastructure declares a $20/month Upstash
usage budget with autoscaling disabled. Maximum safe capacity requires measured
acceptance, not inference from these settings. See
[the request-isolation investigation](request-isolation.md) and
[bounded live testing](deploy.md#bounded-live-load-testing).

## Ownership lifecycle

- Redis atomically binds each ID to an owner hash. A different owner cannot replace the connection, claim its ID, read its logs, or close it.
- Each connection has a generation identifier. Old disconnect timers and keepalive work cannot remove or renew a newer connection.
- Disconnects retain connection metadata/logs for 20 seconds. Active records have a refreshed 24-hour TTL.
- `tunnel-owner:<id>` reservations retain the owner hash and closure state without a TTL, preventing old URLs from becoming another user's tunnel. Preserve these records and use Redis **noeviction**; storage grows with allocated IDs.
- Closing revokes the ID and clears its logs. The client stops reconnecting rather than silently recreating the closed session.

## Upgrading

Coordinate client/server updates: the consolidated routes and list response replace the previous API. Missing owner credentials are rejected; the updated CLI creates them automatically. Startup reserves existing token-bearing records for their recorded owners and permanently retires existing tokenless IDs before serving requests. Old anonymous clients must upgrade and create new URLs. Historical IDs already absent from Redis at upgrade time cannot be recovered or reserved from missing data.

`BURROW_SECRET`, HMAC admin authentication, and the old `burrowd auth` / `burrowd tunnel` commands are removed. There is no application-level override to access another owner's data. The server process and hosting infrastructure still handle the proxied HTTP traffic; this is owner isolation, not end-to-end encryption from the host.
