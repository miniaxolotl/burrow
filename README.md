# Burrow

Public HTTPS tunnels for localhost, by **Bramble Labs Inc.**

Share a local app, receive webhooks, and manage tunnels and request logs from
your terminal. Open source and self-hostable.

[Website](https://useburrow.app) · [Client reference](documentation/client.md) · [Self-hosting](documentation/server.md)

[Worked webhook and multi-port examples](documentation/workflows.md) share the
same source as the website's public reference pages. The
[discovery maintenance guide](documentation/website-discovery.md) explains builds
and publication boundaries; source changes are not a production rollout.

## Quick start

```bash
npm i -g @miniaxolotl/burrowctl
burrowctl tunnel 3000
```

Connects to `brmbl.dev` by default. The CLI creates a private owner credential
automatically, so only you can list, inspect, reconnect, or close your tunnels.
No account or manual key setup is required.

Official CLI releases collect anonymous command/tunnel outcomes and sanitized
diagnostics. Disable with `burrowctl telemetry disable`, `DO_NOT_TRACK=1`, or
`BURROW_TELEMETRY=0`. No command arguments, ports, URLs, credentials or tunnel
content are sent. See [telemetry and privacy](documentation/client.md#telemetry-and-privacy).

The CLI supports macOS and Linux on x64 and ARM64. To operate your own server,
follow the [server setup](documentation/server.md) and
[deployment guide](documentation/deploy.md).

## Website development

Requires Node.js 22+ and pnpm 10.

```bash
pnpm install --frozen-lockfile
pnpm dev
```

The marketing site runs at `http://localhost:3000`. See the
[website guide](packages/website/README.md) for builds and checks.

## Releases

Release Please manages versions and changelogs on `development`. A deliberately
selected, immutable release is promoted through a protected PR to `production`.
Only production publishes packages, archives, containers and the website.
See the [release runbook](documentation/deploy.md) for selection, retries and checks.

## License

[MIT](LICENSE)
