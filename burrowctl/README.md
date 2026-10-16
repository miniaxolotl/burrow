# burrowctl

Share local services over HTTPS with private, account-free tunnel ownership.

```bash
npm install -g @miniaxolotl/burrowctl
burrowctl tunnel 3000
```

Four commands: `tunnel <port...>`, `list`, `logs <id>`, and `close <id>`.
All views and operations are limited to your owner credential. The CLI generates
and stores a separate private credential for each server automatically.

Use `--server <host[:port]>` or `BURROW_SERVER` for a custom server.
The public default is `brmbl.dev`. Remote connections require TLS.

Build from this Go workspace with `go build -o bin/burrowctl ./burrowctl`.

See the [client reference](../documentation/client.md) for configuration,
interactive controls, upgrades, and the ownership lifecycle.
