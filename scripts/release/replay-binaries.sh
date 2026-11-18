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
cleanup() {
  for tree in "$work/base" "$work/candidate"; do
    if [[ -f $tree/.git ]]; then git worktree remove --force "$tree"; fi
  done
  rm -rf "$work"
}
trap cleanup EXIT

# Match the published native CPU levels, ignoring both the caller's build
# environment (including GOEXPERIMENT) and settings persisted by go env -w.
case ${3#*/} in
  amd64) cpu=GOAMD64=v1 ;;
  arm64) cpu=GOARM64=v8.0 ;;
esac
go_env=(env -i "HOME=$HOME" "PATH=$PATH" GOENV=off GOTOOLCHAIN=local
  CGO_ENABLED=0 "GOOS=${3%/*}" "GOARCH=${3#*/}" "$cpu")
go_root=$("${go_env[@]}" go env GOROOT)
go_bin="$go_root/bin/go"

for side in base candidate; do
  git -c core.hooksPath=/dev/null -c core.sparseCheckout=false worktree add --quiet --detach "$work/$side" "${!side}"
  for binary in burrowd burrowctl; do
    (
      cd "$work/$side"
      # Replace only Release Please's generated assignment in this disposable tree.
      sed -E 's/^var Version = "[0-9]+\.[0-9]+\.[0-9]+" \/\/ x-release-please-version$/var Version = "replay" \/\/ x-release-please-version/' \
        "$binary/version.go" > "$work/version.go"
      mv "$work/version.go" "$binary/version.go"
      workspace=off
      if [[ -f go.work ]]; then workspace="$PWD/go.work"; fi
      "${go_env[@]}" "GOWORK=$workspace" "$go_bin" build -mod=readonly -trimpath -buildvcs=false -ldflags='-s -w -buildid=' \
        -o "$work/$side-$binary" "./$binary"
    )
  done
done

printf 'Report-only binary comparison: %s -> %s (%s, %s, CGO_ENABLED=0, %s)\n' \
  "$base" "$candidate" "$3" "$("${go_env[@]}" "$go_bin" env GOVERSION)" "$cpu"
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
