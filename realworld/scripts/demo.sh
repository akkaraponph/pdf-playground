#!/usr/bin/env bash
set -euo pipefail

command -v curl >/dev/null
command -v jq >/dev/null

repo_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
base="${PDFSIGN_BASE_URL:-http://127.0.0.1:3000}/api/v1"
output="${PDFSIGN_OUTPUT_DIR:-$repo_dir/realworld/data/demo-output}"
mkdir -p "$output"

api() {
  curl --fail-with-body --silent --show-error "$@"
}

wait_for() {
  local wanted=$1 metadata status
  for ((attempt=0; attempt<120; attempt++)); do
    metadata=$(api "$base/documents/$id")
    status=$(jq -r '.status' <<<"$metadata")
    case "$status" in
      "$wanted") printf '%s\n' "$metadata"; return 0 ;;
      generation_failed|signing_failed|rejected)
        printf 'Task stopped: %s\n' "$metadata" >&2; return 1 ;;
    esac
    sleep 1
  done
  printf 'Timed out waiting for %s\n' "$wanted" >&2
  return 1
}

created=$(api -X POST "$base/documents/generate" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Professional services agreement","customer":"Acme Ltd","amount_cents":125050,"currency":"USD"}')
id=$(jq -er '.id' <<<"$created")
printf 'Document: %s\n' "$id"

wait_for draft | jq '{id,status,original_sha256}'
api "$base/documents/$id/download?version=original" -o "$output/$id-original.pdf"
api -X POST "$base/documents/$id/submit" | jq '{id,status}'
api -X POST "$base/documents/$id/approve" \
  -H 'Content-Type: application/json' \
  -d '{"reason":"Reviewed contract terms and approved for signing"}' | jq '{id,status}'
wait_for signed | jq '{id,status,original_sha256,signed_sha256,events}'
api "$base/documents/$id/download" -o "$output/$id-signed.pdf"
verification=$(api "$base/documents/$id/verify")
jq -e '.valid == true and .trusted == true and .signatures == 1' <<<"$verification" >/dev/null
jq . <<<"$verification"
printf 'Original: %s\nSigned: %s\n' "$output/$id-original.pdf" "$output/$id-signed.pdf"
