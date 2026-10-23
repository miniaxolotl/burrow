# CI/CD hardening — Burrow review

## Implementation update

The owner expanded the review to executable fixes. Burrow now shares GET-only
bounded GitHub JSON reads, checksum-preserving bootstrap retries, common-action
pin contracts and protected browser/Next cache writers with all four repositories.
Writes are not retried by the read helper. Three attempts share a 30-second budget;
auth/not-found and persistent invalid evidence fail without exposing provider bodies.

Development PR CI now selects website, Go and infrastructure jobs by dependencies;
shared protocol selects all Go jobs plus image/infra validation. Docs-only changes
skip heavy jobs, while unknown/empty/unreadable diffs and package manifests select
all. Renames cover both paths. CI Gate validates exact inventory and selected
success/unselected skip, never blanket skipped-job acceptance. Go tests run
independently of vet/build; the final gate still requires every selected result.
Full push/manual source CI remains mandatory release evidence. Frozen promotion
PRs/dispatches reuse it only after authenticated admission; fake development-PR
promotion inputs fail. Actual deployment operations remain unchanged in this slice.

Runner placement is retained by owner instruction, superseding the proposed vendor
move. Draining and automatic recovery remain subsequent work;
the dated baseline below is historical where superseded by this update.

Infrastructure applies now recheck the protected production tip immediately
before each of the two OpenTofu mutations and before reporting readiness. Release
and maintenance workflows supply a read-only GitHub token and pinned Node runtime.
An isolated command-fixture regression proves a branch superseded between applies
stops before the second write. This is not a provider-side atomic lock or rollback.

Reviewed **22 October 2026**, development `facc1de6b3322953f61632f609483638c3a23037`.
This review refreshes the recovered plan against current source and read-only
GitHub policy. It changes no workflow, runtime, infrastructure or provider setting.
Follow [deployment operations](deploy.md) for the supported release path and the
[shared review](https://github.com/bramble-labs/website/pull/38) for coordination.

## Current baseline and missing work

Original hardening commits `8cb491f`, `1fe4785`, `c837ecd`, `c71e10a` and `50dfcdc`
do not resolve after fetching current remotes; the original local branch/worktree
is absent. Their historical selection/reuse, drain/reconnect, recovery, runner/cache
and read-retry results are not fresh evidence. Recover/compare externally supplied
patches or explicitly reconstruct them. Merged promotion #58 is not proof these
unpublished local patches shipped.

| Area                  | Implemented source                                                                                                     | Planned hardening                                                                     |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| CI                    | `ci.yml`: unconditional website, infrastructure, Go quality/build/test and admission; all-success CI Gate              | Dependency-aware validation/deployment selection with explicit job inventory          |
| Runners/cache         | `ubuntu-latest`, Go/pnpm and BuildKit caches                                                                           | Blacksmith Linux subject to access/lint; build/browser caching and measured sizing    |
| Admission/publication | Frozen candidate, independent source CI, exact release tree, immutable publication preflight/reuse                     | Authenticated validation reuse; bounded GitHub/bootstrap read retries                 |
| Rollout               | `deploy.yml` rejects stale production and applies immutable image digest through OpenTofu; real HTTPS/WSS tunnel smoke | Last-good fenced recovery, desired-state reconciliation, verified rollback and alerts |
| Shutdown              | `Server.Shutdown` marks closing and closes existing tunnel connections before HTTP shutdown                            | Explicit in-flight drain/reconnect continuity qualification; not zero-downtime proof  |

Source references: `.github/workflows/ci.yml`, `deploy.yml`,
`scripts/release/promotion.mjs`, `policy.json`, `scripts/infra/apply.sh`,
`burrowd/internal/server.go` and `burrowctl/internal/live_test.go`.

## Corrections and safety requirements

- Preserve the existing strict gate. Unknown/empty/unreadable diffs must be
  conservative; cover deletions and both rename paths, shared protocol, toolchains,
  lockfiles, infrastructure and website dependencies. Missing selected jobs must
  fail, not become accepted skips. Keep independent exact-source CI and admission.
- Reused evidence must authenticate repository, workflow/producer, source SHA/tree,
  run attempt, complete selected jobs, tooling/policy revision and artifact digest.
  Reject PR-only/self, stale, partial and expired evidence. Cancellation is not reuse.
- Existing npm readback already retries missing versions without republishing;
  that is not a shared transient GitHub retry implementation. Retry only bounded
  read/bootstrap failures, preserve auth/not-found/identity failures, verify
  downloaded digests and never blindly replay ambiguous writes.
- One App Platform instance holds WebSocket/yamux sessions in memory. Client
  reconnection is existing behavior, not proof that active requests survive a
  rollout. Test in-flight requests, reconnect credentials and registration identity,
  stale connections and shutdown deadlines without enabling unsupported scaling.
- Automatic rollback must identify a previously verified image/deployment AND
  compatible runtime/Redis/configuration state. Fence forward and rollback writes
  against newer branch/provider operations; serialize infrastructure ownership and
  reconcile OpenTofu desired state so the next apply cannot undo recovery. Verify
  restored identity and real tunnel health, including rollback failure alerts.
  Do not down-migrate or retire retained staging resources as incidental cleanup.
- Keep the operator's reviewed new-release/revert recovery path until a separately
  reviewed controller and hosted healthy/failed/stale recovery drills are accepted.

## Policy readback

Read-only repository settings/full rulesets confirm strict Actions `CI Gate`
(integration `15368`) on development/production, PR/conversation requirements and
no bypass actors. Production is merge-only. Frozen `promote/**/*` refs and all
tags reject updates/deletion/forced changes. Auto-merge is enabled, rebase disabled,
and delete-branch-on-merge **disabled**. Policy tooling should compare desired and
current state before writes and demonstrate cleanup preserves protected/frozen refs.

## Revised sequence and proof

1. Recover/reconcile missing source and independent retry/promotion work; establish
   fresh Go race/vet/build, infra, website and release/workflow baselines.
2. Add dependency selection/strict aggregation with docs-only, shared protocol,
   rename/delete, unknown/empty/diff-failure and absent/failed/cancelled-job tests.
3. Review runner/cache trust and source cache writers, key invalidation and browser
   caching; benchmark comparable cold/warm/billed job costs before claiming savings.
4. Add authenticated evidence reuse and bounded read/bootstrap failure tests;
   never bypass artifact preflights or overwrite an existing published version.
5. Add draining/fenced deploy/verify/recovery with healthy, failed rollout, failed
   smoke, supersession, incompatible state, failed rollback and desired-state cases.
6. Separately authorize hosted settings/rollout/recovery drills and record exact
   provider readbacks, continuity and alert evidence. Local tests do not close them.

These are future acceptance requirements, not passing tests delivered by this
documentation PR. Execution plans remain local; assign minimally labelled PRs to
the owner, require exact-head checks, and remove only clean stale worktrees after
confirmed merge. No merge or production deployment is authorized by this review.
