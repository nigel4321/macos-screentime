#!/usr/bin/env bash
# Smoke-test the deployed backend's /healthz endpoint.
#
# Usage:
#   ./scripts/smoke-deploy.sh https://macos-screentime-backend.fly.dev
#   ./scripts/smoke-deploy.sh                 # defaults to the URL above

set -euo pipefail

base_url="${1:-https://macos-screentime-backend.fly.dev}"
url="${base_url%/}/healthz"

echo "GET $url"

# -f: fail on >=400. -sS: silent but show errors. -w: append HTTP code.
response="$(curl -fsS -w '\n%{http_code}' "$url")"
body="$(printf '%s\n' "$response" | sed '$d')"
status="$(printf '%s\n' "$response" | tail -n1)"

if [[ "$status" != "200" ]]; then
  echo "smoke: unexpected status $status" >&2
  echo "body: $body" >&2
  exit 1
fi

# Healthz contract: {"status":"ok|degraded","database":"ok|unreachable|disabled"}.
# A 200 with status="degraded" still indicates a problem (typically DB
# unreachable), so smoke fails it explicitly.
if ! printf '%s' "$body" | grep -q '"status":"ok"'; then
  echo "smoke: status not ok: $body" >&2
  exit 1
fi

echo "smoke: ok ($status) — $body"
