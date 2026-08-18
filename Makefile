.PHONY: test test-race vet dashboard-verify verify

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

dashboard-verify:
	pnpm --dir dashboard run verify

verify: test test-race vet dashboard-verify
	bash scripts/verify-no-live-trading-deps.sh
	docker compose config
