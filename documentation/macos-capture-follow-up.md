# macOS secure capture follow-up

## Scope and status

The owner separated macOS capture from [workflow PR #90](https://github.com/miniaxolotl/burrow/pull/90)
because a native macOS test machine is not currently available. This follow-up is
a deferred implementation task, **not an implemented Keychain adapter**. Keep its
PR draft until implementation and native acceptance pass. It does not block #90's
Linux capture/replay scope or ordinary macOS tunneling.

The current `burrowctl/internal/capture/keys_unsupported.go` rejects capture on
non-Linux platforms. Retain that guard until a supported adapter is verified.
Do not remove Darwin npm binaries, raw/routed tunneling, headless output, saved
URLs or macOS cross-builds. None of those requires capture key storage.

## Implementation work

- Add a Darwin-specific `OSKeys.Key` adapter backed by supported macOS Keychain
  APIs; preserve the existing `KeyStore` interface and owner/server scope.
- Bound all operations and explicitly disallow interactive authentication.
  Locked, missing, denied or ambiguous storage must fail before exposure without
  unlocking, opening a prompt, creating a plaintext key file or using a fallback.
- Persist a random 32-byte scope key only in Keychain and verify durable readback.
  Reopening a scope must recover the same key. Never replace an inaccessible key,
  silently reset encrypted data or disclose key bytes in arguments/logs/output.
- Review the adapter's API, dependencies, licensing and packaged-binary behavior.
  Preserve the existing CGO-free distribution contract unless a different build
  strategy is explicitly approved. Do not enable an unsafe shell-based adapter
  merely because it cross-compiles.
- Keep ordinary tunneling independent of Keychain access. Preserve encrypted
  SQLite, bounded copying, scoped erasure, durable replay approval and
  write-ahead dispatch-intent fencing; reuse the existing work store and workers.

## Native acceptance before enabling support

Use disposable test keychains and synthetic requests only, never the operator's
login keychain or real application payloads. Avoid changing the machine's global
authentication or default-keychain settings. Clean up only fixture-owned state.

- [ ] Usable secure storage: create/read/reopen the scoped key; reject cross-scope
      access, duplicate/ambiguous items and corrupted key data.
- [ ] Locked, absent and denied storage: fail within the documented deadline,
      without displaying a prompt, unlocking storage or opening a public tunnel.
- [ ] Ordinary tunneling still works when optional capture storage is unavailable.
- [ ] Actual CLI: capture → stop/restart → inspect summaries → explicitly approve
      replay → inspect durable job status → erase. Verify HTTP failure and
      uncertain outcomes do not automatically resend.
- [ ] Inspect database/journal/temporary files for synthetic plaintext; test TTL,
      erasure fencing, interrupted work and scoped credential failures.
- [ ] Verify both supported Darwin architectures and the shipped npm-wrapper /
      release binary path, including relevant signing/Keychain access behavior.
      Record which cases ran natively; cross-builds alone do not prove compatibility.
- [ ] Run source tests/vet/race, release/policy gates and exact-head source CI.
      Any native CI runner or paid macOS service requires a separate decision;
      this PR does not authorize one.
- [ ] Update client/storage docs and the platform support matrix only after
      native evidence proves support. Link reproducible fixtures and safe captures.

## Resume when macOS access is available

Read [capture storage](capture-storage.md), the [workflow implementation evidence](tunnel-workflows-implementation.md)
and the Linux adapter/tests before selecting the Darwin API. Start with failing
contracts, implement the smallest supported adapter, then verify the actual
packaged binary. Until then, this PR is a tracked draft with documentation only;
there is no macOS capture completion or release-support claim.
