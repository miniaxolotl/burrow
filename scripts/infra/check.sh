#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
expected="$(cat .opentofu-version)"
actual="$(tofu version -json | python3 -c 'import json,sys; print(json.load(sys.stdin)["terraform_version"])')"
[[ "$actual" == "$expected" ]] || { echo "OpenTofu $expected required" >&2; exit 1; }
tofu fmt -check -recursive infra
unset PG_CONN_STR
data_root="$(mktemp -d)"
trap 'rm -rf "$data_root"' EXIT
export TF_ENCRYPTION
TF_ENCRYPTION="$(python3 - <<'PY'
import secrets
print('key_provider "pbkdf2" "state" { passphrase = "' + secrets.token_hex(32) + '" }')
print('method "aes_gcm" "state" { keys = key_provider.pbkdf2.state }')
print('state { method = method.aes_gcm.state }')
print('plan { method = method.aes_gcm.state }')
PY
)"
for environment in staging production; do
  export TF_DATA_DIR="$data_root/$environment"
  tofu -chdir="infra/$environment" init -backend=false -input=false -lockfile=readonly
  tofu -chdir="infra/$environment" validate
done
