#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 {production|development} [-f|--force] [-b|--branch <branch>]" >&2
  exit 1
}

[[ $# -eq 0 ]] && usage

ENV="$1"
shift

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
  development) REMOTE="dokku-preview" ;;
  *)           usage ;;
esac

PUSH_ARGS=("$REMOTE" "$BRANCH:main")
[[ $FORCE -eq 1 ]] && PUSH_ARGS+=("--force")

echo "Deploying branch '$BRANCH' to $ENV ($REMOTE)..."
git push "${PUSH_ARGS[@]}"
