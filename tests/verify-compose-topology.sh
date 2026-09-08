#!/usr/bin/env bash
set -euo pipefail

compose_file="${COMPOSE_FILE:-compose.yaml}"
if [ ! -f "$compose_file" ]; then
	printf '%s\n' 'compose topology file is missing' >&2
	exit 1
fi

config="$(<"$compose_file")"
services=()
in_services=false
while IFS= read -r line; do
	if [ "$line" = 'services:' ]; then
		in_services=true
		continue
	fi
	if [ "$in_services" = true ] && [ -n "$line" ] && [[ "$line" != ' '* ]]; then
		break
	fi
	if [ "$in_services" = true ] && [[ "$line" =~ ^\ \ ([A-Za-z0-9_-]+):[[:space:]]*$ ]]; then
		services+=("${BASH_REMATCH[1]}")
	fi
done <<<"$config"

if [ "${#services[@]}" -ne 2 ] || [ "${services[0]}" != 'api' ] || [ "${services[1]}" != 'dashboard' ]; then
	printf '%s\n' 'compose topology must define exactly api and dashboard services' >&2
	exit 1
fi
if [ "$(printf '%s\n' "$config" | grep -Fc '127.0.0.1:')" -ne 2 ]; then
  printf '%s\n' 'compose must publish exactly two loopback-only application ports' >&2
  exit 1
fi

printf '%s\n' 'PASS: compose has loopback API/dashboard services and no competing database'
