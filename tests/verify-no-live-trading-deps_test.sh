#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
guard="$repo_root/scripts/verify-no-live-trading-deps.sh"
temporary_root="$(cygpath -m "$LOCALAPPDATA")/Temp/verify-no-live-trading-deps-test-$$"
mkdir -p "$temporary_root"
trap 'rm -rf "$temporary_root"' EXIT

run_guard() {
  local fixture="$1"
  local fixture_dir="$temporary_root/$fixture"
  mkdir -p "$fixture_dir/scripts"
  cp "$guard" "$fixture_dir/scripts/verify-no-live-trading-deps.sh"
  git -C "$fixture_dir" init -q
  git -C "$fixture_dir" config user.email test@example.invalid
  git -C "$fixture_dir" config user.name test
  printf '%s\n' "$2" >"$fixture_dir/example.go"
  git -C "$fixture_dir" add .
  git -C "$fixture_dir" commit -qm fixture
  (cd "$fixture_dir" && bash scripts/verify-no-live-trading-deps.sh)
}

run_guard quote 'package fixture
const quoteEndpoint = "/swap/v1/quote"'

run_guard read_only_sign 'package fixture
func report(r interface{ Sign() int }) int { return r.Sign() }'

if run_guard transaction 'package fixture
const buildEndpoint = "/swap/v1/build"'; then
	printf 'guard accepted a Jupiter transaction-construction endpoint\n' >&2
	exit 1
fi

if run_guard build_swap 'package fixture
func BuildSwap() {}'; then
	printf 'guard accepted a swap-construction capability\n' >&2
	exit 1
fi

if run_guard send_transaction 'package fixture
func SendTransaction() {}'; then
	printf 'guard accepted a transaction-send capability\n' >&2
	exit 1
fi

if run_guard sign 'package fixture
func Sign() {}'; then
	printf 'guard accepted a signing capability\n' >&2
	exit 1
fi

if run_guard create_trigger 'package fixture
func CreateTrigger() {}'; then
	printf 'guard accepted an order-trigger capability\n' >&2
	exit 1
fi
