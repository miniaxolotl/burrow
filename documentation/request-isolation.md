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
