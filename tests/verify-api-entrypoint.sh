#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
dockerfile="$root/Dockerfile"
entrypoint="$root/scripts/docker-entrypoint.sh"

require() {
  local pattern="$1"
  local file="$2"
  local description="$3"
  if ! grep -Fqx -- "$pattern" "$file"; then
    printf 'FAIL: %s\n' "$description" >&2
    exit 1
  fi
}

if [ ! -f "$entrypoint" ]; then
  printf '%s\n' 'FAIL: API entrypoint script is missing' >&2
  exit 1
fi

require 'COPY --chmod=755 scripts/docker-entrypoint.sh /app/docker-entrypoint.sh' "$dockerfile" 'Dockerfile must install the executable API entrypoint'
require 'ENTRYPOINT ["/app/docker-entrypoint.sh"]' "$dockerfile" 'Dockerfile must use the API entrypoint'
require 'install -d -m 0750 -o app -g app /app/var/log /app/var/reports' "$entrypoint" 'entrypoint must initialize only bounded runtime directories'
require 'exec su-exec app:app /app/paper-bot' "$entrypoint" 'entrypoint must drop privileges before starting API'

printf '%s\n' 'PASS: API entrypoint bounds runtime-volume ownership setup and drops to app'
