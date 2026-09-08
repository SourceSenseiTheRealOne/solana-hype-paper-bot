#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
guard="$repo_root/tests/verify-github-actions.sh"
workflow="$repo_root/.github/workflows/ci.yml"
temporary_root="$(mktemp -d)"
trap 'rm -rf "$temporary_root"' EXIT

native_tool_path() {
	local path="$1"
	local directory
	directory="$(dirname "$path")"
	local basename
	basename="$(basename "$path")"
	local native_directory
	if native_directory="$(cd "$directory" && pwd -W 2>/dev/null)"; then
		printf '%s/%s\n' "$native_directory" "$basename"
		return
	fi
	printf '%s\n' "$path"
}

make_fixture() {
	local name="$1"
	local fixture="$temporary_root/$name"
	mkdir -p "$fixture/tests" "$fixture/.github/workflows"
	cp "$guard" "$fixture/tests/verify-github-actions.sh"
	cp "$workflow" "$fixture/.github/workflows/ci.yml"
	local git_fixture
	git_fixture="$(native_tool_path "$fixture")"
	git -C "$git_fixture" init -q
	git -C "$git_fixture" config user.email test@example.invalid
	git -C "$git_fixture" config user.name test
	git -C "$git_fixture" add .
	git -C "$git_fixture" commit -qm fixture
	fixture_path="$fixture"
}

replace_once() {
	python3 - "$(native_tool_path "$1")" "$2" "$3" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
old = sys.argv[2]
new = sys.argv[3]
text = path.read_text(encoding="utf-8")
if text.count(old) != 1:
    raise SystemExit(f"expected exactly one occurrence of {old!r}")
path.write_text(text.replace(old, new, 1), encoding="utf-8")
PY
}

run_guard() {
	(cd "$1" && bash tests/verify-github-actions.sh)
}

make_fixture baseline
baseline="$fixture_path"
run_guard "$baseline"

make_fixture pull-request-target
pull_request_target="$fixture_path"
replace_once "$pull_request_target/.github/workflows/ci.yml" 'pull_request:' 'pull_request_target:'
if run_guard "$pull_request_target"; then
	printf '%s\n' 'CI guard accepted pull_request_target' >&2
	exit 1
fi

make_fixture secret-reference
secret_reference="$fixture_path"
printf '%s\n' '# secrets.example must never be used by public CI' >>"$secret_reference/.github/workflows/ci.yml"
if run_guard "$secret_reference"; then
	printf '%s\n' 'CI guard accepted a secret reference' >&2
	exit 1
fi

make_fixture indexed-secret-reference
indexed_secret_reference="$fixture_path"
printf '%s\n' '# ${{ secrets["example"] }} must never be used by public CI' >>"$indexed_secret_reference/.github/workflows/ci.yml"
if run_guard "$indexed_secret_reference"; then
	printf '%s\n' 'CI guard accepted an indexed secret reference' >&2
	exit 1
fi

make_fixture compose-render
compose_render="$fixture_path"
printf '%s\n' '# docker compose config must never render dotenv data' >>"$compose_render/.github/workflows/ci.yml"
if run_guard "$compose_render"; then
	printf '%s\n' 'CI guard accepted dotenv-rendering Compose configuration' >&2
	exit 1
fi

make_fixture mutable-action
mutable_action="$fixture_path"
replace_once "$mutable_action/.github/workflows/ci.yml" \
	'actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e' \
	'actions/setup-go@v7'
if run_guard "$mutable_action"; then
	printf '%s\n' 'CI guard accepted a mutable action tag' >&2
	exit 1
fi

make_fixture actionlint-version
actionlint_version="$fixture_path"
replace_once "$actionlint_version/.github/workflows/ci.yml" \
	'go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12' \
	'go install github.com/rhysd/actionlint/cmd/actionlint@v0.0.0'
if run_guard "$actionlint_version"; then
	printf '%s\n' 'CI guard accepted an unreviewed actionlint version' >&2
	exit 1
fi

make_fixture duplicate-action
duplicate_action="$fixture_path"
replace_once "$duplicate_action/.github/workflows/ci.yml" \
	'      - name: Set up Go
        uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0' \
	'      - name: Duplicate setup Go fixture
        uses: actions/setup-go@0000000000000000000000000000000000000000 # fixture

      - name: Set up Go
        uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0'
if run_guard "$duplicate_action"; then
	printf '%s\n' 'CI guard accepted an unreviewed duplicate action pin' >&2
	exit 1
fi
