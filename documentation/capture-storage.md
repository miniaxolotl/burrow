# Optional local capture store

This is a development component, not a published-release or platform-support
announcement. See [workflow status](tunnel-workflows-implementation.md) and
[client commands](client.md).

## Storage and keys

- The local SQLite work table is the queue; no hosted service/broker is added.
  `modernc.org/sqlite v1.61.1` keeps the packaged CLI CGO-free.
- Sensitive request metadata and bodies are AES-GCM encrypted before SQLite.
  Authenticated data binds schema, owner/server scope and record identity. Neither
  plaintext temporary files nor raw sensitive WAL/journal values are written.
- Linux uses an existing local Secret Service session through `godbus/dbus`.
  Locked/missing/ambiguous facilities fail closed. The adapter never calls Unlock
  or Prompt and does not auto-launch a session bus. The scope master key remains
  in OS credentials. macOS capture remains unavailable until a non-interactive
  Keychain adapter is implemented and verified.
- Limits: 1 MiB request body, eight bounded capture slots, 200 live records,
  20 MiB encrypted payload accounting and a 24-hour access TTL. SQLite is bounded
  to 24 MiB of pages, reserving space for its rollback journal below a 50 MiB
  physical envelope. Job/tombstone rows have independent bounded retention;
  full retention quotas reject new work rather than discard fencing evidence.

## Capture publication

HTTP forwarding tees only bounded memory to a nonblocking writer queue. No capture
is accepted until complete encrypted bytes/metadata commit. Unsupported/incomplete
requests or a full/failed writer are unavailable, not replayable. Clear increments
a durable scope epoch; each capture keeps the epoch observed before copying, so
an old writer cannot restore cleared content. Later requests adopt the new epoch
after the worker observes it; foreground request handlers never read the disk.

## Replay and recovery

Approval commits a job and payload reference before `queued`. Claims are atomic,
one active replay per owner/server scope, with at most 20 pending jobs, a 30-second
deadline and a lease extending beyond dispatch. Preparation backoff is persisted
and capped at three attempts. No transaction stays open during network I/O.

`dispatch_intent` is write-ahead evidence: after it exists, recovery never sends
that job again. Lost outcomes are uncertain; observed response headers/status are
completed, including non-2xx. Explicit approved resends create linked jobs and
fence the old one. The transport uses a fresh loopback connection with no redirects,
proxies, restored credentials or automatic network retries. Bodies of responses
are not archived. Deletion cancels unsent work and conservatively marks dispatched
work uncertain; it cannot undo external effects.

Cleanup is durable local work run at startup and by the active capture worker.
No daemon or wall-clock physical deletion while the machine is off is promised.
Ciphertext clearing and private permissions are not forensic/cryptographic
erasure: the shared scope key, journals/free pages and user backups have separate
lifecycles. Backup restoration is unsupported for replay.
