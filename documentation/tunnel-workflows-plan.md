# Tunnel workflow development plan

**Status: proposal, not implemented.** Prepared against development commit
`552ee32` (0.3.19), reviewed 5 November 2026. Commands, schemas, limits and new
package names below are proposed contracts, not current CLI instructions.

## Goal and scope

Make Burrow useful in repeatable development and automation workflows without
losing its account-free, one-command start:

1. Run without a TUI and expose a reliable machine-readable contract.
2. Reuse an owned URL after exiting and restarting the CLI.
3. Route one public hostname to multiple local HTTP services.
4. Explicitly capture and replay a request locally, with bounded storage and no
   automatic duplicate side effects.

This PR documents development; it does not implement or deploy these features.
Keep `burrowctl tunnel 3000` unchanged. No accounts, hosted request archive,
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

| Milestone / suggested PR            | Depends on | Deliverable and exit gate                                                                                                     |
| ----------------------------------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------- |
| M0 — contracts and fixtures         | None       | Agree output schemas, identity/concurrency semantics, routing rules, capture policy and disposable integration harness.       |
| M1 — headless and structured output | M0         | Shared runtime events, signals, output encoders and subprocess tests; no TUI required.                                        |
| M2 — saved tunnel identities        | M1         | Scoped atomic local registry; restart reuse and permanent-close behavior; no reconnect takeover loop.                         |
| M3 — one-hostname routing           | M1, M2     | HTTP-aware local forwarding, validated route configuration, upgrade/streaming regression tests.                               |
| M4a — local capture foundation      | M3         | Opt-in bounded capture and encrypted local durable work/storage; privacy and crash tests.                                     |
| M4b — local replay                  | M4a        | Single-request replay, explicit side-effect approval, persisted outcomes and recovery; no automatic resend of uncertain work. |
| M5 — combined acceptance            | M1–M4b     | Cross-feature restart/recovery tests, current docs, focused captures and release readiness evidence.                          |

Use separately reviewable PRs, not one large rewrite. Start each implementation
slice with a failing contract/regression test and add only the code needed for
that slice. M1 can ship independently. Do not release replay until M4a and M4b
both meet their gates. There is no calendar or performance estimate yet; size the
HTTP-forwarding and local-storage spikes in M0 before committing a schedule.

## M0 — common contracts and safety fixtures

- Define a runtime event source shared by TUI and headless consumers, rather than
  scraping terminal output or treating repeated snapshots as new events.
- Introduce a stable internal connection key independent of port. A named tunnel
  may route to several ports, and separate tunnels may intentionally use one port.
  Preserve port-based convenience APIs until TUI callers migrate.
- Add fixtures with temporary HOME/config directories, fresh owner credentials,
  two synthetic loopback services and disposable real Redis. Preserve fast unit
  tests, but use real Redis for atomic claims, leases, outages and revocation races.
- Define capability negotiation before adding a new wire behavior. Prefer the
  existing HTTP-over-yamux stream for routing/capture; do not send a new preamble
  to an old server. Unsupported features must fail explicitly, not silently fall
  back to a fresh URL, partial routing or missing capture.
- Validate proposed flags/config and bounded route counts before connecting. Do
  not open tunnels, overwrite profiles or capture data just to display help.

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
- stdout is exclusively the selected data format. Human diagnostics go to
  stderr; structured errors go through the same versioned encoder. Suppress
  duplicate Cobra errors, ANSI codes, banners and library logging in machine mode.
- Use a versioned envelope with `schema_version`, event type, UTC timestamp,
  opaque connection ID and typed data. Document units, nullability, ordering and
  stable error codes. Never emit owner credentials/hashes or capture bodies.
- Emit `connecting`, `connected`, `reconnecting`, `disconnected`, `failed` and
  `stopped` from actual transport transitions. Include a reason and attempt
  number when relevant. `connected` means transport established, **not** local
  application health or successful webhook processing.
- For multiple tunnels, emit readiness only once every requested connection is
  established. Default headless startup is all-or-nothing: on partial failure,
  disconnect this invocation's successful sessions, report each failure and exit
  nonzero. Disconnect is not permanent revocation; retain saved identities.
- Propose exit codes: 0 successful finite operation/clean signal stop; 1 runtime
  failure; 2 invalid arguments/configuration. Honor SIGINT/SIGTERM with a bounded
  drain, and cancel cleanly on stdout failure/broken pipe instead of continuing
  to expose services invisibly. Preserve existing interactive behavior initially.
- Bound output buffering. A slow consumer cannot block forwarding indefinitely;
  on sustained output backpressure, report failure where possible and stop this
  invocation rather than silently dropping lifecycle events.

### Work and acceptance

Primary surfaces: `burrowctl/cmd/root.go`, `burrowctl/cmd/tunnel/`,
`burrowctl/internal/client.go`, `burrowctl/cmd/tui/`, and new shared encoders/event
types. Extend `protocol` only for actual shared server/client contracts.

Test subprocess execution with no TTY, stream parsing, empty lists, log entry-ID
deduplication, partial startup, invalid flags, authorization failures, reconnect
exhaustion/terminal rejection, signals, broken pipes and bounded backpressure.
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
   explicit different token/server must not reuse another scope's mapping.
2. Persist a candidate ID atomically **before** the remote claim; reconcile the
   same candidate after a crash/uncertain response. Use locking, private directory
   permissions (0700), file permissions (0600), atomic replacement and corruption
   detection. Fail with a recovery message rather than discarding corrupt state.
3. Reconnect through the existing owner-reservation checks. Expiry of connection
   metadata is not expiry of ownership. Stop on unauthorized, conflict or revoked
   results; never silently choose another ID for an established alias.
4. Prevent concurrent use locally with an OS-held lock keyed to the scoped alias.
   A PID file alone is not sufficient. Separately add negotiated server-side
   fencing for persistent clients: a live logical attachment rejects a different
   attachment, while reconnect within that attachment atomically advances its
   generation. A takeover must be explicit and terminally fence the old attachment,
   not start a reconnect replacement loop. Preserve legacy reconnect semantics
   until a coordinated migration is documented; do not claim cross-device safety
   before this server gate exists.
5. Graceful exit disconnects; `close ID` remains permanent revocation. Mark a
   matching local alias closed after confirmed success; if closed elsewhere,
   persist that state on the next rejected reconnect. An uncertain close is shown
   as uncertain and reconciled before reuse. Replacing an alias with a new URL is
   an explicit action with an old/new URL warning; do not resurrect revoked IDs.
   Add a bounded owner-authenticated reservation-status read for that reconciliation:
   active-only listing cannot establish whether an offline ID is revoked. Return
   no reservation details to another owner and never expose a global index.
6. Port/route changes for a saved alias require explicit configuration input and
   show the resulting target. Do not derive a new exposure from background
   discovery, current working directory changes or unrelated local processes.

Primary surfaces: `burrowctl/internal/config/`, client session creation,
`burrowd/internal/{server,redis,tunnel}.go`, negotiated shared types, list/close
commands and TUI identity handling. Keep active-only `list` semantics; add a
separate saved-alias listing rather than conflating offline profiles with active
connections. Define that command's final spelling in M0.

### Acceptance

Restart a real CLI subprocess after the 20-second disconnect grace period and
after connection metadata expiry; its URL and owner must remain unchanged. Also
test server restart with retained Redis, crash between local write/remote claim,
disk failure, simultaneous first creation, stale lock recovery, new owner/server,
cross-device attachment conflict, failed takeover, closure while offline and
stale generation cleanup. Losing the owner credential cannot be repaired by a
saved alias; show recovery guidance, never bypass authorization. No offline
serving, guaranteed availability or new retention/eviction policy is implied.

## M3 — one hostname, multiple local services

### Proposed interface and route rules

```sh
burrowctl tunnel --name shop --config ./burrow.json --headless --output jsonl
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
shareable file. Resolve flags over explicit configuration over safe defaults;
reject ambiguous combinations of positional ports and route configuration.

- Keep the route table on the client. Forward one public tunnel to a client-side
  HTTP dispatcher, which selects loopback upstreams. The server must not dial
  user-selected targets or become an arbitrary-network proxy.
- Match the longest **path-segment** prefix: `/api` matches `/api` and `/api/x`,
  not `/apiculture`. Reject duplicates and ambiguous/unsafe encoded paths.
  Specify escaping, repeated slashes, dot segments and trailing-slash behavior
  in table-driven tests; avoid inconsistent normalization across proxy layers.
- Preserve the original path, raw query and body by default. Prefix removal is
  an explicit per-route option with defined exact-prefix behavior. Do not
  silently rewrite redirects, cookie paths, Origin or application authorization.
- Require valid ports, absolute paths and a bounded route count. Propose 32
  routes per named tunnel. Allow loopback HTTP upstreams only in v1; reject
  arbitrary hostnames, remote IPs and embedded credentials. IPv4/IPv6 loopback
  handling must be explicit and tested.
- Preserve public Host by default; make an upstream Host override explicit and
  bounded if needed by development servers. Sanitize hop-by-hop headers and
  untrusted forwarding metadata; never inject management credentials upstream.
- An unmatched path returns 404 unless `/` is configured. A failed upstream
  returns a route-specific 502 without falling through to another application.
- Stream request/response bodies with bounded memory and cancellation. Preserve
  WebSocket upgrades, SSE, chunked responses, trailers and upload backpressure.
  Do not hijack ordinary keep-alive connections: keep each request/host isolated.
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

## M4 — opt-in local capture and request replay

### Proposed interface and capture contract

```sh
burrowctl tunnel 3000 --name webhook --capture
burrowctl requests --name webhook --output json
burrowctl replay REQUEST_ID --name webhook
burrowctl replay REQUEST_ID --name webhook --yes --output jsonl
```

Replay uses **new local capture records**, not historical server log IDs. Capture
starts only after explicit opt-in for that process; restarting without `--capture`
does not silently resume recording. It never uploads bodies/headers to Redis,
the Burrow server's management API, analytics or diagnostic providers.

- Capture at the HTTP-aware client dispatcher before route rewriting, recording
  the selected route/version and approved local target. Replay uses that captured
  mapping, not whatever an edited route file now points to; require approval for
  a changed target. Never replay through the public tunnel URL.
- Propose initial limits: 1 MiB request body, 32 KiB headers, 200 records, 50 MiB
  total capture storage per server/owner scope and a 24-hour TTL, whichever is
  reached first. Review these defaults in M0. Cap metadata/queues as well as
  bodies; no full-body buffering before enforcing limits.
- Capture must not delay or change forwarding. A full/failed capture writer
  marks that request unavailable with a reason; incomplete, truncated, expired
  or dropped bodies are **not replayable**. Skip upgrades, indefinite streams
  and unsupported body encodings in v1 rather than pretending to record them.
- Strip authorization, cookies, proxy credentials and known secret headers before
  durable storage; redact sensitive query keys. Body secrets cannot be reliably
  detected: warn that opt-in bodies can contain secrets, support explicit
  exclusions and use synthetic data in examples. If redaction changes request
  meaning or signatures, label that loss; never present replay as byte-identical.
- Encrypted sensitive capture storage is mandatory, not just 0600 permissions.
  M0 should select an embedded transactional local store (proposed SQLite) and a
  maintained Go driver/keyring integration; this is a new optional local
  persistence component, not a second hosted service. Store keys using supported
  OS credential facilities. If no secure backend exists in a headless session,
  keep ordinary tunneling usable but fail capture clearly; no plaintext fallback
  or prompt requesting a secret. Document key lifecycle and backup limitations.
- Supply scoped list/delete/clear controls and run expiry/cleanup through durable
  local work. Deletion immediately tombstones records, prevents future use and
  removes encrypted payloads/key material according to a documented retention
  policy. A stale capture writer, retry, restored profile or worker cannot revive
  deleted content. Deletion cannot undo a request already sent to an application.

### Durable execution and side effects

Burrow currently has Redis for server ownership, but no durable local job stack.
Add only the embedded local work store required for capture/replay; do not claim
existing forwarding goroutines or timers provide durable job acceptance.

1. A replay command verifies scope, capture freshness/completeness, target and
   approval, then atomically commits a job plus outbox/work record. Queue only
   opaque work IDs; encrypted payloads remain in authorized local storage.
2. A bounded worker in the running CLI services committed work. A replay command
   may run a foreground worker for its own job, so replay does not require an
   installed daemon or a live public tunnel. No detached promises, timers or
   browser polling as job execution. Stop intake and boundedly drain on shutdown.
3. Use confirmed local publication, idempotent consumption, fenced leases,
   deadlines, bounded concurrency and retry budgets. Recover store interruptions
   and expired leases from durable state, not a successful-looking UI timer.
   Never hold a storage transaction across network I/O.
4. Immediately before dispatch and result commit, recheck owner/server scope,
   capture tombstone/expiry, target and lease ownership. Disallow remote targets,
   proxies inherited from environment, redirects and path/Host tricks that can
   escape the approved loopback destination. Do not invent or refresh signatures.
5. Persist `queued`, `processing`, `retry_wait`, `recovering`, `failed`, `uncertain`
   and `completed` with timestamps, attempts and safe error codes. Queue acceptance
   is not execution. `completed` means an HTTP response was received, including a
   non-2xx response; application/background processing success remains unknown.
6. Any request can have side effects, including GET. Require confirmation for all
   replays; noninteractive mode requires explicit `--yes`. Never automatically
   resend after bytes may have reached the upstream. A timeout/crash after dispatch
   is `uncertain`, even when a response was not persisted. Preserve original
   idempotency semantics; do not fabricate an idempotency key or promise exactly-once
   effects. Retry automatically only when dispatch is proven not to have occurred.
7. Show the persisted job ID/status and allow scoped status inspection/recovery.
   A user-requested resend of uncertain work creates a linked new attempt with
   another explicit approval and duplicate-effect warning. Removing a capture
   cancels unsent work and prevents stale workers from committing new payloads.

Use the same durable state in TUI and JSONL. Reserve an inline feedback area;
preserve row identity/order, filters, focus, scroll and draft confirmation across
updates. Replay results are distinct from live traffic and never recursively
captured. Do not expand the content-free telemetry contract to request content.

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

### Required local gates for implementation PRs

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
Update examples and tests in the same implementation PR as each changed contract.
This planning PR only links this proposal; existing current-behavior docs remain
valid and are not prematurely rewritten.

### PR evidence, rollout and cleanup

- Use a task worktree under `~/Development/bramble-labs/worktrees/burrow` (task
  suffix when occupied) and `plan.md`, `notes.md`, `status.md` under
  `~/Development/bramble-labs/plans/burrow/<task>`. Preserve other worktrees/work.
- Each PR targets `development`, has a concise conventional title/description,
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

## Decisions to settle before implementation

M0 must record review outcomes for the proposed flag spellings/output schema,
saved-alias replacement command, server attachment-fencing migration, exact path
normalization rules, capture limits/redaction exclusions, and embedded store plus
secure key backend. These are engineering review gates, not reasons to introduce
an onboarding wizard. Broader changes such as public vanity domains, remote
upstreams, plaintext capture or automatically repeating uncertain requests require
a separate explicit decision, not an implementation assumption.
