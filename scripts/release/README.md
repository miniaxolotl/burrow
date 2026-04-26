# @script/release

Builds burrowctl with goreleaser and publishes to npm.

## Usage

```bash
pnpm --filter @script/release run release
```

## Dry Run

```bash
pnpm --filter @script/release run release -- --dry-run
```

## What it does

1. Installs goreleaser and builds burrowctl binaries for all platforms
2. Updates version in platform-specific packages (`burrowctl-linux-x64`, etc.)
3. Publishes all burrowctl packages to npm
4. Creates GitHub release (if TAG is set and not `latest`)

## Environment Variables

| Variable | Description |
|----------|-------------|
| `NPM_TOKEN` | npm token for package publishing |
| `GH_TOKEN` | GitHub token for release creation |
