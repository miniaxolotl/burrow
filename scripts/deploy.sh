#!/usr/bin/env bash
# Deploy script: builds and pushes multi-platform Docker image to GHCR and/or Docker Hub,
# then creates a GitHub release.
#
# Registries are enabled by setting the corresponding env var:
#   GHCR_REGISTRY=ghcr.io/miniaxolotl   → push to GitHub Container Registry
#   DOCKERHUB_REGISTRY=miniaxolotl      → push to Docker Hub
#
# GitHub release is created when GH_TOKEN or GITHUB_TOKEN is set.
#
# Examples:
#   GHCR_REGISTRY=ghcr.io/miniaxolotl TAG=v0.1.0 ./scripts/deploy.sh
#   DOCKERHUB_REGISTRY=miniaxolotl TAG=v0.1.0 ./scripts/deploy.sh
#   GHCR_REGISTRY=ghcr.io/miniaxolotl DOCKERHUB_REGISTRY=miniaxolotl TAG=v0.1.0 ./scripts/deploy.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -f "${ROOT}/.env.deploy" ]]; then
  set -a
  # shellcheck source=/dev/null
  source "${ROOT}/.env.deploy"
  set +a
fi

IMAGE="burrowd"
REPO="miniaxolotl/burrow"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"

registry_login() {
  if [[ -n "${GHCR_REGISTRY:-}" ]]; then
    local token="${GH_TOKEN:-${GITHUB_TOKEN:-}}"
    if [[ -n "$token" ]]; then
      echo "$token" | run docker login ghcr.io -u "$USER" --password-stdin
    else
      echo "⊘ No GH_TOKEN — skipping GHCR login"
    fi
  fi

  if [[ -n "${DOCKERHUB_REGISTRY:-}" ]]; then
    local token="${DOCKERHUB_TOKEN:-}"
    if [[ -n "$token" ]]; then
      echo "$token" | run docker login -u "${DOCKERHUB_REGISTRY%%/*}" --password-stdin
    else
      echo "⊘ No DOCKERHUB_TOKEN — skipping Docker Hub login"
    fi
  fi
}

run() {
  echo "> $*"
  "$@"
}

git_revision() {
  git -C "$ROOT" rev-parse HEAD
}

build_tags_args() {
  local tags=("$@")
  local args=()
  for t in "${tags[@]}"; do
    args+=("-t" "$t")
  done
  echo "${args[@]}"
}

create_github_release() {
  local tag="$1"
  local token="${GH_TOKEN:-${GITHUB_TOKEN:-}}"

  if [[ -z "$token" ]]; then
    echo "⊘ No GH_TOKEN or GITHUB_TOKEN — skipping GitHub release"
    return
  fi

  echo ""
  echo "--- Creating GitHub release ${tag} ---"

  if ! git -C "$ROOT" tag -l "$tag" | grep -q "^${tag}$"; then
    run git -C "$ROOT" tag -a "$tag" -m "Release ${tag}"
    run git -C "$ROOT" push origin "$tag"
  fi

  run gh release create "$tag" --generate-notes --repo "$REPO"
  echo "✓ GitHub release ${tag} created"
}

deploy() {
  registry_login

  local tag="${TAG:-latest}"
  local version

  version="$(git -C "$ROOT" describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "")"

  local tags=()
  if [[ "$tag" == "latest" ]]; then
    tags=("latest")
    [[ -n "$version" ]] && tags+=("v${version}")
  else
    tags=("$tag")
    local bare_tag="${tag#v}"
    [[ "$bare_tag" == "$version" ]] && tags+=("latest")
  fi

  local revision created
  revision="$(git_revision)"
  created="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  local label_args=(
    "--label" "org.opencontainers.image.version=${tags[0]#v}"
    "--label" "org.opencontainers.image.revision=${revision}"
    "--label" "org.opencontainers.image.created=${created}"
  )

  local registries=()
  [[ -n "${GHCR_REGISTRY:-}" ]]      && registries+=("GHCR:${GHCR_REGISTRY}")
  [[ -n "${DOCKERHUB_REGISTRY:-}" ]] && registries+=("Docker Hub:${DOCKERHUB_REGISTRY}")

  echo ""
  echo "=== Deploy ${IMAGE}:$(IFS=,; echo "${tags[*]}") ==="
  echo ""

  if [[ "${#registries[@]}" -eq 0 ]]; then
    local tag_args=()
    for t in "${tags[@]}"; do
      tag_args+=("-t" "${IMAGE}:${t}")
    done
    run docker buildx build \
      --platform "$PLATFORMS" \
      "${label_args[@]}" \
      "${tag_args[@]}" \
      --load \
      "$ROOT"
    echo ""
    echo "✓ Built $(IFS=,; echo "${tags[@]/#/${IMAGE}:}") (no registry set, skipping push)"
  else
    for entry in "${registries[@]}"; do
      local name="${entry%%:*}"
      local url="${entry#*:}"

      echo ""
      echo "--- Pushing to ${name} ---"

      local tag_args=()
      for t in "${tags[@]}"; do
        tag_args+=("-t" "${url}/${IMAGE}:${t}")
      done

      run docker buildx build \
        --platform "$PLATFORMS" \
        "${label_args[@]}" \
        "${tag_args[@]}" \
        --push \
        "$ROOT"

      for t in "${tags[@]}"; do
        echo "✓ Pushed ${url}/${IMAGE}:${t}"
      done
    done
    echo ""
    local registry_names=()
    for entry in "${registries[@]}"; do registry_names+=("${entry%%:*}"); done
    echo "✓ Deployed to $(IFS=" + "; echo "${registry_names[*]}")"
  fi

  if [[ "$tag" != "latest" ]]; then
    create_github_release "$tag"
  fi
}

deploy
