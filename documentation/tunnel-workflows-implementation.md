# Tunnel workflow implementation

This is an in-progress implementation of the
[workflow development plan](tunnel-workflows-plan.md), not a release announcement.
All four features belong to one draft implementation PR stacked on planning
PR #84. After that parent merges, retarget the same implementation PR to
`development` and revalidate. Do not create per-feature implementation PRs.

## Checkpoints

| Checkpoint                                     | Status                                                                                                                                                                                              |
| ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Isolated CLI subprocess fixtures               | Implemented: temporary HOME, no inherited owner/server configuration, private-owner and help/validation baselines.                                                                                  |
| Headless lifecycle and structured output       | Initial M1a implementation: headless/TTY detection, JSON/JSONL, numeric units, truthful transport events, signals/cancellation and bounded output/drain. Final combined acceptance remains pending. |
| Registration acknowledgement and saved aliases | Pending.                                                                                                                                                                                            |
| One-hostname local routing                     | Pending.                                                                                                                                                                                            |
| Encrypted capture and durable local replay     | Pending.                                                                                                                                                                                            |
| Combined acceptance and full exact-head gates  | Pending.                                                                                                                                                                                            |

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
- No registration acknowledgement is implemented yet: output deliberately reports
  `registration_confirmed: false` and never emits `ready`. Saved aliases, routing,
  encrypted capture/replay and real-Redis acceptance remain subsequent checkpoints
  in this same implementation PR.

### Signal-cause compatibility regression

The first full source CI run caught a Go 1.26.2-specific cancellation case: an
interrupted HTTP request can wrap the signal context's cause (`terminated signal
received`) rather than `context.Canceled`. The CLI now recognizes either the
cancelled command context's error or its cause, without treating unrelated errors
as clean shutdown. Text and JSON subprocess regressions cover it; both passed
500 repetitions each on CI's exact Go 1.26.2 toolchain after the fix. Do not infer
CI success from a newer local toolchain's passing result alone.
