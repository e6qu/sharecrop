APP := bin/sharecrop

.PHONY: build check-contracts check-copy-paste check-dead-code check-format check-policy check-ts ci contracts css docker-down docker-up e2e-ui elm fmt frontend lint migrate-up serve test test-deno test-go test-http test-integration vet

build: frontend
	go build -o $(APP) ./cmd/sharecrop

contracts:
	go run ./cmd/sharecrop generate elm-contracts

check-contracts:
	go run ./cmd/sharecrop generate elm-contracts
	git diff --exit-code -- web/elm/src/Sharecrop/Generated

check-format:
	test -z "$$(gofmt -l cmd internal tests web | grep -E '\\.go$$')"
	deno fmt --check deno.json tools tests

check-policy:
	deno task check:policy

check-ts:
	deno task check:ts

check-copy-paste:
	deno run --allow-read tools/check_copy_paste.ts

check-dead-code:
	go tool deadcode -test ./...

ci: check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test frontend build test-integration test-http e2e-ui

css:
	deno task css:build

docker-up:
	docker compose up -d postgres

docker-down:
	docker compose down

e2e-ui:
	deno task e2e:ui

elm:
	test -n "$(ELM_BIN)"
	ELM_BIN=$(ELM_BIN) deno task elm:build

fmt:
	gofmt -w cmd internal tests web
	deno fmt deno.json tools tests

frontend: contracts
	test -n "$(ELM_BIN)"
	ELM_BIN=$(ELM_BIN) deno task frontend:build

lint:
	deno task lint

migrate-up:
	go run ./cmd/sharecrop migrate up

serve:
	go run ./cmd/sharecrop serve

test: test-go test-deno

test-deno:
	deno task test

test-go:
	go test ./...

test-http:
	go test -tags http_e2e ./tests/http_e2e

test-integration:
	go test -tags integration ./tests/integration

vet:
	go vet ./...
