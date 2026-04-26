#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 {production|preview} [-f|--force] [-b|--branch <branch>]" >&2
  exit 1
}

ENV="${1:-preview}"
shift || true

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
FORCE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    -f|--force)   FORCE=1; shift ;;
    -b|--branch)  [[ $# -lt 2 ]] && usage; BRANCH="$2"; shift 2 ;;
    *)            usage ;;
  esac
done

case "$ENV" in
  production)  REMOTE="dokku" ;;
  preview) REMOTE="dokku-preview" ;;
  *)           usage ;;
esac

PUSH_ARGS=("$REMOTE" "$BRANCH:main")
[[ $FORCE -eq 1 ]] && PUSH_ARGS+=("--force")

echo "Deploying branch '$BRANCH' to $ENV ($REMOTE)..."
git push "${PUSH_ARGS[@]}"
