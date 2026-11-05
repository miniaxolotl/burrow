# Tunnel workflow implementation

This is an in-progress implementation of the
[workflow development plan](tunnel-workflows-plan.md), not a release announcement.
All four features belong to one draft implementation PR stacked on planning
PR #84. After that parent merges, retarget the same implementation PR to
`development` and revalidate. Do not create per-feature implementation PRs.

## Checkpoints

| Checkpoint                                     | Status                                                                                                             |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| Isolated CLI subprocess fixtures               | Implemented: temporary HOME, no inherited owner/server configuration, private-owner and help/validation baselines. |
| Headless lifecycle and structured output       | Pending.                                                                                                           |
| Registration acknowledgement and saved aliases | Pending.                                                                                                           |
| One-hostname local routing                     | Pending.                                                                                                           |
| Encrypted capture and durable local replay     | Pending.                                                                                                           |
| Combined acceptance and full exact-head gates  | Pending.                                                                                                           |

Each checkpoint uses focused passing commits with matching tests and documentation.
Keep the PR draft until the whole plan's acceptance gates pass. No production
deployment, public tunnel or application-content capture is part of these tests.
