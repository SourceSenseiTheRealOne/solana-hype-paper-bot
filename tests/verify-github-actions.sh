#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
workflow="$repo_root/.github/workflows/ci.yml"

if [[ ! -f "$workflow" ]]; then
	printf '%s\n' 'FAIL: GitHub Actions CI workflow is missing' >&2
	exit 1
fi

python3 - "$workflow" <<'PY'
from pathlib import Path
import re
import sys

workflow = Path(sys.argv[1])
text = workflow.read_text(encoding="utf-8")

required = (
    "pull_request:",
    "push:",
    "- development",
    "- staging",
    "- main",
    "permissions:",
    "contents: read",
    "pull-requests: read",
    "persist-credentials: false",
    "go-version-file: go.mod",
    "go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12",
    "$(go env GOPATH)/bin/actionlint -color",
    "node-version: '22'",
    "corepack enable",
    "pnpm --dir dashboard install --frozen-lockfile",
    "postgres:16-alpine",
    "POSTGRES_HOST_AUTH_METHOD: trust",
    "psql",
    "TEST_DATABASE_URL",
    "go test ./...",
    "go test -race ./...",
    "go vet ./...",
    "go build ./cmd/paper-bot",
    "pnpm --dir dashboard run verify",
    "bash scripts/verify-no-live-trading-deps.sh",
    "bash tests/verify-no-live-trading-deps_test.sh",
    "bash tests/verify-compose-topology.sh",
    "bash tests/verify-compose-topology_test.sh",
    "bash tests/verify-dashboard-proxy.sh",
    "bash tests/verify-github-actions.sh",
    "bash tests/verify-github-actions_test.sh",
    "git diff --check",
)
for token in required:
    if token not in text:
        raise SystemExit(f"FAIL: CI workflow is missing required token: {token}")

forbidden = (
    "pull_request_target:",
    ".env.local",
    "docker compose config",
    "docker-compose config",
    "GITHUB_TOKEN:",
)
for token in forbidden:
    if token in text:
        raise SystemExit(f"FAIL: CI workflow contains forbidden token: {token}")

if re.search(r"\bsecrets\s*(?:\.|\[)", text):
    raise SystemExit("FAIL: CI workflow contains a forbidden secret expression")

expected_actions = (
    "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1",
    "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",
    "actions/setup-node@820762786026740c76f36085b0efc47a31fe5020",
    "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1",
    "actions/dependency-review-action@a1d282b36b6f3519aa1f3fc636f609c47dddb294",
)
seen_actions = []
for line in text.splitlines():
    stripped = line.strip()
    if not stripped.startswith("uses:"):
        continue
    reference = stripped.removeprefix("uses:").strip().split("#", 1)[0].strip()
    if "@" not in reference:
        raise SystemExit(f"FAIL: CI action reference is missing an immutable SHA: {reference}")
    _, ref = reference.rsplit("@", 1)
    if len(ref) != 40 or any(character not in "0123456789abcdef" for character in ref):
        raise SystemExit(f"FAIL: CI action reference is not a full lowercase commit SHA: {reference}")
    seen_actions.append(reference)

if tuple(seen_actions) != expected_actions:
    raise SystemExit(
        "FAIL: CI action allowlist or immutable pins differ from the reviewed set: "
        f"{seen_actions}"
    )

print("PASS: GitHub Actions CI workflow is public-safe, SHA-pinned, and enforces required gates")
PY
