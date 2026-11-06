# burrowctl

Share local services over HTTPS. No account, login, or manual credential setup.

## Before you start

- Use macOS or Linux on x64 or ARM64. The npm package does not ship a Windows binary.
- The package-manager installation needs Node.js and npm or pnpm in your terminal.
  [Native binaries](https://github.com/miniaxolotl/burrow/releases) do not need Node.js.
- Have a local HTTP service running, and use test data. Burrow does not start your
  application or add authentication to it.

## Your first tunnel

In a terminal, install the client and check that the executable is available:

```bash
npm install -g @miniaxolotl/burrowctl
burrowctl --version
```

If you use pnpm, install with `pnpm add -g @miniaxolotl/burrowctl` instead. For a
global-directory permission error, use a writable package-manager prefix or the
native binary; do not change system permissions just to install Burrow.

Start your application using its normal development command. Verify its port
locally before opening a tunnel, for example:

```bash
curl -i http://127.0.0.1:3000/
```

Check for the response your application normally returns. If this request fails,
fix the local service first. In another terminal, using that same port, run:

```bash
burrowctl tunnel 3000
```

Keep this terminal open. The interactive manager displays the assigned HTTPS
address. Copy the actual address into a browser: it should show the same app as
your local URL. The example address in a screenshot is not your tunnel.

For a full disposable endpoint with an exact expected response, continue with
the [webhook walkthrough](workflows.md#receive-a-test-webhook). It also covers
multiple local services and explicit cleanup.

## Commands

```bash
burrowctl tunnel 3000 8080          # Share ports and open the interactive manager
burrowctl list                     # Your active tunnels and count
burrowctl logs <id> --follow        # Your tunnel's request logs
burrowctl close <id>                # Close your tunnel permanently
```

`logs --tail N` limits the output. All commands support `--server <host[:port]>`, for example:

```bash
burrowctl tunnel 3000 --server staging.brmbl.dev
```

There are no privileged users, global lists, or admin commands. Each command sees only the tunnels belonging to its owner credential. `list` works from a second terminal using the same server and local configuration.

## Headless mode and structured output

**Development branch:** these additions are being implemented in the draft
[workflow implementation](tunnel-workflows-implementation.md); this is not a
claim that the published npm release already includes them. Capture currently
requires a usable Linux Secret Service backend; macOS capture remains unavailable
pending a verified non-interactive Keychain adapter.

```sh
burrowctl tunnel 3000 --headless
burrowctl tunnel 3000 --output jsonl
burrowctl list --output json
burrowctl logs TUNNEL_ID --tail 20 --output json
burrowctl logs TUNNEL_ID --follow --output jsonl
burrowctl close TUNNEL_ID --output json
```

An ordinary interactive terminal still opens the manager. When stdin or stdout
is not a terminal, `tunnel` defaults to plain headless output. `--headless` makes
that choice explicit; JSON/JSONL implies it. Headless mode never opens a browser,
copies a URL or reads interactive input. Keep the process running to forward.

| Format  | Contract                                                                                                            |
| ------- | ------------------------------------------------------------------------------------------------------------------- |
| `text`  | Human-readable output without TUI control codes in headless mode.                                                   |
| `json`  | One result/error envelope for `list`, non-following `logs` and `close`.                                             |
| `jsonl` | One envelope per line; required for machine-readable `tunnel` and `logs --follow`. Finite commands also support it. |

Structured stdout contains only envelopes. Errors use that same format, not a
second human error or usage banner. Human errors use stderr; a broken/blocked
output sink can require a short stderr diagnostic because stdout is unavailable.
Help, version and telemetry preferences remain ordinary text, not structured APIs.

Every envelope has `schema_version: 1`, `type`, UTC `timestamp`, per-invocation
increasing `sequence`, nullable `tunnel_id`, and `data`. `timestamp` is emission
time; transport events separately include their actual `data.observed_at` so
queued output does not pretend an old observation is fresh. Ignore unknown
additive fields/event types; incompatible meanings require a schema-version change.

- Finite success: `result`; failure: `error` with a bounded `data.code` and a
  sanitized message. Current codes are `invalid_argument` and `operation_failed`.
  Provider response bodies and owner credentials are not printed in machine errors.
- Transport events: `connecting`, `connected`, `disconnected`, `reconnecting`,
  `failed`; clean process cancellation emits `stopped`. Reconnect events include
  consecutive `attempt` and actual `retry_delay_ms`, with capped jittered backoff.
- `connected` means transport only. On a compatible server, `registered` carries
  its acknowledged generation and `registration_confirmed: true`; aggregate
  `ready` follows only after every requested tunnel is registered. `not_ready`
  invalidates that barrier on connection loss. Old servers have no registration
  barrier and never produce `ready`. None of these proves application health or
  successful application processing.
- List results contain `tunnels` and `count`; unknown `latency_ms` is null, and
  response usage is `total_size_bytes`. Log results are arrays; follow emits `log`
  entries without repeating a snapshot. `--tail` applies only to the first
  followed snapshot; entry IDs preserve late request completions.
- Log `duration_ms` uses integer milliseconds; `size_bytes` retains the server's
  response-stream measurement, including HTTP headers. Unknown `status_code` is
  null, not zero/200; the current server forwarding path does not populate HTTP
  status. These remain metadata logs, not replayable request captures.

Exit codes are 0 for successful commands/clean SIGINT or SIGTERM shutdown, 1 for
runtime/output failure, and 2 for invalid arguments/configuration. Streaming
commands reject `--output json`; use `jsonl` instead. Arguments are checked before
creating an owner. Connection startup is bounded to 30 seconds and shutdown drain
to five seconds. An output write blocked for two seconds fails rather than
leaving forwarding running invisibly behind a stalled consumer.

Headless startup stops the invocation if any requested connection fails and
disconnects connections it already opened. This is rollback, **not atomic public
exposure**: a connection may have been briefly reachable. It does not permanently
revoke those IDs. A terminal authorization/ownership/revocation rejection during
reconnect likewise stops the whole headless invocation nonzero.

## Automatic ownership

### Encrypted local capture and durable replay (development)

```sh
burrowctl tunnel 3000 --name webhook --capture --output jsonl
burrowctl requests list --name webhook --output json
burrowctl replay REQUEST_ID --name webhook --yes --output jsonl
burrowctl replay status JOB_ID --output json
burrowctl replay jobs --name webhook --output json
burrowctl replay retry JOB_ID --yes --output jsonl
burrowctl requests delete REQUEST_ID --name webhook --yes --output json
burrowctl requests clear --name webhook --yes --output json
```

Capture also works without a name, including with `--config`; select its records
with `--tunnel ID` instead. Selectors are mutually exclusive. Capture must be
enabled explicitly for each process and currently supports one tunnel per command.
It uses the HTTP dispatcher even for a single upstream; the strict path policy
therefore applies. Historical server logs are not replayable capture records.

**Bodies can contain secrets.** Capture is local-only, encrypted with AES-GCM
before entering SQLite, never uploaded to Burrow, Redis or telemetry. Supported
Linux sessions require an existing unlocked Secret Service collection on a local
session bus. Keys stay in that OS facility. Missing/locked/ambiguous backends fail
before exposure, without calling Unlock/Prompt or falling back to a key file.
macOS currently refuses capture; ordinary headless/raw/routed tunneling still
works without opening this store or consulting a credential facility.

Limits are 1 MiB per request body, 32 KiB request headers, eight in-flight/pending
copies, 200 live records and a 24-hour access TTL. The reviewed initial payload
budget is **20 MiB**, not the proposal's 50 MiB live-payload budget: a 24 MiB SQLite
page limit plus its worst-case rollback journal stays below a 50 MiB physical
envelope. Pending replay references count toward the same payload budget and are
protected from eviction. Expired/oldest unreferenced records are evicted first;
if no safe space exists, the new capture is unavailable rather than blocking
forwarding or exceeding quota. Content-free job/tombstone rows are bounded too.

Only `capture_saved` means a complete encrypted record committed durably. A
bounded nonblocking tee queues persistence; incomplete bodies, excessive bodies,
unsupported encodings, upgrades/SSE, request trailers, excluded paths and writer
backpressure report `capture_unavailable` with a reason. There is no plaintext
spool. `--capture-exclude /private,/uploads` skips explicit path prefixes.
Live forwarding retains application authorization; durable copies strip cookies,
authorization, proxy/secret/signature headers and redact sensitive query keys.
This can change meaning or break signatures. Bodies are not automatically scrubbed.
Default lists expose IDs, route summaries, sizes, timestamps and redaction state,
not headers, query values or bodies.

With capture enabled, press `x` in the interactive manager for local captures.
`r` opens a scoped replay confirmation; `y` approves exactly one durable job.
`d` opens local erasure confirmation. The fixed feedback area displays persisted
job/HTTP states, including completed HTTP failures and uncertain outcomes. A
refresh preserves selected record identity and its visible row; confirmation
cancel/back preserves the selection. This pane does not read or print raw bodies.

Replay always requires approval, including GET. Without an interactive terminal,
use `--yes`; that approves only the selected record and captured loopback target.
Review the method/path, age, target, digest and redaction/duplicate-effect warning.
A port can now belong to another app: Burrow cannot prove application identity.
Replay never goes through the public URL, restores secrets, regenerates signatures
or invents idempotency keys. Changed captured config files/current alias target
digests fail before dispatch; target overrides are not supported.

`queued` is emitted only after the command/work row commits. A foreground worker
claims a fenced lease, with one active replay per owner/server scope, at most 20
pending jobs, a 30-second job deadline and at most three preparation attempts.
Transient pre-send storage errors use persisted `retry_wait`; interrupted
preparation becomes `recovering` and requires explicit retry. Before any HTTP
client can write, `dispatch_intent` commits after expiry/tombstone/lease checks.
Once intent exists, that job is **never automatically sent again**, even if no
byte was actually sent. A crash/lost response is conservatively `uncertain`.

The worker uses a fresh loopback connection, no environment proxy, redirects,
cookie jar, credential injection or automatic network retries. Response bodies
are not archived. `completed` means response headers/status committed, including
4xx/5xx, not successful application processing. Non-2xx, failed, uncertain and
cancelled outcomes exit nonzero. `status` reads durable state without executing
anything. Explicit `retry` approves a linked new job and fences the old one;
an active old worker is a conflict. Normal tunnel startup never dispatches old jobs.
Capture summaries include their last job; `replay jobs` recovers lost acceptance
receipts and retained content-free outcomes even after a capture is erased. Its
optional selector narrows the current owner/server scope. An uncertain leaf job
cannot be bypassed with a fresh unlinked replay; use an explicitly approved retry.

Deletion immediately tombstones/fences unsent work; after dispatch, cancellation
is best-effort and cannot undo effects. Cleanup is durable local work executed
while a CLI worker runs or at its next startup: the TTL is not guaranteed physical
deletion while the machine is off. Ciphertext is cleared from active rows, but
the shared scope key remains in OS credentials. SQLite journals/free pages,
filesystem snapshots and user-managed backups prevent a forensic/cryptographic
erasure promise. Capture backups/restore/exports are unsupported; restoring an
old backup can predate tombstones and must not be used for replay.

### One hostname, multiple local services (development)

Create an explicit route file, then run `burrowctl tunnel --config ./burrow.json`
(optionally with `--name shop` and/or `--output jsonl`):

```json
{
  "version": 1,
  "routes": [
    { "path": "/api", "port": 8080, "strip_prefix": true },
    { "path": "/", "port": 3000 }
  ]
}
```

The longest escaped path-segment prefix wins, case-sensitively: `/api` matches
`/api/x`, not `/apiculture` or `/%61pi/x`. Prefix stripping is explicit; `/api`
and `/api/` become `/`, and `/api/x` becomes `/x`. Query/body and public Host are
preserved; `host_header` optionally sets a bounded authority for that route.
Unmatched paths return 404 unless `/` exists. An unavailable target returns 502
without falling through. Targets are fixed loopback ports only, with IPv4/IPv6
fallback, no environment proxy and no caller-selected destination.

No file is auto-loaded or hot-reloaded. Reject positional ports plus `--config`,
unknown/duplicate fields, unsupported versions, files over 64 KiB and more than
32 routes before exposure. Prefixes cannot contain escapes, queries, fragments,
dot segments, backslashes, repeated/trailing slashes (except `/`) or non-ASCII.
Routed requests containing encoded separators, dot segments (including encoded
dots), NUL, backslashes, repeated slashes or invalid escapes return 400, not a
cleaned/redirected path. Plain single-port raw mode retains existing behavior.

The private HTTP dispatcher admits one request/upgrade lifetime per yamux stream,
with 32 KiB headers, at most 128 concurrent streams, bounded dial/header deadlines
and explicit overload failure. It streams bodies, trailers, SSE and WebSocket
upgrades without an overall short streaming deadline. It rejects CONNECT and
absolute-form targets and does not forward a second pipelined request. It strips
untrusted Forwarded/X-Forwarded/X-Real-IP and Burrow-management/proxy credential
headers; no trusted client identity is asserted on this path.

The route table appears in text and typed lifecycle/readiness data. Press `r` in
the manager for the full scrollable startup table. Routed configuration is fixed
for that invocation; the manager cannot add an inferred upstream. A routed tunnel
counts as one server connection; the first port is informational metadata, not
the complete route table. A shared public origin also shares cookie/security
trust: configure application cookie paths, base paths and CORS appropriately.

### Saved URLs (development)

```sh
burrowctl tunnel 3000 --name webhook --output jsonl
burrowctl saved --output json
burrowctl close TUNNEL_ID --output json
burrowctl tunnel 3000 --name webhook --replace --yes --output jsonl
```

Names are private local aliases, not vanity subdomains. A saved alias reuses its
random URL across CLI/server restarts for the same canonical server and owner.
Unnamed commands still create fresh URLs. One named tunnel is allowed per command;
ports are explicit. `saved` reports local saved/closed/uncertain/conflict state and
observation time, not a live server snapshot, and never generates an owner token.

Persistent mode requires capability discovery and registration support; old or
unreachable servers fail before a persistent claim, with no random-URL fallback.
Private candidate IDs, logical attachments and attempt nonces are atomically
persisted before admission. Kernel-held locks prevent simultaneous local use.
After uncertain admission, the same nonce reconciles the committed generation;
a stale generation is terminal, never an automatic takeover. Copying private owner
and alias files gives their holder equivalent authority, not a new trust boundary.

Normal exit disconnects without revoking the URL. Explicit `close` is permanent;
uncertain closes are reconciled via an owner-only reservation read before reuse.
Replacement requires a confirmed closed alias and `--yes` and chooses a new random
ID; it never clears a tombstone. Replacement text warns about the old URL and target.
Missing/corrupt/newer local state fails without discarding it. Preserve files and
owner credentials for recovery; a URL alone cannot recover ownership.

Only one server instance is supported. Persistent sessions heartbeat every ten
seconds; an unresponsive transport is closed after a twenty-second ping timeout
(at most thirty seconds from an idle failure). Busy admission may wait within the
30-second startup deadline. Connection metadata expiry does not expire ownership
or fencing. Legacy clients cannot attach to persistent IDs, even as the same owner.

The first tunnel command generates a random 256-bit owner credential for that server. It is stored under `~/.config/burrow/owners/<server-hash>.token`, with directory permissions 0700 and file permissions 0600. Concurrent first launches share one identity. Different servers receive different credentials; returning to a server reuses its identity.

Keep this directory private and back it up to retain access to your sessions. Sharing a tunnel URL does not share its credential or grant access to its logs, counts, reconnection, or closure. The exposed application's contents remain public unless that application requires authentication.

## Configuration

Default settings live in `~/.config/burrow/client.json`. Flags and environment variables override them.

| Flag       | Environment     | Purpose                                          |
| ---------- | --------------- | ------------------------------------------------ |
| `--server` | `BURROW_SERVER` | Server address; defaults to `brmbl.dev`          |
| `--token`  | `BURROW_TOKEN`  | Optional explicit owner credential               |
| `--domain` | `BURROW_DOMAIN` | Fallback domain if an older server omits the URL |

A saved explicit token is used only for its configured server, never forwarded to a different `--server` override. Management requests reject cross-origin redirects. There is no shared server secret or credential with elevated privileges.

TLS is required for remote servers. `localhost`, `127.0.0.1`, and `::1` use HTTP for local development.

## Telemetry and privacy

Official release binaries send anonymous operation outcomes to PostHog and
sanitized errors, sampled traces and operation metrics/logs to Sentry in the US.
There is no account identity: `~/.config/burrow/telemetry-id` is a random local UUID
(0600), separate from owner credentials and browser analytics. No command
arguments, filesystem paths, hostnames, target ports, tunnel IDs, URLs, IPs,
headers, bodies or clipboard contents are collected. Telemetry headers are never
added to tunnels or local requests.

```bash
burrowctl telemetry status
burrowctl telemetry disable          # Persist opt-out and clear analytics identity
burrowctl telemetry enable           # New identity on the next invocation
DO_NOT_TRACK=1 burrowctl tunnel 3000  # Disable for this process
BURROW_TELEMETRY=0 burrowctl list     # Equivalent environment opt-out
```

Running commands honor persisted opt-out before uploading. Re-enabling never
revives an older process's cleared identity. CI and local source builds do not
initialize telemetry. Help, version and telemetry preference commands do not emit
events. Opt-out does not erase already received provider data; contact
support@useburrow.app for requests. Random IDs are not linked across website/CLI or
products, and do not provide access to tunnels.

First-value activation means the first forwarding stream in a tunnel session
returned bytes from the local service. It does not inspect HTTP response status
or content, and subsequent requests do not repeat the activation event. TUI and
CLI creation share that same owner. Expected network/authorization failures are
bounded `unavailable` outcomes; unexpected operational failures use `error`.

Telemetry uses bounded nonblocking queues and a maximum 550ms shutdown wait;
provider failure or queue overflow drops telemetry rather than delaying tunnels.
See [the observability contract](product-observability.md) for provider settings.

## Interactive controls

| Key            | Action                         |
| -------------- | ------------------------------ |
| `n`            | Add a port                     |
| `c` / `o`      | Copy / open the selected URL   |
| `d`            | Close the selected tunnel      |
| `l`            | View logs / return to the list |
| `s`            | Change sorting                 |
| `f` / `r`      | Follow / refresh logs          |
| `?`            | Help                           |
| `q` / `Ctrl+C` | Exit                           |

## Reconnection and upgrades

Transient connection failures retry the same URL with exponential backoff while the CLI is running. Request logs survive short disconnects for 20 seconds. Ownership stays reserved after connection records expire, so another owner cannot claim a previously shared URL. Explicit closure revokes the ID; create a new tunnel for a fresh URL. Authorization failures, ownership conflicts, and revocation stop automatic reconnection.

Upgrade the client and server together for the owner-only API. Older tokenless clients must upgrade and create new tunnels; legacy anonymous IDs are retired. Existing explicit tokens can still identify their owner, but never grant privileged access. Use the four commands above in place of the old nested `tunnel` subcommands. The old `auth` commands and `--secret` flag are removed.

See the [server API](server.md) and [deployment guide](deploy.md).

## Troubleshooting setup

| Symptom                         | Check                                                                                                                                                                     |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `burrowctl` is not found        | Confirm the installation completed and the package manager's global executable directory is on PATH. Reopen the terminal after changing PATH.                             |
| The URL cannot reach the app    | Repeat the local curl request, check the selected port and keep the tunnel process running. A frontend may also need its own allowed-host or public-origin configuration. |
| A webhook returns 404           | Confirm the method and path against the application. Opening a URL in a browser sends GET; it does not send a test POST.                                                  |
| Management cannot find a tunnel | Use the same server and owner configuration as the process that created it. Do not delete the owner files or borrow someone else's credential.                            |
| A connection keeps failing      | Check outbound connectivity to the selected server. Do not disable TLS verification to make a remote server connect.                                                      |

For unresolved issues, include the operating system, `burrowctl --version`, a
redacted command and whether the local request succeeds when
[contacting support](https://useburrow.app/support). Exclude credentials,
authorization headers and private request bodies.
