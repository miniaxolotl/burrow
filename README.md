# burrow

[![npm](https://img.shields.io/npm/v/@miniaxolotl/burrowctl)](https://npmjs.com/package/@miniaxolotl/burrowctl)
[![License](https://img.shields.io/npm/l/@miniaxolotl/burrowctl)](LICENSE)

Expose local services to the internet via HTTPS subdomains.

```
https://swiftly-ancient-silent-dragon.burrow.mawa.dev → localhost:3000
```

## Server

```bash
cp .env.example .env   # set BURROW_SECRET
docker compose up -d
```

## Client

```bash
burrowctl auth login <token> --server your-server:25701
burrowctl tunnel create --port 3000
```

Or one-off:

```bash
BURROW_SERVER=your-server:25701 BURROW_SECRET=xxx burrowctl tunnel create --port 3000
```

## TUI

`create` launches an interactive manager:

| Key | Action |
|-----|--------|
| `n` | New tunnel |
| `c` | Copy URL |
| `o` | Open in browser |
| `d` | Close tunnel |
| `l` | View request logs |
| `q` | Quit |

## Local Dev

```bash
docker compose up -d redis
go build -o bin/burrowd ./burrowd
go build -o bin/burrowctl ./burrowctl
./bin/burrowd serve --secret dev --domain localhost
./bin/burrowctl tunnel create --server localhost:25701 --secret dev --port 3000
```

## Deploy

See `documentation/server.md` for Dokku deployment, wildcard SSL, and nginx config.

## Documentation

- [Client CLI reference](documentation/client.md)
- [Server setup & deployment](documentation/server.md)

## License

MIT
