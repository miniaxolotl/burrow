# Tunnel operations development plan

**Proposal only.** Prepared against development `1f18a30` after workflow PRs
#84/#88/#90 and heavy-load PR #87 merged. This planning PR changes no runtime,
protocol, infrastructure or published release. Commands, fields and limits below
are implementation targets, not instructions for the current CLI.

## Outcomes and boundaries

1. **Deployment-aware draining:** stop new admission, let admitted requests finish
   within a deadline, then reconnect with honest interruption reporting.
2. **Local health diagnostics:** distinguish server reachability, registration,
   local TCP reachability and observed application responses without conflating them.
3. **Better traffic inspection:** useful status/error metadata, bounded filtering,
   stable selection and contextual details in CLI/TUI without requiring capture.

Keep the one-command quick start, owner isolation, persistent attachment fencing,
streaming/upgrades and explicit replay approval. No account, setup wizard, daemon,
new hosted service, public diagnostic listener, horizontal scaling, automatic
business-request retries or zero-downtime claim. macOS secure capture remains
separately deferred in [draft #91](https://github.com/miniaxolotl/burrow/pull/91);
these features must work on Linux and macOS without capture or a credential backend.

## Verified baseline

| Area         | Current behavior / source                                                                                                                                                                                                                                    |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Shutdown     | `burrowd/internal/server.go:Shutdown` closes all yamux sessions before `http.Server.Shutdown`. `burrowd/cmd/serve.go` supplies a ten-second context. There is no request drain barrier or deployment notice.                                                 |
| Hosting      | `infra/modules/tunnel-environment/main.tf` declares one instance, `/health` checks, 30-second termination grace and five-second provider drain. Active sessions remain process-local. Provider termination ordering has not been established by this review. |
| Health       | `/health` always returns `{"status":"ok"}`. Client `trackLatency` calls it per tunnel each second, retains old latency after failures and does not check response status. This is not local-app health.                                                      |
| Registration | `protocol/registration.go` reserves the first negotiated server-opened stream for one bounded acknowledgement plus EOF. Client `ready` means aggregate registration, not application health.                                                                 |
| Forwarding   | `burrowctl/internal/client.go` retains raw single-port forwarding; routing/capture use `internal/routing/http.go`, with loopback-only dialing, strict paths and a 128-stream admission bound.                                                                |
| Server logs  | `protocol/message.go` and `server.go` retain 200 completed entries per tunnel, opaque IDs and response-stream usage. HTTP status is not populated; proxy errors may be absent from history. No route knowledge exists on the server.                         |
| Inspection   | `cmd/tunnel/logs.go` supports tail/follow and bounded ID deduplication. TUI logs fetch synchronously inside `Update`, select by index and may follow the bottom even when follow is off. Capture selection already uses a separate identity-preserving path. |
| Replay       | Linux encrypted capture/replay already has durable SQLite work, leases, dispatch intent, erasure fences and uncertain outcomes. Ordinary tunneling does not open that store.                                                                                 |

References: [client](client.md), [server](server.md), [deployment](deploy.md),
[workflow implementation](tunnel-workflows-implementation.md),
[request isolation](request-isolation.md) and [heavy-load evidence](heavy-load-2026-11-05.md).
The earlier [workflow plan](tunnel-workflows-plan.md) retains a historical,
pre-implementation baseline; it is not the current starting point for this work.

## Delivery and shared rules

Use one follow-up implementation PR with focused, passing incremental commits;
this planning PR does not authorize rollout. Do not mix in macOS capture, access
controls, provider replacement or unrelated UI rewrites. Development order is
O0 → O1 → O2 → O3 → O4 → O5; the hard dependencies are:

| Slice                       | Depends on | Deliverable / exit gate                                                                                                                                  |
| --------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| O0 — fixtures and contracts | None       | Isolated owners/HOME, disposable Redis, synthetic loopback HTTP/SSE/WS, controlled request barriers and pinned old/new protocol fixtures.                |
| O1 — server drain           | O0         | Atomic admission barrier, in-flight ownership, bounded shutdown and process readiness; race and cancellation gates.                                      |
| O2 — lifecycle notice       | O1         | Negotiated drain channel, visible reconnect/backoff and compatibility matrix; no early session teardown.                                                 |
| O3 — health                 | O0         | Passive scoped health, freshness and bounded explicit diagnostics; no application probe by default. Can be developed independently of O2.                |
| O4 — inspection             | O0         | Server status/error records, optional local metadata ring, shared bounded query logic, stable CLI/TUI behavior. Can be developed independently of O2/O3. |
| O5 — combined acceptance    | O1–O4      | Real-service/crash/terminal evidence, current docs, release/infra gates and explicitly pending hosted acceptance.                                        |

For each backend change, write a failing behavior test before implementation, then
commit the passing code, relevant tests and docs together. Reuse existing upstream
HTTP, terminal, strict-JSON and output helpers. Do not replace transports or build
a generic job framework to implement these features.

Process-bound tunnel sessions, request counters and live metadata are ephemeral
transport state, **not durable background jobs**. Shutdown cannot be resumed on a
dead process; reconnect establishes a new fenced generation. Bounded interactive
diagnostic reads may remain synchronous outside the TUI reducer. No detached
background scans, scheduled active probes or exports are introduced. If later
requirements need durable work, reuse the existing SQLite work/worker approach
with opaque IDs, scope, fences, deadlines and persisted outcomes; queue acceptance
must not be presented as completion. Never replay an original request on recovery.

## O1 — bounded drain and readiness

### State and admission

Use a process state `starting → serving → draining → stopped`. Draining is
monotonic and idempotent: repeated signals or concurrent `Shutdown` calls share
one completion result/deadline and cannot return premature success. The first
transition atomically disables registration **and** public request admission.
An admitted request acquires its session reference and in-flight slot under the
same lock/barrier. It cannot be admitted between checking state and opening its
yamux stream. Release exactly once on completion, cancellation, panic or upgrade
closure. Avoid holding lifecycle/map locks during network I/O or drain waits.

Proposed shutdown order:

1. Mark draining and reject new registrations before any reservation/generation
   mutation. Requests rejected by this barrier return 503, `Cache-Control: no-store`
   and a bounded `Retry-After`; no request body is forwarded. This is advice to
   callers, **not permission for Burrow to resend**.
2. Keep already admitted sessions and their HTTP/SSE/upgrade streams usable.
   Notify negotiated clients without waiting for a slow client to read. Stop idle
   HTTP reuse where supported, but keep the listener long enough to serve readiness
   and reject late admission explicitly.
3. Wait until the admitted requests finish or the drain deadline expires. A
   negotiated upgrade/SSE counts until it closes, not only until response headers.
   Idle tunnel control connections do not keep the drain alive forever.
4. Close remaining streams and yamux sessions, then finish HTTP/listener shutdown
   within a separate cleanup budget. Go HTTP shutdown alone does not wait for
   hijacked connections; explicitly track those lifetimes and force-close them.
5. Stop intake/maintenance and join owned transport workers. Conditional registry
   cleanup must not remove/refresh a newer generation or revoke an owned URL.
   A Redis outage cannot extend the deadline. Retain reservations/tombstones;
   the next attachment reconciles durable state normally.

Explicit owner closure/revocation still wins during draining and may interrupt
requests immediately. A delayed cleanup, log callback or old drain event cannot
touch a replacement generation. Preserve public-host dispatch before management
paths: a tunnel request for `/health` or `/ready` still goes to its local app.

### Proposed limits and health endpoints

- Add `burrowd serve --drain-timeout 15s` / `BURROW_DRAIN_TIMEOUT`; initially
  accept 0–15 seconds. Zero means explicit immediate drain, not unbounded waiting.
  Reserve five seconds for cleanup; the command's total shutdown cap becomes
  20 seconds, further bounded by a caller's earlier deadline.
- Retain public `/health` as the existing content-free liveness response. Add
  public `GET /ready`: 200 only when serving and a bounded Redis readiness read
  succeeds; otherwise 503. Use at most a one-second Redis read and a single-flight,
  one-second cache; never disclose owners, IDs, counts, versions or Redis errors.
  A failed/expired observation cannot reuse an older success as fresh readiness.
- Change the provider health path only in the infrastructure slice, after proving
  how provider probes, route removal, SIGTERM, grace and force-kill interact. Do
  not assume the declared 30 seconds starts after the five-second provider drain.
  Require a conservative budget covering provider drain, the 20-second process
  cap and a five-second margin; fail configuration validation if that cannot hold.
  If the provider cannot demonstrate sufficient ordering/budget, leave hosted
  activation blocked rather than increasing replicas or inventing continuity.
- Keep exact-image/provider-generation readiness, attachment compatibility floor,
  publication-before-rollout and real HTTPS/WSS smoke checks. `/ready` supplements
  them; an old healthy instance cannot prove the requested generation is active.

**Availability boundary:** one process cannot transfer live streams to its
replacement. New requests can fail while draining/reconnecting, and long streams
can be terminated at deadline. Bounded completion of admitted requests is the
local target; hosted improvement needs separate authorized provider evidence.

## O2 — negotiated deployment notices and reconnect

Do not reuse registration-v1's single-object/EOF stream or send control JSON on
an unmarked HTTP stream. Proposed extension:

- Offer/echo `X-Burrow-Lifecycle: 1` only with registration v1. Advertise an
  optional lifecycle-v1 capability. No echo means existing reconnect behavior,
  with drain awareness explicitly unavailable; never infer planned deployment
  from a generic EOF, 502, 503 or timeout.
- For negotiated peers only, open a **second reserved server-opened yamux stream**
  before publishing the session. Send a bounded hello after registration commit,
  with version, tunnel ID and generation; the client consumes/validates both
  reserved streams before starting HTTP acceptance or claiming readiness.
- Keep that second stream for server-to-client newline-delimited frames, at most
  2 KiB each, one hello plus one terminal `draining` notice per generation. Include
  a fixed reason (`server_shutdown`), remaining budget in milliseconds and a
  generation-scoped notice ID. Do not claim a software deployment unless the
  process actually has that provenance. Bound handshake writes to five seconds
  and notice writes to one second, within the shared drain budget.
- Validate identity, size, version and ordering. Duplicate notices are idempotent;
  an old reader cannot alter a new session. An unexpected channel failure makes
  lifecycle state unknown and closes that negotiated session into ordinary
  reconnect, rather than leaking control bytes into the local app. A notice-write
  failure cannot block other sessions or imply delivery was acknowledged.
  Use bounded concurrent notice writes within the common shutdown deadline,
  never serial one-second waits per tunnel or an unbounded goroutine fan-out.

On a valid notice, emit `draining`, invalidate aggregate `ready` with
`not_ready(reason=server_draining)` and show the budget as an estimate. Registration
remains a historical fact, but the process no longer admits new traffic. **Do not
immediately reconnect or cancel existing forwarding**: wait for server closure or
the negotiated bounded deadline. Then use existing capped jittered backoff,
original tunnel ID and persistent nonce/generation reconciliation. New readiness
requires fresh registration and admission on every tunnel. Passive app failures
do not invalidate this transport barrier.

Keep revoked/unauthorized/stale-attachment failures terminal; shutdown does not
authorize takeover or bypass busy admission. No preemptive parallel connection,
random-ID fallback or local request resend. Abrupt crash/lost notice stays
`transport_lost`, never a fabricated graceful deployment.

Use additive lifecycle events/fields under CLI envelope v1, documenting the
expanded readiness invalidation reasons. Emit actual observation timestamps,
attempt and retry delay. Render bounded error codes, not remote bodies. If output
backpressure prevents truthful state, retain the current fail-visible/stop policy.

Compatibility gate: old client/new server receives no lifecycle stream; new
client/old server stays transport/registration-only as supported; new/new receives
the negotiated channel. Persistent mode always requires its existing fencing
capability, regardless of optional lifecycle support.

## O3 — local health without invented readiness

### Independent observations

Maintain separate scoped records rather than one green `healthy` boolean:

| Dimension              | Meaning / source                                                                                                                             |
| ---------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Server reachability    | Last successful/failed bounded apex read, status and time. Not proof the tunnel or local app works.                                          |
| Registration/admission | Acknowledged tunnel/generation and serving/draining/reconnecting state. Unknown on legacy peers.                                             |
| Local TCP              | Actual loopback dial result per configured target. An accepted socket proves only TCP reachability.                                          |
| Application response   | Last observed HTTP status and time on the HTTP dispatcher. An upstream 4xx/5xx is an application response, not a failed tunnel registration. |

Record `observed_at`, source, target/configuration revision, bounded reason and
elapsed time. Show freshness separately (`fresh`, `stale`, `unknown`); after
30 seconds without an observation show stale, not continuous success. A new
failure replaces the previous current outcome while retaining a labelled last
success timestamp. Scope by canonical server/owner, tunnel generation and local
route/port; a late result cannot overwrite new config or re-create closed state.

Start with passive observation of real traffic in raw and HTTP paths. Raw mode
can report dial/copy failures, **not HTTP application status**. Preserve separate
IPv4/IPv6 attempts under the existing five-second loopback preparation bound;
one family's refusal is not aggregate failure if the other succeeds. Reuse the
loopback dialer, never caller-controlled hosts, environment proxies or DNS.

Replace per-tunnel apex latency polling with one bounded, non-overlapping
reachability read per running client's server scope, at most once every five
seconds with a two-second read deadline. Schedule the next refresh only after
the previous bounded read ends; check HTTP status, honour cancellation
and expose last failure/staleness. This is a live interactive status read, not a
job executor or application probe. Do not add periodic active upstream checks.

### Proposed explicit diagnostic command

```sh
# Proposed; not available yet. Does not open a public tunnel.
burrowctl doctor 3000 --output json
burrowctl doctor --config ./burrow.json --output json
```

Accept exactly positional ports or an explicit existing route config; reuse
validation and infer/deduplicate targets from it. No credential/config wizard,
automatic config-file discovery or target editing. Validate before all I/O;
help/invalid input never generates an owner, opens capture or touches a keyring.

Run bounded interactive reads: one apex `/health` read and TCP connect/
close to each explicit loopback target, at most four concurrently, at most
32 targets and ten seconds total, five seconds per target within that deadline.
Return every target's actual result, including `not_checked` on budget exhaustion.
No HTTP application request, TLS guessing, redirects, public tunnel creation,
reservation, revocation or arbitrary URL/path fetching. A TCP connection may be
visible to an app, so document it rather than call it side-effect-free.

An offline local check can still succeed while server reachability fails. The
standalone process cannot observe another process's registration/passive history;
report that dimension unavailable, not inferred from an owner list. Text/JSON
must explain likely next steps (start the explicit port; check binding; wait for
reconnect; inspect application logs) without asserting an unproven root cause.
Use existing exits: 0 for requested checks satisfied, 1 for failed/incomplete
checks, 2 for invalid config. Cancellation stops reads and returns the documented
clean-cancellation outcome, never a successful diagnostic summary.

In the running TUI, use a scoped health details view from the selected tunnel or
route; preserve back/selection and show last observation/freshness. No modal wizard
or full-page spinner. A user-requested recheck uses the same bounded TCP check,
not a background application request. If future application probes are wanted,
they need a separately approved method/path/side-effect contract.

## O4 — truthful metadata and useful inspection

### Source boundaries and measurements

Keep three explicitly labelled sources; do not join by timestamp/path or pretend
their opaque IDs identify the same event:

1. **Server logs:** owner-authorized, up to 200 retained completed requests per
   tunnel. Populate authoritative final HTTP status from the reverse proxy;
   capture proxy-generated errors as such. Preserve existing response-stream
   `size_bytes` semantics (headers included), not a new payload/bandwidth meaning.
   A status read from the yamux peer may have been generated by the client proxy;
   without explicit evidence its origin is `tunnel_peer`, not proven application
   success/failure. Do not invent a route or local error classification there.
2. **Local live inspection:** in-memory metadata on the running client, enabled
   by proposed `--inspect`. It uses the existing private HTTP dispatcher even for
   a single port and works with routes/capture, but does not enable either capture
   or replay. Explicitly document its stricter path/HTTP policy versus raw mode.
   Keep the raw path unchanged when inspection, routing and capture are off.
3. **Encrypted captures:** existing opt-in persisted request copies and durable
   replay outcomes. Retain separate IDs, expiry, redaction and erasure semantics.
   A log row without a saved capture is never replayable. Do not extend the
   capture schema merely to make server/local rows look correlated.

Extend shared HTTP observations to support local health/inspection alongside
capture without replacing `Table.Observe` or running duplicate body tees. Local
metadata contains ID, generation, method, escaped path without query, matched
route prefix (nullable on rejection), start/end time, numeric duration, observed
status, status origin and bounded outcome/reason. No request/response headers,
bodies, query values, cookies or new IP field. Do not send local routes, ports or
health observations to the server, Redis or telemetry.

Record app response status at headers, but mark the request completed only when
the body/stream lifetime finishes. Distinguish upstream HTTP error, local dial
failure, route rejection, overload, cancellation, transport loss and drain deadline
when actually observed. Mid-body failure can retain HTTP 200 with
`outcome=interrupted`; do not rewrite it to 502 after headers. 101 means upgraded,
not application completion. Unknown status/duration/reason remains null/unknown,
including observations from old servers. A client-side interruption cannot prove
that a side effect did not occur.

Instrument `ReverseProxy.ModifyResponse`/error paths and explicit stream lifetime
tracking without breaking Flusher, Hijacker, informational responses, trailers
or `http.ErrAbortHandler`. Finish exactly once on errors/panics too. Mark an
unsupported/abrupt case unknown rather than guessing. Session-fenced callbacks
must not append old-generation records to a new session's history.

Bound local completed metadata to 200 entries and 512 KiB per tunnel, evict oldest
first; cap each path at 2 KiB UTF-8 with explicit truncation. No disk persistence
or optional store/keyring access. Active inspection state is bounded by the
existing 128 admitted HTTP streams; publish completed rows nonblocking and show
dropped/evicted counts when known. Keep server history at 200 and add the same
per-path/per-tunnel byte bounds. These are per-tunnel limits, not a claim of a
global owner/tunnel quota. Reset volatile metadata when the process/session ends;
mark reset/rollover and retained-window limits visibly.

### Shared filtering contract

```sh
# Proposed server-metadata queries; keep existing non-filtered output compatible.
burrowctl logs TUNNEL_ID --method POST --status 5xx --min-duration 250ms --path /api
burrowctl logs TUNNEL_ID --follow --search hook --output jsonl
# Local capture summary filter; does not scan decrypted bodies.
burrowctl requests list --name webhook --route /api --method POST --output json
# Running-client local inspection, not a new remote management endpoint.
burrowctl tunnel 3000 --inspect
```

One query implementation serves CLI and TUI adapters. Combine dimensions with
AND; repeated method/status values use OR. Method is an exact HTTP token; status
accepts an exact final code or `2xx`–`5xx`; unknown matches only an explicit
`unknown` selector. Duration is a nonnegative Go duration, inclusive lower bound;
unknown duration never matches a numeric threshold. Path is a case-sensitive
escaped path-segment prefix; `/api` does not match `/apiculture`. Search is a
case-insensitive literal substring of displayed method/path/route/reason only,
not regex or body/header/query/IP search. Bound search/path input to 256 bytes
and repeated values to 16. Validate before network/storage I/O.

Apply filters/sort to the **entire authorized retained source snapshot**, before
tail or viewport slicing. Default order is completion/append order with stable
ID tie-breaks, not start timestamp. Support explicit duration/status ordering in
the finite CLI/TUI view with unknown values last; follow uses append order only.
Expose finite CLI ordering as `--sort duration` / `--sort status` (ascending)
with `--descending`; reject these flags with follow rather than silently ignore
them. Start-time fields remain observation data, not the default arrival order.
Show `matching / retained`, source, freshness and truncation/window/reset warnings;
never call them all-time totals. Existing finite `logs` JSON remains an array;
use additive row fields and an additive `snapshot` event for JSONL follow metadata,
not a silent envelope shape change.

Server logs have no route dimension. CLI `logs --route` must fail with an
actionable unsupported-source message, not guess from the request prefix or
silently return zero rows. Route filtering belongs to the running local view and
`requests list`; capture summaries do not have original response latency/status,
so those filters there must likewise fail explicitly. No private local RPC or
cross-process inspector service is introduced.

For old-server snapshots, safe method/path/search/duration filtering remains
client-side over the bounded retained set. Status filters must explain unknown
measurements, not fabricate 200. Add optional capability/field presence for new
outcome metadata; unknown additive fields remain ignorable by old clients.

Follow fetches must remain single-flight and bounded. Deduplicate by source,
generation/window and opaque ID, tracking the full retained snapshot, not only
matching rows. Initial `--tail` applies after filtering; later rows emit once,
including late completions. Report possible gaps/unsupported continuity when
polling misses an eviction; bounded snapshots are not a durable event stream.
Never claim exact missed counts when the source cannot provide them.

### TUI behavior and privacy

- Keep controls above results and a fixed feedback/freshness line. Add compact
  filter/search controls and a focused details view, not nested empty panels or
  another primary navigation destination. Preserve the existing visual language.
- Fetch outside `Update` through cancellable commands with one in-flight fetch,
  a selected-source/generation token and bounded deadline. Late responses for a
  previous tunnel/view cannot replace the current rows or error state.
- Preserve filter drafts, source, sort, focus, selected ID and visible row offset
  through refresh and details/back. If selection is evicted, show that fact and
  choose the nearest remaining row. No unexpected jump to a different request.
- Follow must be explicit: pause it on manual navigation; only resume/jump to
  newest on the user's follow action. Pausing holds the viewport, not an
  unbounded hidden backlog. Refresh failures retain rows labelled stale/error;
  no full-screen loading state, repeated toast or movement of the input/footer.
- Sanitise terminal control sequences in paths/methods/errors, preserve printable
  Unicode, clip by display-cell width and never copy hidden secrets. Metadata
  paths can themselves be sensitive; keep them owner-scoped/in memory and out of
  telemetry and public readiness. No export, body search or increased retention.

## Acceptance gates

| Gate               | Required evidence before the implementation PR is ready                                                                                                                                                                                                                                                                          |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Admission race     | Barrier-controlled concurrent register/request/shutdown tests: no admission after the transition, no reserved generation changed by a rejected registration, no Redis/network I/O under map locks.                                                                                                                               |
| Drain              | Admitted delayed GET/POST completes exactly once before deadline; new traffic gets 503 without reaching the app. Infinite body, stalled headers, SSE, WS and cancelled clients finish/close within the 20-second cap plus a documented scheduling tolerance. Repeat signals/callers observe one truthful result.                 |
| Failure/ownership  | Disposable real Redis outage, conditional cleanup against newer generation, revocation during drain and abrupt process kill. Same URL reconnects when valid; no tombstone loss, takeover or automatic original POST resend. Count effects at the synthetic app.                                                                  |
| Compatibility      | Pinned old/new clients and servers, registration-only and lifecycle peers; lost/duplicate/oversized/malformed/control-EOF notices; no control bytes reach an upstream. Persistent capability/downgrade guards remain enforced.                                                                                                   |
| Health             | Closed port, IPv4-only/IPv6-only, blackholed dial, TCP accept without HTTP, app 404/500, server 503/offline, stale success and late old-config results. Startup remains usable with an offline app; passive health sends zero synthetic app requests. Doctor respects 4/32/5s/10s bounds and never creates a tunnel/owner/store. |
| Measurements       | HTTP 200/204/302/404/500, 100/103 before final response, local-generated 502/503, mid-body abort, upload/trailers, SSE/WS and drain interruption. Unknown remains unknown; no duplicate records or changed response-byte meaning.                                                                                                |
| Query scope        | Mixed data beyond one viewport, late completion, 200-entry rollover, unknown fields, invalid flags, UTF-8/truncation/ANSI injection, identical timestamps and route mismatch. Filtering precedes tail/page; counts describe only the retained authorized source.                                                                 |
| Stable interaction | Native terminal fixtures at 80×24 and 110×32, plus narrow 60×20 fallback: search draft/focus, selected visible row, sort/filter and details/back survive slow/success/error/offline refresh, eviction and tunnel switching. No reducer-blocking network I/O.                                                                     |
| Resource/privacy   | Stalled output/metadata consumer cannot block forwarding; bounded memory/worker counts, cancelled fetches and leak/race checks. Capture off opens no SQLite/keyring, writes no bodies and sends no route/health metadata to providers. Existing replay crash/erasure/uncertainty tests remain green.                             |
| Combined           | Isolated CLI subprocess plus real Redis and synthetic app, named routed inspected tunnel, concurrent requests, SIGTERM drain, same-ID reconnect, application failure/recovery and history filtering. Repeat with capture off and existing Linux encrypted capture/replay acceptance on.                                          |

Run Go tests/vet across all three modules and race tests; Linux builds and Darwin
cross-builds are required but **do not prove native macOS acceptance**. Reuse the
real-service and tmux harnesses under `burrowctl/cmd/`, `burrowctl/internal/` and
`scripts/test/`. Use short injected deadlines for fast tests plus a real-time
default-budget integration gate; do not replace predicates with sleeps or relax
limits to turn failures green. Record measured shutdown/reconnect durations,
completed/interrupted effects and actual tested workload, not a throughput SLO.

Infrastructure/release changes also require `bash scripts/infra/check.sh`,
`node --test scripts/ci/*.test.mjs scripts/release/*.test.mjs`, Docker build and
configuration-policy coverage. No new runner or bypass. Verify native exact-head
`CI Gate`; correctly unselected jobs must be skipped. Do not substitute a green
manual source dispatch for a failing native PR check.

## Documentation, release and approval boundaries

Update `client.md`, `server.md`, CLI help and `deploy.md` alongside each
implemented contract. Review `tunnel-workflows-implementation.md`, README and
`cicd-hardening-review.md` for stale status/continuity claims. Keep schema v1
compatibility examples and distinguish server logs, local live inspection and
encrypted replayable captures. Document health freshness, capture-off privacy,
timeouts, forced termination and unsupported sources/platform evidence.

Capture only actual synthetic terminal states after implementation: drain/reconnect,
health failure/recovery and filtered/details/pause. Store relevant captures under
`documentation/previews/tunnel-operations/`; embed at most four previews per PR row
and link remaining evidence from docs. This planning-only PR has no screenshots
because no interface changed.

Source acceptance is not deployment acceptance. Before changing the hosted health
path or claiming rollout improvement, require an explicit owner-authorized test
of the real provider termination sequence with exact digests and synthetic traffic.
Record admission cutoff, completed/interrupted requests, SIGTERM/force-kill timing,
reconnect and new-generation readiness separately. Keep this gate **pending** until
evidence exists; no live mutation, paid resource, public tunnel or traffic test is
authorized by this plan. If native macOS verification is unavailable, say so and
leave that acceptance item pending instead of inferring it from cross-compilation.

Production publication/promotion remains unchanged. Use a verified compatible
last-good digest or reviewed forward fix; never roll back below the persistent
attachment floor, erase durable state or enable replicas to mask drain failures.
After the owner merges each task PR, remove only its clean stale worktree and
retain local plans/evidence. No automatic merge or deployment is part of this work.
