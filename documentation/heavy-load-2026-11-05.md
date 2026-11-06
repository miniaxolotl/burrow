# Heavy production load test — 5 November 2026

## Result: latency safety stop, not HTTP corruption

Production v0.3.19 completed **7,000 valid HTTP responses** through three isolated
synthetic tunnels. The 100-RPS stage passed. At the 250-RPS stage, one response took
**3.5 seconds**, exceeding the operator's two-second observed-latency guard. The
heavy test failed safely and did **not** run the planned 500/750/1,000-RPS stages.
Do not describe the 52,000-request plan or 1,000 RPS as executed or accepted.

| Target RPS | Requests | Achieved RPS | HTTP/content errors | Latency guard breaches | p50 | p95 | p99 | Max |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 100 | 2,000 | 99.89 | 0 | 0 | 24 ms | 33 ms | 51 ms | 174 ms |
| 250 | 5,000 | 242.34 | 0 | 1 | 66 ms | 250 ms | 331 ms | 3,500 ms |

Stage wall times, including final response drain, were 20.023 s and 20.633 s.
The latency guard is evaluated when a response completes; all 5,000 higher-stage
requests had completed by the time the slow response triggered it. Peak observed
in-flight requests were five and 44; the configured ceiling was 128. No measured
request was retried. This guard is a test safety choice, **not** a published
product SLO, hard capacity ceiling or requests-per-second quota.

## Isolation and provenance

- Production commit: `1478cfff563c467b2587c99bd65395b47923217a`; release source:
  `552ee32a7a15e89241e12c881597834fcfc5f80f` (v0.3.19).
- Runtime remained ACTIVE on deployment `0fbd1f17-e615-405f-82a5-4b900cc00542`, one
  `apps-s-1vcpu-0.5gb` instance, before and after testing. Immutable image digest:
  `sha256:17eab6978efac142d936ac00f46b9fa42723ee05753509e7067d9f37a20c546d`.
- Harness source: `09d456b4353a8272a540d47074907f7120e64ec3`. Test executable SHA-256:
  `548a665e224cf00b0b3dc65d6d81e5b0cbf77814a91915f5004641257620ee87`.
- A temporary Toronto `s-2vcpu-2gb-intel` Droplet ran a source-built test executable,
  private temporary HOME/config, fresh in-memory owner and unique response marker.
  HTTP keep-alive was enabled; every status, marker, path and 1-KiB echo body was
  checked. The global client/config and existing development tunnels were untouched.
- The two-vCPU generator used 7.864 CPU-seconds over 44.545 seconds including
  setup/smoke: average CPU 17.65% of one core, sampled peak 27.98% of one core,
  maximum RSS 18,404 KiB. No generator CPU/RAM saturation was observed. These are
  generator measurements, **not** production-server resource metrics.

## Recovery, cleanup and interpretation

The separate HTTPS/apex-isolation and same-URL reconnection smoke passed afterward
in 2.53 s. A subsequent apex health check returned correct JSON, HTTP 200, in
118 ms. Active test-owner tunnels were explicitly closed and absence verified.
The temporary Droplet, provider SSH key and local private-key files were removed;
independent provider list reads verified absence. Durable revoked ownership
tombstones remain by server design. No runtime configuration, release or deployment
changed during this test.

This sample shows higher tail latency in the 250-RPS stage, but one outlier does
not establish a repeatable breakpoint or identify a software defect. Server CPU,
memory, ingress timings and network/provider effects were not isolated. Correlate
those before any higher load or performance repair; investigation is tracked in
[issue #86](https://github.com/miniaxolotl/burrow/issues/86). The earlier
[2,100-request bounded acceptance](request-isolation.md#production-verification--v0319)
remains a separate successful short workload, not a maximum-capacity claim.

For the explicitly guarded heavy command and fixed traffic limits, see
[deployment testing](deploy.md#bounded-live-load-testing). No screenshots are
relevant to this backend-only benchmark.
