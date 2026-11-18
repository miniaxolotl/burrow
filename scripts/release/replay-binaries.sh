#!/usr/bin/env bash
# Compare two reviewed source snapshots without product/SHA stamp churn.
set -euo pipefail

if [[ $# != 3 || ! $3 =~ ^(linux|darwin)/(amd64|arm64)$ ]]; then
  echo 'usage: bash scripts/release/replay-binaries.sh BASE CANDIDATE {linux,darwin}/{amd64,arm64}' >&2
  exit 2
fi
cd "$(git rev-parse --show-toplevel)"
base=$(git rev-parse --verify --end-of-options "$1^{commit}")
candidate=$(git rev-parse --verify --end-of-options "$2^{commit}")
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export GOTOOLCHAIN=local CGO_ENABLED=0 GOOS="${3%/*}" GOARCH="${3#*/}" GOFLAGS='' GOWORK=auto
go_root=$(go env GOROOT)
go_bin="$go_root/bin/go"

for side in base candidate; do
  mkdir "$work/$side"
  git archive "${!side}" | tar -x -C "$work/$side"
  for binary in burrowd burrowctl; do
    (
      cd "$work/$side"
      # Replace only Release Please's generated assignment in this disposable tree.
      sed -E 's/^var Version = "[0-9]+\.[0-9]+\.[0-9]+" \/\/ x-release-please-version$/var Version = "replay" \/\/ x-release-please-version/' \
        "$binary/version.go" > "$work/version.go"
      mv "$work/version.go" "$binary/version.go"
      "$go_bin" build -mod=readonly -trimpath -buildvcs=false -ldflags='-s -w -buildid=' \
        -o "$work/$side-$binary" "./$binary"
    )
  done
done

printf 'Report-only binary comparison: %s -> %s (%s, %s)\n' "$base" "$candidate" "$3" "$("$go_bin" env GOVERSION)"
printf 'binary\tresult\tbase SHA256\tcandidate SHA256\n'
for binary in burrowd burrowctl; do
  before=$(sha256sum < "$work/base-$binary")
  after=$(sha256sum < "$work/candidate-$binary")
  result=different
  if [[ $before == "$after" ]]; then result=identical; fi
  printf '%s\t%s\t%s\t%s\n' "$binary" "$result" "${before%% *}" "${after%% *}"
done
echo 'Review the full diff for site, wrapper, build inputs and installer impact. No version or delivery decision.'
git diff --no-renames --name-status "$base" "$candidate" --
