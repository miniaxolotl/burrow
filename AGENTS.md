# AGENTS.md

## Project

Burrow is a tunneling service: expose local services to the internet via HTTPS subdomains. Two Go binaries (`burrowd` server, `burrowctl` client) share a `protocol` module. The npm package `@miniaxolotl/burrowctl` is a thin Node.js wrapper that dispatches to platform-specific Go binaries.

## Repository layout

- **`burrowd/`** — Go server (cobra CLI, entrypoint `main.go`, commands in `cmd/`, core in `internal/`)
- **`burrowctl/`** — Go client (cobra CLI, TUI via charmbracelet/bubbletea in `cmd/tui/`)
- **`protocol/`** — Shared Go module (`burrow/protocol`) with WebSocket/message/auth types; both binaries replace-directive it as `../protocol`
- **`packages/burrowctl/`** — npm wrapper (`bin/burrowctl.js` resolves platform-specific binary)
- **`packages/burrowctl-{darwin,linux}-{arm64,x64}/`** — platform npm packages, each ships a pre-built Go binary
- **`scripts/release/`** — TypeScript script that runs goreleaser and publishes npm packages
- **`scripts/deploy/`** — TypeScript script that builds/pushes Docker images
- **`lib/`** — shared pnpm workspace configs (eslint, typescript)

## Build & dev commands

```bash
# Build both Go binaries (must run from repo root, uses go.work)
go work sync
go build -o bin/burrowctl ./burrowctl
go build -o bin/burrowd ./burrowd

# Or use the helper script
bash scripts/build.sh

# Lint (CI does this)
go vet ./burrowd/... ./burrowctl/... ./protocol/...

# Test (no test files exist yet)
go test ./burrowd/... ./burrowctl/... ./protocol/...

# npm wrapper build (from repo root)
pnpm install --frozen-lockfile
pnpm run build          # builds the npm package via workspace filter
pnpm run lint           # lints the npm package
pnpm run format         # prettier on ts/md/json
```

## Key constraints

- This is a **Go workspace** (`go.work`) with 3 modules. Always `go work sync` before building if module deps changed.
- The `protocol` module is linked via `replace` directive in both `burrowd/go.mod` and `burrowctl/go.mod` — never `go install` it independently.
- Version strings live in `burrowd/version.go` and `burrowctl/version.go`; release-please bumps them, goreleaser overrides at link time with `-X main.Version`.
- Server requires **Redis** (see `docker-compose.yml`). Default port 25701.
- `.env` is gitignored; copy `.env.example` and set `BURROW_SECRET` + `BURROW_DOMAIN` to run locally.
- Branches: `development` → release-please creates PRs; `production` → triggers npm publish + Docker deploy with `latest` tags.
- No Go test files exist yet, but CI runs `go test` on all three modules.
- Node >= 22, pnpm 10, Go 1.26.