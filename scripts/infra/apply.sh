#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
environment="${1:?staging or production required}"
[[ "$environment" == staging || "$environment" == production ]] || exit 1
root="infra/$environment"
fence() {
  if [[ "${GITHUB_ACTIONS:-}" == true ]]; then
    node scripts/ci/current-ref.mjs
  fi
}
: "${PG_CONN_STR:?}" "${TF_ENCRYPTION:?}" "${DIGITALOCEAN_ACCESS_TOKEN:?}"
: "${CLOUDFLARE_API_TOKEN:?}" "${UPSTASH_EMAIL:?}" "${UPSTASH_API_KEY:?}"
tofu -chdir="$root" init -input=false -lockfile=readonly
if [[ -z "${TF_VAR_image_digest:-}" ]]; then
  TF_VAR_image_digest="$(tofu -chdir="$root" output -raw image_digest)"
  export TF_VAR_image_digest
fi
python3 scripts/infra/domain-records.py "$root"
tofu -chdir="$root" plan -input=false -lock-timeout=120s -out=tfplan
fence
tofu -chdir="$root" apply -input=false tfplan
# Wildcard TXT tokens exist only after the first app/domain creation. Refresh
# them on every deployment and scheduled reconciliation for certificate renewal.
python3 scripts/infra/domain-records.py "$root" --wait
tofu -chdir="$root" plan -input=false -lock-timeout=120s -out=tfplan
fence
tofu -chdir="$root" apply -input=false tfplan
url="$(tofu -chdir="$root" output -raw url)"
for _ in $(seq 1 90); do
  if curl --fail --silent --show-error --max-time 10 "$url/health" >/dev/null 2>&1; then
    fence
    echo "$environment ready at $url"
    exit 0
  fi
  sleep 10
done
echo "$environment did not become healthy at $url" >&2
exit 1
