#!/usr/bin/env bash
set -euo pipefail

config='dashboard/nginx.conf'

if ! grep -Fqx '    resolver 127.0.0.11 ipv6=off valid=10s;' "$config"; then
  printf '%s\n' 'dashboard proxy must use Docker dynamic DNS resolution' >&2
  exit 1
fi
if [ "$(grep -Fc '        proxy_pass http://$api_upstream:8080;' "$config")" -ne 2 ]; then
  printf '%s\n' 'dashboard proxy must dynamically resolve both API locations' >&2
  exit 1
fi

printf '%s\n' 'PASS: dashboard proxy dynamically resolves the API upstream'
