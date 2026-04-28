# burrowctl

[CLI client](https://github.com/miniaxolotl/burrow) for the burrow tunnel server. Expose local services to the internet via HTTPS subdomains.

## Install

```bash
npm i -g @miniaxolotl/burrowctl
```

## Quick start

```bash
burrowctl auth login <token>
burrowctl tunnel create --port 3000
```

No account required for the default server (`burrow.mawa.dev`). Your tunnel will be available at `https://<tunnel-id>.burrow.mawa.dev`.

## Usage

```bash
# Create a tunnel
burrowctl tunnel create --port 3000

# Create multiple tunnels
burrowctl tunnel create --port 3000 --port 8080

# List active tunnels
burrowctl tunnel list

# View tunnel logs
burrowctl tunnel inspect <tunnel-id> --follow

# Close a tunnel
burrowctl tunnel close <tunnel-id>
```

## Configure a custom server

```bash
burrowctl auth set-server your-server.example.com
burrowctl auth login <token>
burrowctl tunnel create --port 3000
```

## Links

- [GitHub Repository](https://github.com/miniaxolotl/burrow)
- [Package Registry](https://npmjs.com/package/@miniaxolotl/burrowctl)
- [Server Setup](https://github.com/miniaxolotl/burrow#self-hosting)