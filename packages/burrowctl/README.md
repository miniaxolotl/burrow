# burrowctl

Share localhost over HTTPS. No account or manual key setup.

```bash
npm install -g @miniaxolotl/burrowctl
burrowctl tunnel 3000 8080
burrowctl list
burrowctl logs <id> --follow
burrowctl close <id>
```

Connects to `brmbl.dev` by default. Use `--server your-server.example.com` for a
custom server. The CLI generates a private, per-server owner credential and
stores it under `~/.config/burrow/owners/`. Only that credential can list/count,
inspect, reconnect, or close its tunnels. There are no privileged users.

Shared tunnel URLs expose your local application publicly; use the application's
authentication when access should be restricted.

Supports Linux and macOS on x64 and ARM64.

- [Client reference](https://github.com/miniaxolotl/burrow/blob/production/documentation/client.md)
- [Server setup](https://github.com/miniaxolotl/burrow/blob/production/documentation/server.md)
- [Source](https://github.com/miniaxolotl/burrow)
