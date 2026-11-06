# Tunnel operations implementation

Stacked on planning PR #93; no merge, release or hosted activation is implied.
The original [plan](tunnel-operations-plan.md) remains a historical proposal.

## Review decisions

- Admission and request lifetime share one short map barrier. Registry sequencing
  uses bounded cancellable key gates instead of a global network mutex. Drain
  never waits on a map/lifecycle lock held across remote I/O.
- Shutdown stops new traffic first, retains admitted HTTP/SSE/upgrade lifetimes,
  then closes transports after completion/deadline. Repeated calls share the first
  result. No original request is automatically retried.
- Registration-v1's object/EOF contract is unchanged. Only explicit lifecycle-v1
  peers consume a separate reserved stream. Generation-fenced notices invalidate
  transport readiness without cancelling admitted work early. Legacy peers get
  ordinary reconnect, not fabricated shutdown provenance.
- Health dimensions remain independent. Doctor is credential-free, explicit,
  bounded and TCP-only. Passive HTTP 500 is an application response, not failed
  registration. Legacy local callbacks use a private session window fence without
  pretending that window is an acknowledged server generation.
- Inspection composes with capture's existing body wrapper once. It is metadata
  only, volatile, bounded and separate from encrypted replayable requests. Source
  limitations are validated before management/storage access.
- Compatibility adjustment to the proposal: local inspection resets on session
  replacement, but the existing server history retention across same-owner
  reconnect is preserved. Generation-labelled rows and fenced callbacks keep the
  sources truthful without silently discarding the legacy authorized log window.
- Log refresh runs outside the terminal reducer, single-flight with deadlines and
  scope tokens. Rows/drafts/selection remain mounted on failed refresh; manual
  navigation pauses follow. No local inspector listener or daemon is introduced.

## Verification status

Unit/subprocess tests cover admission, delayed POST completion, deadline SSE and
hijacked-WebSocket interruption, idle/zero-timeout truthfulness, public-host
dispatch, registration EOF, lifecycle framing, readiness invalidation, doctor
privacy/failure, query/tail scope, metadata bounds, informational-status
handling, mid-body interruption, selection/draft stability and nonblocking
refresh. Full ordinary and race suites pass across all three modules, and
Darwin amd64/arm64 cross-builds compile.

Real-service acceptance on disposable local infrastructure (explicitly gated):

- `TestOperationsProcessesWithRealRedis`: real `burrowd`/`burrowctl` binaries,
  real SIGTERM, default 15s drain, disposable Redis, isolated HOME. Measured a
  15.004s shutdown: admitted POST completed exactly once, late POST rejected
  with 503 + `Retry-After` without reaching the app, forced SSE interruption
  reported as an error, same tunnel URL re-registered with a new generation, no
  automatic resend (one effect at the synthetic app).
- `TestNativeOperationsTerminal` (tmux): paused selection, filter drafts,
  details, drain/reconnect notice and scoped health failure/recovery at
  80×24, 110×32 and 60×20; captures under
  [previews/tunnel-operations](previews/tunnel-operations/).
- Combined workflows (`TestCombinedWorkflowsRealServices`) pass with `--inspect`
  enabled alongside Linux encrypted capture/replay on the disposable secure
  backend, so inspection composes with capture without duplicate body tees.

Exact-head native PR `CI Gate` is verified on the pull request itself; this
document records source evidence only.

Native macOS acceptance is pending: cross-compilation cannot prove native
behavior. Hosted termination ordering and rollout improvement are pending
owner-authorized provider tests. `/health` remains the hosted health path;
`/ready` supplements, not replaces, immutable-image/generation admission. No
replica increase, rollback floor change, paid resource or production traffic
test is part of this work.
