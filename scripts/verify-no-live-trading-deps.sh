#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

forbidden_pattern='@solana/wallet-adapter|@solana/web3\.js|solana-go|solana-sdk|wallet-adapter|seed[[:space:]_-]*phrase|private[[:space:]_-]*key|sign(Transaction|AllTransactions|Message)|send(Raw)?Transaction|swap(Transaction|Instructions)|jupiter[^[:space:]]*(swap|execute)|/swap/v[0-9]+/(build|execute|order|swap|transaction|instructions)|(^|[^[:alnum:]_])(func|function)[[:space:]]+(\([^)]*\)[[:space:]]+)?(BuildSwap|CreateTrigger|SendTransaction|Sign)[[:space:]]*\('

declare -a files=()
while IFS= read -r -d '' path; do
  case "$path" in
    go.mod|go.sum|*/package.json|*.go|*.ts|*.tsx|*.js|*.mjs|*.cjs)
      files+=("$path")
      ;;
  esac
done < <(git ls-files -z --cached --others --exclude-standard)

if ((${#files[@]} == 0)); then
  printf 'PASS: no source or manifest files to inspect\n'
  exit 0
fi

if matches=$(grep -EIn -- "$forbidden_pattern" "${files[@]}" 2>/dev/null); then
  printf 'FAIL: forbidden live-trading dependency or capability detected:\n%s\n' "$matches" >&2
  exit 1
fi

printf 'PASS: no wallet, signing, transaction-send, or Jupiter execution dependencies found\n'
