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
claim that the published npm release already includes them. Routing and
capture/replay remain pending.

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
