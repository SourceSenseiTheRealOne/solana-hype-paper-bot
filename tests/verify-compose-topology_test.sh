#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
guard="$repo_root/tests/verify-compose-topology.sh"
temporary_root="$(mktemp -d)"
mkdir -p "$temporary_root/tests" "$temporary_root/bin"
trap 'rm -rf "$temporary_root"' EXIT

cp "$guard" "$temporary_root/tests/verify-compose-topology.sh"
printf '%s\n' 'services:' '  api:' '    ports:' '      - 127.0.0.1:8080:8080' '  dashboard:' '    ports:' '      - 127.0.0.1:4173:8080' >"$temporary_root/compose.yaml"
printf '%s\n' '#!/usr/bin/env bash' 'exit 99' >"$temporary_root/bin/docker"
chmod +x "$temporary_root/bin/docker"

if ! (cd "$temporary_root" && PATH="$temporary_root/bin:$PATH" bash tests/verify-compose-topology.sh); then
	printf '%s\n' 'compose topology guard must inspect static compose topology without invoking Docker' >&2
	exit 1
fi
