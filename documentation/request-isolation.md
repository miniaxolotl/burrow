# HTTP request-isolation regression

## Observed failure — 5 November 2026

Production v0.3.18 intermittently returned an unrelated development application's
404 page for Burrow apex health and management requests. An independent temporary
Toronto Droplet, a fresh in-memory owner and three synthetic local services
reproduced an unexpected 404 without the fixture marker at the first 5-RPS stage:
8 of 9 measured requests succeeded before the safety stop. Across that small
aborted sample, p50 was 104 ms and p95/max 168 ms. This was a correctness failure
at light load, **not** a capacity measurement or evidence of saturation.

The operator's global client/config and development tunnels were not used or
modified. The independent fixture's active tunnels were closed and its temporary
Droplet, provider SSH key and local private keys were deleted and absence verified.

## Cause and repair

The former proxy hijacked an entire HTTP origin connection after its first tunnel
request and relayed every subsequent byte to that first local service. An ingress
can reuse an origin connection across Host values, so a later request for another
tunnel or the apex management API could bypass Burrow's Host dispatcher. The same
design delayed request logs until connection closure and could not respond through
an HTTP/2 ResponseWriter, which does not support hijacking.

A deterministic local reproduction sends a tunnel request, an apex health request,
a second tunnel request and management/error requests on the **same** origin
connection. Before the repair, the apex received the first local service's body.
Additional failing tests found no per-request logs while keep-alive stayed open
and an empty HTTP/2 response. A live ingress pooling trace was not captured; the
local mechanism and independently observed symptom are distinguished here.

The repair uses Go's standard `httputil.ReverseProxy` with a request-scoped yamux
stream/HTTP transport. Ordinary requests remain under net/http dispatch; only a
negotiated 101 upgrade takes over the connection. Public Host, paths, query strings,
bodies, response headers and response-stream accounting remain available. No
application quotas, credentials, dependencies or infrastructure settings change.

## Verification and rollout

```bash
go test ./burrowd/internal -run '^(TestProxy|TestTunnelHTTPKeepAlive)' -count=1 -v
go test ./burrowd/... ./burrowctl/... ./protocol/...
go vet ./burrowd/... ./burrowctl/... ./protocol/...
# A C compiler and CGO are required for the race detector:
CGO_ENABLED=1 go test -race ./burrowd/... ./burrowctl/... ./protocol/...
```

Tests cover same-origin cross-host reuse, per-request history, HTTP/2 and WebSocket
echo in addition to the existing keep-alive/body regression. No screenshots are
needed for this backend-only change.

The source fix must ship through a green development PR, Release Please patch
release and frozen production promotion; a local pass does not mean the hosted
service is repaired. After rollout, run both the
[deployment smoke and isolated bounded load test](deploy.md#bounded-live-load-testing).
Record new results separately from the v0.3.18 failure above. A passing 100-RPS
short ramp is not proof of maximum or sustained capacity.

## Production verification — v0.3.19

The repair shipped on 5 November 2026 through
[fix #81](https://github.com/miniaxolotl/burrow/pull/81),
[source release #82](https://github.com/miniaxolotl/burrow/pull/82), and
[frozen promotion #83](https://github.com/miniaxolotl/burrow/pull/83).
[Deployment 42107710694](https://github.com/miniaxolotl/burrow/actions/runs/42107710694)
passed publication, immutable-image rollout and the enhanced HTTPS/reconnection
smoke test. Exact source `552ee32a7a15e89241e12c881597834fcfc5f80f` and production
merge `1478cfff563c467b2587c99bd65395b47923217a` have identical trees. App Platform
reported ACTIVE deployment `0fbd1f17-e615-405f-82a5-4b900cc00542`; its image matched
the published manifest:

```text
ghcr.io/miniaxolotl/burrowd@sha256:17eab6978efac142d936ac00f46b9fa42723ee05753509e7067d9f37a20c546d
```

An independent temporary Toronto `s-1vcpu-1gb-intel` Droplet ran the exact-release
compiled `TestLiveTunnelLoad` and `TestLiveTunnel` against production. The load
fixture kept HTTP keep-alive enabled, used a fresh in-memory owner and unique
marker/path/body checks, and forwarded through three simultaneous synthetic
tunnels. **All 2,100 measured load requests passed without retries or errors.**

| Target RPS | Requests | Achieved RPS | p50 | p95 | p99 |
| --- | --- | --- | --- | --- | --- |
| 5 | 50 | 4.98 | 32 ms | 59 ms | 65 ms |
| 15 | 150 | 14.96 | 28 ms | 68 ms | 113 ms |
| 30 | 300 | 29.92 | 27 ms | 43 ms | 73 ms |
| 60 | 600 | 59.81 | 28 ms | 39 ms | 64 ms |
| 100 | 1,000 | 99.58 | 28 ms | 40 ms | 58 ms |

Each stage lasted about ten seconds. Peak observed in-flight requests were six
(the safety ceiling was 32); the largest observed latency across stages was
211 ms. These are end-to-end measurements from that generator, not isolated
server CPU timings. They establish the tested short workload, **not maximum
capacity, a sustained-load SLO, a bandwidth allowance or a 32-concurrent-request
acceptance result**.

Additional live checks passed:

- Management routing retained the three-test-tunnel owner scope between stages.
- Upload-and-echo integrity: 64 KiB in 39 ms, 1 MiB in 150 ms, 4 MiB in 513 ms.
- History retained 200 entries, including the latest payload completion. For the
  payload tunnel, response-stream usage was 6,205,518 bytes versus 5,561,189 bytes
  in retained entries, confirming eviction did not erase its cumulative count.
- A transport WebSocket message of 1 MiB + 1 byte was rejected with close code
  1009. This is distinct from the successfully streamed 4 MiB HTTP body.
- The separate smoke verified HTTPS bodies/headers, query strings, API-like
  application paths, apex health isolation and same-URL reconnection in 2.59 s.
- Published npm CLI `0.3.19` launched from an isolated cache. Public website/apex
  health returned HTTP 200 with valid TLS; `www` redirected to `useburrow.app`.
  The production website deployment reported success for the production commit.

Both tests passed. Active test-owner tunnels were explicitly closed and absence
verified. The temporary Droplet, provider SSH key and local private-key files were
removed and their absence verified. Persistent revoked tunnel ownership
tombstones remain by design. The operator's global CLI/config was never changed.

A later [heavier 100–1,000 RPS plan](heavy-load-2026-11-05.md) stopped at the
250-RPS stage after one 3.5-second response exceeded the test latency guard. All
7,000 completed responses had correct status/content; higher stages were not run.
That follow-up does not turn this bounded acceptance into a maximum-capacity claim.
