# Tunnel workflow implementation

This is an in-progress implementation of the
[workflow development plan](tunnel-workflows-plan.md), not a release announcement.
The owner subsequently requested a second implementation PR stacked on #88 for
all remaining features. #88 preserves the verified initial headless checkpoint;
`feat/tunnel-workflows-complete` contains the remaining work. Keep the child draft
until combined acceptance passes. Retarget and revalidate as parents merge.

## Checkpoints

| Checkpoint                                     | Status                                                                                                                                                                                                 |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Isolated CLI subprocess fixtures               | Implemented: temporary HOME, no inherited owner/server configuration, private-owner and help/validation baselines.                                                                                     |
| Headless lifecycle and structured output       | Initial M1a implementation: headless/TTY detection, JSON/JSONL, numeric units, truthful transport events, signals/cancellation and bounded output/drain. Final combined acceptance remains pending.    |
| Registration acknowledgement and saved aliases | Implemented in child: negotiated acknowledgement, aggregate readiness, durable attachment CAS/nonce reconciliation, private locked aliases and close/replacement state. More fault acceptance pending. |
| One-hostname local routing                     | Pending.                                                                                                                                                                                               |
| Encrypted capture and durable local replay     | Pending.                                                                                                                                                                                               |
| Combined acceptance and full exact-head gates  | Pending.                                                                                                                                                                                               |

Each checkpoint uses focused passing commits with matching tests and documentation.
Keep the PR draft until the whole plan's acceptance gates pass. No production
deployment, public tunnel or application-content capture is part of these tests.

## Implemented checkpoint evidence

- Subprocess fixtures use temporary HOME, private generated owners, synthetic
  loopback HTTP/WebSocket/yamux peers and disabled telemetry. They do not load the
  operator's credentials or connect to the hosted Burrow service.
- Red/green regressions cover structured results/errors, argument exit codes,
  real local forwarding, SIGTERM, partial startup rollback, repeated log snapshots,
  terminal reconnect failure, numeric units/unknown HTTP status, broken stdout and
  stalled writes. A stalled WebSocket upgrade is cancelled by closing only the
  handshake socket, without closing an accepted session on internal dial cleanup.
- Full normal Go tests and vet pass locally. The full race suite also passed in
  the existing Go 1.26.10 container with network disabled and read-only source/cache;
  rerun it on each published checkpoint. The host itself has no C compiler.
- #88 has no registration acknowledgement. The child adds negotiated registration
  and aggregate `ready` / `not_ready`; stable aliases, routing and capture/replay
  are not implied by that barrier. Combined acceptance remains pending.

### Registration protocol

Clients offer `X-Burrow-Registration: 1`; servers echo it only when supported.
Other requested versions are rejected before ownership reservation. Without the
echo, new clients retain transport-only legacy behavior and never emit `ready`.
No application probe is sent.

The first server-opened yamux stream is reserved for a single JSON registration
object followed by EOF, capped at 2048 bytes: version 1, tunnel ID and generation.
The server opens this stream before exposing the session, then writes only after
Redis registration and active-session installation. HTTP uses separate streams.
The server write deadline is five seconds; client acceptance/read is within the
30-second connection/startup deadline and cancelled with the command. Invalid,
oversized or mismatched acknowledgements fail startup. No control data is sent
to legacy clients.

`connected` reports transport establishment; `registered` includes the matching
generation and `registration_confirmed: true`. Aggregate `ready` contains all
acknowledged IDs/generations/URLs. Loss emits `not_ready` before reconnect details;
recovery needs fresh acknowledgements for every tunnel. Neither registration nor
a response status proves application processing succeeded.

### Persistent attachment evidence

Server unit tests cover conditional/idempotent commit, metadata expiry, stale
refresh/cleanup, revocation and legacy bypass; live loopback tests reject busy
sessions. Private-registry tests cover restart/uncertain attempt reuse, scope,
locking, corrupt/newer schemas, symlinks/permissions and close/replacement races.
An actual CLI subprocess reuses its URL after metadata removal and server restart
with retained Redis. This passed against both miniredis and a disposable Redis
7.4 container. The disconnect grace is shortened explicitly in that fixture;
default-grace and partition-liveness acceptance remain pending. No hosted service
or public tunnel is involved.

The rollout path now enforces the attachment-state image compatibility floor
documented in [deployment guidance](deploy.md). Release-policy tests include the
new guard without weakening current-ref or single-instance checks.

### Signal-cause compatibility regression

The first full source CI run caught a Go 1.26.2-specific cancellation case: an
interrupted HTTP request can wrap the signal context's cause (`terminated signal
received`) rather than `context.Canceled`. The CLI now recognizes either the
cancelled command context's error or its cause, without treating unrelated errors
as clean shutdown. Text and JSON subprocess regressions cover it; both passed
500 repetitions each on CI's exact Go 1.26.2 toolchain after the fix. Do not infer
CI success from a newer local toolchain's passing result alone.
