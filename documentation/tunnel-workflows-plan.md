# Tunnel workflow development plan

**Status: historical target contract; Linux delivery merged in #84/#88/#90.**
See [implemented checkpoints and limits](tunnel-workflows-implementation.md) for
the actual scope, including deferred macOS capture. The pre-implementation baseline
and original delivery instructions below are retained as historical context, not
current worktree/PR instructions. New operational work has a separate
[development plan](tunnel-operations-plan.md). Prepared against development commit
`552ee32` (0.3.19), refined after source review on 5 November 2026. Commands,
schemas and limits below are proposed contracts, not current CLI instructions.

**Owner-approved platform split:** #90 delivers these workflows with Linux-only
capture/replay. macOS secure capture implementation and native Keychain/package
acceptance are deferred to [draft PR #91](https://github.com/miniaxolotl/burrow/pull/91)
until a macOS environment is available. The macOS gates below are that follow-up's
requirements, not blockers for #90. Ordinary macOS tunneling, aliases, routing,
headless mode and shipped Darwin binaries remain in scope and unchanged.

## Goal and scope

Make Burrow useful in repeatable development and automation workflows without
losing its account-free, one-command start:

1. Run without a TUI and expose a reliable machine-readable contract.
2. Reuse an owned URL after exiting and restarting the CLI.
3. Route one public hostname to multiple local HTTP services.
4. Explicitly capture and replay a request locally, with bounded storage and no
   automatic duplicate side effects.

This PR documents development; it does not implement or deploy these features.
Keep the interactive `burrowctl tunnel 3000` quick start unchanged. No accounts, hosted request archive,
custom domains, arbitrary remote upstreams, daemon installation, automatic webhook
registration, team sharing or deployment-continuity promise in this scope.

## Verified starting point

| Area         | Current behavior and implementation                                                                                                                                                                                      |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Creation     | `burrowctl/cmd/tunnel/create.go` always starts the TUI after creating ports; partial failure still opens it.                                                                                                             |
| Identity     | `burrowctl/internal/client.go` generates a random ID in `CreateTunnel`; reconnect retries that ID only while this client runs. Connections are keyed by local port.                                                      |
| Ownership    | `burrowd/internal/redis.go` permanently reserves owner hashes and revocation tombstones independently of expiring connection metadata. Reconnecting with a known, unrevoked ID already works for its owner.              |
| Replacement  | `burrowd/internal/server.go` replaces a previous same-owner session after a successful handshake. Generation checks protect cleanup, but competing persistent processes could keep replacing each other.                 |
| Forwarding   | The server uses one yamux stream per HTTP request. The client copies stream bytes to one `localhost:<port>`; HTTP parsing/routing does not yet happen there.                                                             |
| Logs         | `protocol/message.go` describes bounded request metadata, not request headers/bodies. Its `StatusCode` field is not populated by the server forwarding path; it must not be treated as an authoritative response status. |
| Isolation    | Management credentials are server-scoped, management redirects are restricted, and public tunnel paths are dispatched before management routes. Preserve those boundaries.                                               |
| Availability | Active sessions live in process memory; deployment currently uses one server instance. A reserved hostname does not serve an app when its client is offline.                                                             |

Sources: [client reference](client.md), [server reference](server.md),
[request-isolation investigation](request-isolation.md), and
[deployment runbook](deploy.md). Existing references describe current behavior;
do not replace them with proposed commands before implementation passes its gates.

## Delivery order and dependencies

**Delivery updated by owner: preserve initial implementation PR #88; #90 contains
the remaining slices with Linux capture, and #91 separately tracks macOS capture.** Use atomic,
incremental commits grouped by related behavior, with M1 → M2 → M3 → M4 as the
recommended development order. Milestones are checkpoints inside that PR, not
separate PRs, branches or partial feature releases. Hard dependencies are narrower:

| Internal milestone / commit group     | Hard dependency | Deliverable and exit gate                                                                                                                    |
| ------------------------------------- | --------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| M0 — reusable fixtures                | None            | Isolated subprocess/Redis harness and baseline behavior tests; no speculative framework rewrite.                                             |
| M1a — runtime and output              | M0              | Headless lifecycle, JSON/JSONL, cancellation and output-contract tests; no server-confirmed readiness claim at this intermediate checkpoint. |
| M1b — registration acknowledgement    | M1a             | Negotiated server acknowledgement after registration, aggregate `ready` and old/new compatibility tests.                                     |
| M2a — persistent attachment admission | M1b             | Server capability, generation fencing and bounded transport recovery; legacy-bypass tests.                                                   |
| M2b — saved aliases                   | M1a, M2a        | Atomic scoped local registry, restart reuse, explicit replacement and close reconciliation.                                                  |
| M3a — HTTP-aware local forwarding     | M0              | Opt-in per-request adapter preserving streaming/upgrades and request isolation. Shared by routing and capture.                               |
| M3b — route configuration             | M1a, M3a        | One hostname, validated local route table and route diagnostics. Works with or without a saved alias.                                        |
| M4a — encrypted local capture         | M1a, M3a        | Opt-in bounded capture and durable local storage/cleanup; privacy and crash tests. Does not require multi-service routing.                   |
| M4b — local replay                    | M4a             | Approval, write-ahead dispatch fencing, persisted outcomes and explicit recovery; no automatic resend of uncertain work.                     |
| M5 — combined acceptance              | M1–M4 complete  | All four features verified together, current docs, focused captures and release readiness evidence before the single PR is ready.            |

Start each behavior with a failing contract/regression test locally, then commit
the passing implementation, tests and directly related documentation together.
Each commit must build and pass its relevant tests; do not publish intentionally
broken intermediate commits. Review the output schema before M1, attachment
protocol before M2, HTTP adapter before M3, and store/key backend before M4;
**do not block starting headless work on choosing a replay database**. The PR
remains draft until every checkpoint and the combined acceptance scenario pass.
Size each spike before scheduling it; there is no calendar estimate here.

### Commit organization within the single implementation PR

The owner's latest instruction supersedes the earlier single-implementation-PR
arrangement: #88 retains the initial checkpoint; #90 targets #88's branch and
contains the remaining Linux capture scope. The owner then separated macOS
capture into draft #91 stacked on #90. Preserve all task worktrees and
retarget/revalidate as parents merge. Suggested focused commits:

1. `test(tunnels): add isolated workflow fixtures`
2. `feat(cli): add cancellable headless output` — lifecycle and versioned encoders.
3. `feat(protocol): acknowledge registered tunnel sessions` — negotiation and compatibility.
4. `feat(tunnels): fence persistent session admission` — server state and downgrade guard.
5. `feat(cli): persist scoped tunnel aliases` — local registry, resume and closure.
6. `refactor(client): add per-request HTTP forwarding` — adapter plus raw-path regressions.
7. `feat(tunnels): route paths to local services` — route parser and dispatcher.
8. `feat(capture): store bounded encrypted requests` — opt-in capture and cleanup.
9. `feat(replay): execute approved local requests durably` — worker and recovery.
10. `test(tunnels): verify combined workflow acceptance` — integration evidence and final doc cross-check.

These are logical boundaries, not a mandatory commit count. Split a large group
into additional atomic commits with like changes; do not mix unrelated features
or create one catch-all implementation commit. Keep matching help/docs with each
contract change, rather than postponing every documentation update to commit 10.
Record the final commit-to-checkpoint mapping in the concise PR description so
reviewers can follow the increments without reading the entire diff at once.

### Changes made during plan review

- Kept delivery in one implementation PR with atomic, related commits; separated
  development order from real dependencies. Routing and capture reuse one
  HTTP adapter but do not require saved aliases or each other.
- Distinguished transport connection, server registration and application health.
- Removed automatic live-session takeover from v1; competing attachments fail
  safely, and legacy clients cannot bypass persistent-mode fencing.
- Defined output/error framing, alias replacement, route matching and replay
  recovery rather than leaving all interfaces to an open-ended design phase.
- Added write-ahead dispatch intent, bounded nonblocking capture and honest local
  expiry/backup limitations; leases alone cannot prevent duplicate HTTP effects.

## M0 — common fixtures and implementation boundaries

- Define a runtime event source shared by TUI and headless consumers, rather than
  scraping terminal output or treating repeated snapshots as new events.
- When M2/M3 need it, introduce an internal connection key independent of port. A named tunnel
  may route to several ports, and separate tunnels may intentionally use one port.
  Preserve port-based convenience APIs until TUI callers migrate.
- Add fixtures with temporary HOME/config directories, fresh owner credentials,
  two synthetic loopback services and disposable real Redis. Preserve fast unit
  tests, but use real Redis for atomic claims, leases, outages and revocation races.
- Keep HTTP request streams wire-compatible. Routing/capture happen on the client
  and do not require a new HTTP preamble or server-side route table. Negotiate only
  actual new server capabilities: registration acknowledgement and persistent
  attachment admission. Unknown required capabilities fail before reservation or
  replacement; optional unsupported capabilities must be reported as unavailable.
  Use a content-free capability discovery read before attempting a required-mode
  connection, and require the new server to enforce it again during admission.
  Discovery failure/old-server 404 never licenses a persistent-mode fallback.
- Validate proposed flags/config and bounded route counts before connecting. Do
  not open tunnels, overwrite profiles or capture data just to display help.
- Keep the existing raw single-port path unchanged until its replacement passes
  compatibility tests. Add no embedded store/keyring dependency to normal tunneling
  until the optional capture subsystem is introduced; no new hosted service.

## M1 — headless mode and structured output

### Proposed interface

```sh
burrowctl tunnel 3000 --headless
burrowctl tunnel 3000 --headless --output jsonl
burrowctl list --output json
burrowctl logs TUNNEL_ID --follow --output jsonl
burrowctl close TUNNEL_ID --output json
```

- `--headless` never starts Bubble Tea, reads keyboard input, opens a browser or
  writes clipboard contents. If stdin or stdout is not a TTY, use headless text
  mode by default; an ordinary interactive terminal retains today's TUI.
- `--output text|json|jsonl`: finite commands support JSON; long-running tunnel
  and follow commands support JSON Lines. Reject `--output json` on a streaming
  command and explain the correct flag. Machine output implies headless mode.
  Finite commands support all three formats; `tunnel`, `logs --follow` and replay
  progress support text/JSONL only. Empty collections stay arrays, not prose placeholders.
- stdout is exclusively the selected data format, including structured errors.
  JSON returns exactly one success/error document; JSONL returns one envelope per
  line, with no second JSON document at exit. Human-mode errors use stderr;
  machine-mode stderr is reserved for unrecoverable encoder/output failures.
  Suppress duplicate Cobra/main errors, ANSI codes, banners and library logging.
- Use an envelope with `schema_version: 1`, `type`, UTC `timestamp`, per-invocation
  monotonic `sequence`, nullable `tunnel_id` and typed `data`. IDs are strings;
  durations use integer milliseconds and sizes integer bytes. Finite success uses
  `type: result`; failure uses `type: error` with `data.code` and a sanitized
  message. Additive fields are compatible; consumers ignore unknown fields/event
  types. Breaking meanings require a new schema version. Never emit owner tokens,
  owner fingerprints, attachment credentials or capture bodies by default.
- Emit `connecting`, `connected`, `reconnecting`, `disconnected`, `failed` and
  `stopped` from actual transport transitions. Include a reason and attempt
  number when relevant. `connected` means transport established, **not** registered
  forwarding, local application health or successful webhook processing.
- In M1b, emit aggregate `ready` only after every requested tunnel has a matching
  server registration acknowledgement. Add `not_ready` immediately when one loses
  that state and a new `ready` after all recover; do not leave an old ready event
  implying a disconnected tunnel is still usable. Include the acknowledged
  generation in events, but never treat an acknowledgement as a health check.
- Headless startup has a 30-second overall connection deadline, also covering
  registration acknowledgement when supported. Legacy-server startup can finish
  transport-only but explicitly lacks the `ready` barrier.
  On partial failure, disconnect this invocation's successful sessions, report
  each failure and exit nonzero. This is **startup rollback, not atomic exposure**:
  a connected tunnel may have been reachable briefly. Do not revoke saved IDs.
  After startup, retry transient disconnects with capped jittered backoff until
  cancellation; report time/attempt/freshness. Terminal rejection of any requested
  tunnel stops the invocation nonzero rather than leaving a partial set running.
- Propose exit codes: 0 successful finite operation/clean signal stop; 1 runtime
  failure; 2 invalid arguments/configuration. Honor SIGINT/SIGTERM with a bounded
  five-second drain, and cancel cleanly on stdout failure/broken pipe instead of
  continuing to expose services invisibly. Propagate command context through
  management HTTP, dialing, stream handlers and worker shutdown; today's detached
  `context.Background()` tunnel lifetime must not survive cancellation. Preserve
  existing interactive behavior initially and document signal exits for scripts.
- Bound output buffering. A slow consumer cannot block forwarding indefinitely;
  on sustained output backpressure, report failure where possible and stop this
  invocation rather than silently dropping lifecycle events. Keep lifecycle output
  separate from droppable display-only metrics; one slow TUI must not stall another
  consumer. Tests must cover a writer blocked without returning an error, not just
  a closed pipe.

### Registration acknowledgement and compatibility

The server currently sends WebSocket `101` **before** Redis registration and
installation of the active session. A URL header or successful yamux constructor
cannot establish readiness. M1b adds a negotiated bounded control exchange after
both steps, carrying the tunnel ID and generation. Keep control data separate from
HTTP request bytes; specify framing, ordering, timeout and maximum size in the
protocol commit group. Never send control frames to clients that did not negotiate them.

New headless clients on old servers may report `connected` with
`registration_confirmed: false`, but must not synthesize `ready`; automation that
needs that barrier checks for the capability/ready event and fails on its own
deadline. Stable aliases require their stronger M2 capability and fail closed
without it. No automatic HTTP probe: it could trigger application side effects.

Illustrative JSONL envelope (sample IDs/timestamp, not a live tunnel):

```json
{
  "schema_version": 1,
  "type": "connected",
  "timestamp": "2026-11-05T00:00:00Z",
  "sequence": 1,
  "tunnel_id": "sample-tunnel",
  "data": { "registration_confirmed": false }
}
```

### Work and acceptance

Primary surfaces: `burrowctl/main.go`, `burrowctl/cmd/root.go`, `burrowctl/cmd/tunnel/`,
`burrowctl/internal/client.go`, `burrowctl/cmd/tui/`, and new shared encoders/event
types. M1b also touches server session admission and `protocol`. Centralize exit
mapping; today's `main.go` always returns exit 1 for errors and must not print a
second human error after machine output.

Test subprocess execution with no TTY, stream parsing, empty lists, log entry-ID
deduplication, partial startup, invalid flags, authorization failures, reconnect
exhaustion/terminal rejection, signals, broken pipes and bounded backpressure.
Reject malformed/unknown output arguments before creating credentials or tunnels.
Pause/fail Redis registration after `101`: no `ready` may be emitted. Drop a
ready session and confirm `not_ready` precedes recovery. Check a 30-second startup
deadline and five-second drain with an injected clock plus bounded subprocess tests.
Assert every stdout line parses in JSONL mode and contains no secrets. Verify a
successful status stream does not mask failed local forwarding. Keep existing
interactive controls and anonymous telemetry opt-out behavior unchanged.

## M2 — stable URLs across CLI restarts

### Proposed interface and identity model

```sh
burrowctl tunnel 3000 --name webhook
# Exit, restart later with the same server, owner and name: same random public URL.
burrowctl tunnel 3000 --name webhook --headless --output jsonl
```

`--name` is a private local alias, **not** a chosen public subdomain. Keep random
public IDs and the existing explicit-close tombstone model. Initially allow one
named connection per command; M3 adds its multiple upstreams. Unnamed commands
keep generating fresh URLs, avoiding accidental reuse between unrelated projects
that happen to use port 3000.

1. Add a versioned private registry under the existing Burrow configuration
   directory, scoped by canonical server origin, owner-credential fingerprint and
   local name. Do not store the token or infer the owner from a public URL. An
   explicit different token/server must not reuse another scope's mapping. Names
   are scoped to the owner/server, not the working directory; bound their length
   and character set and encode them as data, never raw filesystem paths. File
   operations must reject unsafe symlinks and verify private ownership/permissions.
2. Persist a candidate ID atomically **before** the remote claim; reconcile the
   same candidate after a crash/uncertain response. Use locking, private directory
   permissions (0700), file permissions (0600), atomic replacement and corruption
   detection. Fail with a recovery message rather than discarding corrupt state.
3. Reconnect through the existing owner-reservation checks. Expiry of connection
   metadata is not expiry of ownership. Stop on unauthorized, conflict or revoked
   results; never silently choose another ID for an established alias.
4. Prevent concurrent use locally with an OS-held lock keyed to the scoped alias.
   A PID file alone is not sufficient. M2a must implement the attachment contract
   below before M2b enables reuse; do not rely on the current same-owner replacement.
5. Graceful exit disconnects; `close ID` remains permanent revocation. Mark a
   matching local alias closed after confirmed success; if closed elsewhere,
   persist that state on the next rejected reconnect. An uncertain close is shown
   as uncertain and reconciled before reuse. Replacing an alias with a new URL is
   an explicit action with an old/new URL warning; do not resurrect revoked IDs.
   Proposed replacement command: `burrowctl tunnel 3000 --name webhook --replace`.
   Require a confirmed closed alias and explicit confirmation (`--yes` headlessly);
   replacement allocates a new ID rather than clearing its server tombstone.
   Add a bounded owner-authenticated reservation-status read for that reconciliation:
   active-only listing cannot establish whether an offline ID is revoked. Return
   no reservation details to another owner and never expose a global index. Use
   one identical not-found/unauthorized shape for an absent or another owner's ID;
   an outage is not evidence that an ID is unowned or closed.
6. Port/route changes for a saved alias require explicit configuration input and
   show the resulting target. Do not derive a new exposure from background
   discovery, current working directory changes or unrelated local processes.

### Persistent attachment contract (M2a)

- Durable ownership and the right to drive a **live connection** are different.
  Mark an ID as persistent-mode in its non-expiring owner record. Legacy connect
  attempts for that ID must be rejected, even from the same owner; otherwise an
  old reconnect loop bypasses the new admission rule. Unnamed legacy IDs retain
  today's behavior. Enabling persistent mode requires negotiated support.
- Persist a private logical attachment identity and last acknowledged generation
  in the local alias. Acquire its OS lock before any claim. A reconnect performs
  an atomic compare-and-swap from that expected generation; it never reads a newer
  generation after conflict merely to force a takeover. A replaced/stale process
  exits terminally. Credential/config copying is not an additional security
  boundary: anyone given those private credentials can act as that owner.
- Persist a unique attempt nonce before admission. Permanent owner/mode reservation
  may precede the transport handshake, but the server atomically commits the active
  attachment, generation and nonce only after the transport is prepared;
  retrying an uncertain attempt reconciles that same outcome rather than advancing
  the generation again. Register-then-ack and crash-after-register tests are required.
- Reuse the existing **single-instance** session registry and lifecycle lock to
  reject admission while a persistent ID has a live transport, including from the
  same attachment. Do not add a second distributed session-leasing system or
  increase Redis refreshes to every few seconds for this feature. Negotiate bounded
  transport liveness (proposed 10-second heartbeat/30-second stale-transport limit),
  distinct from the 24-hour metadata TTL. Normal process exit closes the transport;
  crash/network-partition recovery may wait for that liveness bound.
- After confirmed disconnect, admit only a conditional advance from the saved
  generation. A `busy` result may retry within the startup deadline; a stale
  generation result is terminal and cannot be turned into takeover by fetching
  the latest generation. Server restart has no surviving transports; reconcile
  durable ownership/generation/nonce from Redis without waiting for metadata expiry.
  The old generation cannot refresh, reconnect or clear the new one.
- Extend existing generation checks on register, refresh, disconnect and cleanup.
  Keep Redis authoritative for owner/revocation/generation and the single active
  server authoritative for live transport presence. The source/deployment gate
  must continue to enforce that single-instance assumption; multi-instance
  admission/routing would need a separate design. Cancelling an old request cannot
  undo effects it already caused.

This is a design contract to test, not a request to build multi-instance routing.
It preserves current single-instance deployment. The capability must not be
advertised until **all** connection paths for persistent IDs enforce it.

Primary surfaces: `burrowctl/internal/config/`, client session creation,
`burrowd/internal/{server,redis,tunnel}.go`, negotiated shared types, list/close
commands and TUI identity handling. Keep active-only `list` semantics; add a
separate `burrowctl saved --output json` local-alias listing rather than conflating
offline profiles with active connections. Show saved/closed/uncertain local state
and its last observation time; do not claim it is a live server snapshot.

### Acceptance

Restart a real CLI subprocess after the 20-second disconnect grace period and
after connection metadata expiry; its URL and owner must remain unchanged. Also
test server restart with retained Redis, crash between local write/remote claim,
disk failure, simultaneous first creation, stale lock recovery, new owner/server,
cross-device attachment conflict, legacy-client bypass, half-open transport expiry, uncertain
admission, closure while offline and stale generation cleanup. Never wait 24 hours
to reuse a crashed client's alias: require recovery within the documented transport
liveness bound, or return a typed busy error without changing the URL. Losing the owner credential cannot be repaired by a
saved alias; show recovery guidance, never bypass authorization. No offline
serving, guaranteed availability or new retention/eviction policy is implied.

## M3 — one hostname, multiple local services

### Proposed interface and route rules

```sh
burrowctl tunnel --name shop --config ./burrow.json --headless --output jsonl
# Omitting --name also works; it creates a fresh random URL.
burrowctl tunnel --config ./burrow.json --headless
```

```json
{
  "version": 1,
  "routes": [
    { "path": "/api", "port": 8080 },
    { "path": "/", "port": 3000 }
  ]
}
```

This is optional, explicit route configuration, not a general application
orchestrator. No token, captured body or private owner identity belongs in this
shareable file. Require `--config` explicitly; do not execute or auto-load files
from the current directory. Reject unknown fields, duplicate JSON keys, unsupported
versions, files over 64 KiB and combinations of positional ports with `routes`.
The first version contains only `version` and `routes`; do not invent overlapping
server/token settings or merge route arrays from flags, environment and files.

- Keep the route table on the client. Forward one public tunnel to a client-side
  HTTP dispatcher, which selects loopback upstreams. The server must not dial
  user-selected targets or become an arbitrary-network proxy.
  The first route's port supplies the existing mandatory `port` metadata during
  connection; it is informational, not an instruction to bypass the dispatcher.
  A routed tunnel counts as one connection in the server's list. Show the complete
  route table from local configuration, never infer it from that single port.
- Match the longest **path-segment** prefix: `/api` matches `/api` and `/api/x`,
  not `/apiculture`. Configured prefixes are `/` or absolute paths without query,
  fragment, backslash, percent escapes, non-ASCII characters, dot segments, repeated slashes or trailing
  slash. Duplicate prefixes are invalid; declaration order never breaks ties.
- Match the request's escaped path without cleaning or redirecting it. Reject
  malformed escapes, encoded slash/backslash, NUL, dot segments (including encoded
  dots), literal backslashes and repeated slashes with 400 in routed mode. Do not
  recursively decode. Preserve safe encoded bytes and a trailing request slash.
  Single-port raw mode retains its existing behavior; this routing policy is not
  a claim to normalize every application's URL semantics.
- Preserve the original path, raw query and body by default. Prefix removal is
  an explicit `strip_prefix: true` option: `/api` maps to `/`, `/api/` maps to `/`
  and `/api/x` maps to `/x`; `/` with stripping is invalid. Rewrite the escaped
  path and parsed path consistently and preserve the raw query. Do not
  silently rewrite redirects, cookie paths, Origin or application authorization.
- Require valid ports, absolute paths and a bounded route count. Propose 32
  routes per tunnel, named or unnamed. The only target field is `port` (1–65535):
  dial fixed `127.0.0.1`/`::1`, not DNS-resolved arbitrary hostnames or a caller's
  Host/absolute-form URL. Preserve the current localhost IPv4/IPv6 compatibility
  with a bounded loopback-only dial strategy. No proxy-from-environment transport.
- Preserve public Host by default; allow an explicit bounded per-route
  `host_header` string if needed by development servers (authority only, no URL,
  CR/LF, path or credentials). Sanitize hop-by-hop headers while preserving a
  validated negotiated Upgrade. Forwarding headers must come from a documented
  trusted source: never promote public client input into an asserted trusted
  identity/scheme. Do not inject or reflect Burrow management credentials.
- An unmatched path returns 404 unless `/` is configured. A failed upstream
  returns a route-specific 502 without falling through to another application.
- Stream request/response bodies with bounded memory and cancellation. Preserve
  WebSocket upgrades, SSE, chunked responses, trailers and upload backpressure.
  Do not hijack ordinary keep-alive connections: keep each request/host isolated.
  Implement the adapter as a private yamux-stream-backed HTTP listener/dispatcher,
  not a public localhost management port. Bound headers, concurrent streams and
  upstream deadlines; do not impose an overall short deadline on valid SSE or
  upgraded sessions. Overflow fails explicitly without falling back to raw TCP.
  Admit one HTTP request per stream (or one negotiated upgrade lifetime), not a
  second pipelined request or arbitrary CONNECT tunnel.
- Load and validate configuration before exposure; do not hot-reload in v1.
  Surface the active route table in TUI/text output and typed readiness data.
  A stable URL does not make cookies isolated between apps: explain shared-origin
  trust and application-level cookie/base-path/CORS configuration.

Primary surfaces: client stream handling and new route/config parser, TUI route
summary and machine-output schema. Reuse a tested HTTP forwarding layer for both
single-port and routed behavior where possible; retain compatibility tests for
the existing raw stream path before changing its default.

### Acceptance

Run two marker services behind one hostname. Verify root/API routing, exact and
nested matches, unmatched paths, encoded-path rejection, prefix removal, body and
query integrity, Host policy, status/error propagation and a stopped upstream.
Exercise WebSocket echo/HMR, SSE cancellation, large uploads and bounded memory.
Extend existing request-isolation tests with keep-alive requests across different
hosts, owners and management-looking paths (`/health`, `/tunnels`, `/tunnels/x/logs`).
Restart with the same alias/routes and prove the URL is unchanged. A malformed
route file must expose nothing and leave the prior saved mapping intact.

Minimum route decision fixtures for the example above:

| Request path                                                          | Expected behavior                                                              |
| --------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| `/api`, `/api/`, `/api/orders?limit=1`                                | Port 8080; preserve path/query unless that route explicitly strips its prefix. |
| `/apiculture`, `/API`, `/%61pi/orders`                                | Port 3000 via `/`; prefix matching is escaped and case-sensitive.              |
| `/api%2forders`, `/api/../admin`, `/api/%2e%2e/admin`, `/api//orders` | 400; never normalize and route to a different service.                         |
| `/other` with no configured `/`                                       | 404, not the first route as a fallback.                                        |

## M4 — opt-in local capture and request replay

### Proposed interface and capture contract

```sh
burrowctl tunnel 3000 --name webhook --capture
burrowctl requests list --name webhook --output json
burrowctl replay REQUEST_ID --name webhook
burrowctl replay REQUEST_ID --name webhook --yes --output jsonl
burrowctl replay status JOB_ID --output json
burrowctl replay retry JOB_ID --yes --output jsonl
burrowctl requests delete REQUEST_ID --name webhook --yes
burrowctl requests clear --name webhook --yes
```

Replay uses **new local capture records**, not historical server log IDs. Capture
starts only after explicit opt-in for that process; restarting without `--capture`
does not silently resume recording. It never uploads bodies/headers to Redis,
the Burrow server's management API, analytics or diagnostic providers.
Capture also works without a saved alias: scope records to server/owner/tunnel ID,
and accept `--tunnel ID` instead of `--name` in request/replay commands. Selectors
are mutually exclusive. Replay uses private local data; an online management call
is not required and a public URL alone never grants access to captures.

- Capture at the HTTP-aware client dispatcher before route rewriting, recording
  the selected route/version and approved local target. Replay uses that captured
  mapping, not whatever an edited route file now points to. Show the current
  approved target and configuration digest in confirmation; v1 rejects config
  drift rather than introducing target overrides. Never replay through the public
  tunnel URL. A port can be reused by another app after restart: matching a port
  or profile does not prove application identity; the operator must verify it.
- Propose initial limits: 1 MiB request body, 32 KiB headers, 200 records, 50 MiB
  total capture storage per server/owner scope and a 24-hour TTL, whichever is
  reached first. Review these defaults before M4a. Cap metadata/queues as well as
  bodies; no full-body buffering before enforcing limits.
  Evict expired/oldest unreferenced captures first. Pending replay payloads count
  toward the same quota; if there is no safe space, drop the new capture with a
  visible reason rather than exceed the cap or silently corrupt a queued replay.
- Capture adds bounded copying/CPU, not a zero-overhead promise. Use a nonblocking,
  size-limited tee into a bounded writer queue; disk/keyring I/O must not block
  forwarding. A full/failed capture writer
  marks that request unavailable with a reason; incomplete, truncated, expired
  or dropped bodies are **not replayable**. Skip upgrades, indefinite streams
  and non-identity Content-Encoding in v1 rather than pretending to record them.
  Publish a replayable record only after the complete body, encrypted payload and
  metadata commit durably. An interrupted spool is incomplete, not accepted work;
  startup cleanup reclaims it without claiming its request was captured.
- Strip authorization, cookies, proxy credentials and known secret headers before
  durable storage; redact sensitive query keys. Body secrets cannot be reliably
  detected: warn that opt-in bodies can contain secrets, support explicit
  exclusions and use synthetic data in examples. If redaction changes request
  meaning or signatures, label that loss; never present replay as byte-identical.
  Do not silently rehydrate stripped secrets from process environment, application
  configuration or a subsequent request. List output exposes IDs, sizes, sanitized
  summaries and replayability reasons, not raw headers/query values/bodies.
- Encrypted sensitive capture storage is mandatory, not just 0600 permissions.
  Before M4a, select an embedded transactional local store (proposed SQLite) and a
  maintained Go driver/keyring integration; this is a new optional local
  persistence component, not a second hosted service. Store keys using supported
  OS credential facilities. If no secure backend exists in a headless session,
  keep ordinary tunneling usable but fail capture clearly; no plaintext fallback
  or prompt requesting a secret. Initialize lazily only for capture/replay, not
  first-run tunneling. Encrypt temporary spools, sensitive metadata, queued payloads
  and result/error content too; raw values must not leak into SQLite WAL/journals,
  paths or crash reports. Authenticate scope/schema/record identity with ciphertext
  to reject cross-record swaps. Test the packaged binary on Linux with available,
  locked and absent credential facilities in #90; the equivalent macOS native
  gates belong to #91. Capture support is a separate compatibility row from
  ordinary headless tunneling.
- Supply scoped list/delete/clear controls and run expiry/cleanup through durable
  local work. Deletion immediately tombstones records, prevents future use and
  removes encrypted payloads/key material according to a documented retention
  policy. Recheck expiry on every read/dispatch; no daemon means physical cleanup
  can occur only while a CLI worker runs or at its next startup. State that limit:
  24 hours is an access TTL, not guaranteed wall-clock deletion while the machine
  is off. A stale writer/retry/worker cannot revive tombstoned content. Deletion
  cannot undo an already-dispatched request; cancel it best-effort and retain only
  a content-free terminal/uncertain job record.
- Default backups/exports of captures are out of scope. Document SQLite WAL/free
  pages, key retention and user-managed backup limits; do not promise forensic or
  cryptographic erasure without proving the full key/copy lifecycle. A full old
  backup can predate tombstones, so backup restoration is unsupported for replay
  in v1, not something a local tombstone can magically prevent.

### Durable execution and side effects

Burrow currently has Redis for server ownership, but no durable local job stack.
Add only the embedded local work store required for capture/replay; do not claim
existing forwarding goroutines or timers provide durable job acceptance.

1. Confirm the method, sanitized path, loopback target, capture age, redaction and
   duplicate-effect warning for **every** replay (GET can mutate too). Without a
   TTY, require `--yes`; that flag approves this selected record/target, not later
   route drift. Never regenerate signatures or fabricate an idempotency key.
2. Atomically commit the command, encrypted payload reference and durable work row
   before emitting `queued`. The embedded work table itself can be the queue: no
   separate broker/second outbox service is needed. Queue opaque IDs, not payloads.
   Failed commit means no acceptance and no network effect.
   Resolve the existing server/owner scope without creating a fresh credential
   during list/status/replay. A missing credential or ambiguous record is an error,
   not permission to search other owners or infer authorization from the URL.
3. The replay CLI runs a foreground bounded worker for that job. It does not need
   an installed daemon or live public tunnel. A normal tunnel restart may process
   capture cleanup but must not silently dispatch old replay jobs. After interruption,
   inspect persisted status and use explicit `replay retry` for a linked new job.
   Retrying must atomically fence/cancel the old job before dispatching the new one;
   an active old worker is a conflict, not permission for two sends.
4. Claim work with an atomic lease/generation. Propose one active replay per
   server/owner scope, at most 20 queued jobs, a 30-second absolute job deadline and
   five-second shutdown drain. Persist retry/backoff/recovery states for pre-send
   preparation failures, with at most three preparation attempts; expiry is
   terminal. Leases use the transactional store's time authority and must exceed
   the dispatch deadline. Never hold a transaction during network I/O.
5. **Before any HTTP client can write**, atomically persist `dispatch_intent` under
   the current lease after checking scope, capture TTL/tombstone, approval, target
   and deadline. Once this marker exists, recovery **never automatically dispatches
   that job again**, even if a crash happened before the first byte was actually
   sent. A conservative false-uncertain outcome is safer than duplicate effects.
   Lease expiry alone does not make an HTTP side effect safe to repeat.
6. Use a dedicated loopback-only transport with no redirects, environment proxies,
   cookies, credential injection or automatic network retries (including Go's
   transparent retries of idempotent requests on reused connections). Reconstruct
   framing/Content-Length from the stored body; do not replay hop-by-hop/framing
   headers or CONNECT/upgrades. Use a fresh connection per attempt and test that
   an upstream disconnect cannot produce a second request.
7. Persist `queued`, `processing`, `retry_wait`, `recovering`, `failed`, `uncertain`,
   `cancelled` and `completed`, plus timestamps, safe error code and HTTP status
   when observed. `completed` means response headers/status were durably observed,
   including 4xx/5xx; it does not mean application processing succeeded. Bound and
   discard response bodies by default; do not add a sensitive response archive.
   A lost result after dispatch is `uncertain`. HTTP status and transport/job
   outcome are separate fields; an observed 500 is not a transport retry trigger.
   The CLI exits nonzero for failed/uncertain/cancelled work and for observed
   non-2xx HTTP responses, while preserving the distinct job and HTTP fields.
8. Recheck scope, tombstone and lease before committing a result. Deletion before
   dispatch intent prevents a send; after intent, cancellation is best-effort and
   may not prevent the effect. A stale worker cannot publish payloads or overwrite
   a new generation's state. Status reads are safe; explicit resend of uncertain
   work requires another warning/approval and creates a linked new job, never a
   misleading transition back to pristine `queued` on the original.

Minimum crash-recovery table:

| Durable evidence                     | Recovery action                                                                                                                              |
| ------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------- |
| No committed job                     | No acceptance, no send.                                                                                                                      |
| Committed job, no dispatch intent    | The invoking worker may recover preparation within its deadline; after interruption require explicit recovery, not unrelated tunnel startup. |
| Dispatch intent, no durable response | Mark uncertain; **never** recover by sending automatically.                                                                                  |
| Durable response/status              | Read the completed result; do not repeat the request.                                                                                        |
| Capture tombstone/expiry             | Cancel unsent work; preserve only content-free status for already-dispatched/uncertain work.                                                 |

Use the same durable state in TUI and JSONL. Reserve an inline feedback area;
preserve row identity/order, filters, focus, scroll and draft confirmation across
updates. Replay results are distinct from live traffic and never recursively
captured. Do not expand the content-free telemetry contract to request content.
Bound content-free job/tombstone retention too (proposed seven days), retaining
them at least through all relevant deadlines and leases; cleanup cannot make an
expired job eligible to execute again.

### Acceptance

Replay synthetic JSON/binary requests against a loopback recorder and compare
method, approved headers, path/query, body and route selection. Test redaction,
expired signatures, disabled/missing capture, exact limits, slow uploads,
unsupported streams, encrypted-at-rest inspection, missing keyring, wrong owner,
route changes, redirects and private-network target rejection.

Crash/restart workers before dispatch, during dispatch and after receipt; test
duplicate publication, lease expiry, storage outage/full disk, competing workers,
erase-during-capture and erase-during-replay. An uncertain POST must never resend
automatically. Check persisted status through CLI restart; a deleted record must
not return. Test content/credentials cannot enter telemetry or default JSON output.

## M5 — combined verification and documentation

### Compatibility and rollback gates

| Client/server or storage combination                   | Required result                                                                                                                        |
| ------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------- |
| Old CLI + new server, ordinary unnamed ID              | Existing raw tunneling and owner-only management continue to work.                                                                     |
| Old CLI + new server, persistent-mode ID               | Reject without replacing the active session, even for the same owner.                                                                  |
| New CLI + old server, unnamed/headless/routed          | Existing wire path works; report registration acknowledgement unavailable, never invent `ready`.                                       |
| New CLI + old server, `--name`                         | Capability preflight fails before creating/resuming a persistent reservation; no random-URL fallback.                                  |
| New CLI + new server                                   | Test registration ack, saved aliases, fencing and route/capture combinations with feature flags both on and off.                       |
| Capture on Linux, usable/locked/missing secure backend | #90: encrypted capture succeeds only with a supported usable backend; ordinary tunneling remains available otherwise.                  |
| Capture on macOS, usable/locked/missing secure backend | Deferred to #91: native adapter/acceptance required before enabling support; #90 rejects capture without affecting ordinary tunneling. |
| Older CLI opens a newer saved/capture schema           | Refuse that feature without rewriting the file; ordinary unrelated tunneling remains usable.                                           |

Once an ID has persistent-mode state, **do not roll back to a pre-fencing server**:
that binary ignores the new mode and could reintroduce takeover. The implementation PR
must add an admission/rollout guard for this downgrade before enabling persistent
IDs. Keep a tested compatible last-good build or use a reviewed forward fix; do
not erase mode flags, owner reservations or tombstones to permit rollback. This is
an explicit development deliverable, not an assertion that today's deployment
pipeline already knows this state version.

### Required local gates for the implementation PR

Use the repository-pinned Go 1.26 and pnpm versions. Run `go work sync` if module
dependencies change and review its diff. From the repository root:

```sh
go vet ./burrowd/... ./burrowctl/... ./protocol/...
go test ./burrowd/... ./burrowctl/... ./protocol/...
go test -race ./burrowd/... ./burrowctl/... ./protocol/...
go build -o bin/burrowctl ./burrowctl
go build -o bin/burrowd ./burrowd
```

Run new integration/subprocess suites with disposable services and isolated HOME.
Only run live public tunnel tests with explicit authorization for that target.
If website examples change, run website lint, typecheck, production build and
browser tests; if release tooling changes, run
`node --test scripts/release/*.test.mjs`. Preserve exact-head CI classification
and `CI Gate`; a skipped unselected job is not an executed test result.

Combined scenario: start a named routed tunnel headlessly, observe truthful
readiness, send a synthetic request with capture enabled, stop/restart beyond
disconnect grace, reuse the URL, replay locally with explicit approval, interrupt
a replay to verify uncertainty, close the tunnel, and prove it cannot resume.
Measure actual time to transport readiness and first valid local response
separately; do not claim the setup target or forwarding performance without
observed evidence. Capture-disabled forwarding must remain usable throughout
capture/replay failures.

### Documentation and stale-information checklist

| File / surface                           | Update when the corresponding feature is verified                                                                                                                                           |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `README.md`                              | Add a short workflow overview and working commands; retain the one-port quick start.                                                                                                        |
| `documentation/client.md`                | Document flags, JSON/JSONL schemas, exits/signals, alias scope, offline status, route config, capture limits and replay approval/recovery.                                                  |
| `documentation/server.md`                | Explain persistent attachment capabilities, generation fencing, compatibility and unchanged revocation/ownership retention; do not imply HTTP status logging is fixed unless tested.        |
| `documentation/workflows.md`             | Add restart-safe webhook and same-origin frontend/API examples plus synthetic local replay with signature/idempotency caveats.                                                              |
| `documentation/product-observability.md` | Reconfirm capture data never enters provider reporting and add content-free outcomes only if needed.                                                                                        |
| `documentation/deploy.md`                | Record compatible client/server rollout and rollback limits; do not imply stable URLs fix single-instance deployment continuity.                                                            |
| Website reference content and CLI help   | Check `packages/website/src/content/documentation.ts`, its source Markdown and feature copy against actual commands and limits; keep proposed features out of live marketing until shipped. |

Search for stale statements about always opening the manager, fresh URLs on every
start, one port per hostname, four total commands, metadata-only local inspection,
and the absence of local capture. Keep **server metadata logs** distinct from
**opt-in local capture**; local capture does not change server log retention.
Update examples and tests in the same focused commit as each changed contract.
This planning PR only links this proposal; existing current-behavior docs remain
valid and are not prematurely rewritten.

### PR evidence, rollout and cleanup

- Use a task worktree under `~/Development/bramble-labs/worktrees/burrow` (task
  suffix when occupied) and `plan.md`, `notes.md`, `status.md` under
  `~/Development/bramble-labs/plans/burrow/<task>`. Preserve other worktrees/work.
- The single implementation PR targets `development`, has a concise conventional title/description,
  is assigned to `miniaxolotl` and has one relevant label. Record tests and
  unresolved constraints honestly; use no Copilot.
- Only save screenshots for relevant visual/TUI changes, using synthetic data.
  Keep captures and explanatory evidence in `documentation/`; embed only selected
  previews in the PR, at most four per row, and link to the rest. This nonvisual
  planning PR needs no screenshots.
- Before merge, require current-head `CI Gate`, no conflicts and resolved review
  conversations. Do not merge or deploy merely because planning is complete.
- Client/server capability changes require an explicit compatibility matrix and
  rollback test before release. Stage server capability support before enabling
  dependent clients; test new/old combinations and explicit unsupported errors.
  Preserve reservation/tombstone data and versioned local state on rollback; an
  older client must reject unknown local schema rather than overwrite it.
- Follow [release conventions](release-conventions.md) and the existing frozen
  production-promotion flow. Do not hand-bump versions, publish artifacts, create
  paid services or change live infrastructure as part of development acceptance.
- After confirmed merge, remove only that task's clean stale worktree. Retain
  unmerged or dirty worktrees and record the reason rather than forcing removal.

## Remaining review gates and risks

The defaults above are concrete proposal decisions, not shipped contracts. Change
them with a recorded reason and corresponding fixtures, rather than leaving basic
behavior to individual implementers. Only these implementation spikes remain:

| Before slice | Required review artifact / stop condition                                                                                                                                                                   |
| ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| M1b          | Bounded negotiated control framing and registration-failure trace. Stop if old peers could interpret control data as HTTP.                                                                                  |
| M2a          | Attachment CAS/nonce/transport-liveness state table, crash traces and downgrade guard design. Stop if a legacy/stale connection can bypass fencing or an uncertain handshake can replace twice.             |
| M3a          | Per-request stream-backed adapter spike with WebSocket/SSE, isolation and bounded-resource evidence. Do not replace the raw default based on unit parsing tests alone.                                      |
| M4a          | #90: Linux store/driver/key-backend matrix, license/build/package review, encrypted WAL/spool inspection and deletion policy. #91: equivalent macOS adapter and native verification. No plaintext fallback. |
| M4b          | Crash and duplicate-effect matrix proving no post-intent automatic redispatch, including transport retries. Leases and a mocked happy path are insufficient evidence.                                       |

Record measured setup time and forwarding overhead against the same disposable
baseline with capture off/on, including oversized/slow bodies and a stalled disk.
Choose a regression budget after that baseline exists, before enabling the new
adapter by default; do not invent an unmeasured speed claim. Headless mode and
stable URLs must remain usable when optional capture lacks a secure key backend;
that does not waive #90's Linux capture compatibility/failure gates or #91's
separate native macOS requirements.

Public vanity domains, remote upstreams, live-session takeover, capture backups,
plaintext capture and automatically repeating uncertain requests stay out of
scope. Adding them requires a separate explicit decision, not an implementation
assumption or another onboarding step.
