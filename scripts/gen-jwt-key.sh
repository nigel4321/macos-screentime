#!/usr/bin/env bash
# Generates a fresh ES256 signing key (EC P-256) and prints it as a
# PEM. Pipe into `fly secrets set` to wire it into the deployed app:
#
#   ./scripts/gen-jwt-key.sh | flyctl secrets set --app macos-screentime-backend JWT_SIGNING_KEY=-
#
# The auth package accepts both SEC1 ("EC PRIVATE KEY") and PKCS#8
# ("PRIVATE KEY") PEMs; openssl's default output is SEC1.

set -euo pipefail
exec openssl ecparam -name prime256v1 -genkey -noout
