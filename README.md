# Burrow

Tunnel server for exposing local services via subdomains.

## How it works

```
Client → HTTPS → Nginx → burrowd (WebSocket/yamux) → local service
```

## Quick Start

**Server:**
```bash
go build -o bin/burrowd ./burrowd
./bin/burrowd serve --secret SECRET --domain example.com
```

**Client:**
```bash
go build -o bin/burrowctl ./burrowctl
./bin/burrowctl tunnel create --port 3000 --server example.com
```

## Components

| Component | Description |
|-----------|-------------|
| `burrowd` | Server daemon |
| `burrowctl` | Client CLI |
| `protocol` | Shared message types |

## Environment Variables

**Server:**

| Variable | Description | Default |
|----------|-------------|---------|
| `BURROW_PORT` | Listen port | `25701` |
| `BURROW_REDIS_URL` | Redis URL | `localhost:6379` |
| `BURROW_SECRET` | Auth secret | (required) |
| `BURROW_DOMAIN` | Tunnel domain | `inkspire.app` |

`PORT` and `REDIS_URL` are also supported as fallbacks.

**Client:**

| Variable | Description |
|----------|-------------|
| `BURROW_TOKEN` | Auth token |

## Tunnel URLs

`https://{random-id}.{domain}` e.g. `https://arcane-dragon-42.inkspire.app`

## Development

```bash
go build -o bin/burrowd ./burrowd
go build -o bin/burrowctl ./burrowctl
go test ./...
```

## Docker Compose

```bash
docker-compose up --build
```

## Dokku Deployment

```bash
dokku apps:create burrowd

# Install Redis plugin if needed
sudo dokku plugin:install https://github.com/dokku/dokku-redis.git

dokku redis:create burrowd-redis
dokku redis:link burrowd burrowd-redis
dokku config:set burrowd BURROW_SECRET=xxx BURROW_DOMAIN=example.com

git remote add dokku dokku@server:burrowd
git push dokku main
```

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/tunnels` | GET | List tunnels (auth required) |
| `/tunnel/ws` | GET | WebSocket |
| `/tunnel/{id}` | POST | Create tunnel |
| `/tunnel/{id}` | DELETE | Delete tunnel |

## License

MIT