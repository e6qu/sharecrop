APP := bin/sharecrop

.PHONY: build container check-contracts check-copy-paste check-dead-code check-format check-openapi check-policy check-ts check-wasi-bridge check-wasm-scenario-parity ci contracts css db-checks docker-down docker-up e2e-ui elm fmt frontend lint migrate-up openapi serve test test-deno test-go test-http test-integration vet wasi-app-guest wasi-bridge

build: frontend wasi-app-guest
	go build -o $(APP) ./cmd/sharecrop

# container builds the multi-arch backend image (manifest + per-arch tags) via
# tools/build_container.sh. Pass IMAGE=<repo:tag>; refresh web/static with
# `make frontend` first if the UI changed. See docs/deployment.md.
container:
	test -n "$(IMAGE)"
	tools/build_container.sh $(IMAGE)

# wasi-app-guest compiles the app guest to a wasip1 module and writes it to the
# embed path, so `build` bakes it into the sharecrop binary. WASI hosting is the
# default: `sharecrop serve` handles the dynamic routes by pooling this guest
# against Postgres — the same WASM artifact the browser demo runs. Set
# SHARECROP_WASI_MODE=native to run the in-process mux instead. The committed
# internal/wasiguest/app-guest.wasm is an empty placeholder this target replaces;
# do not commit the built (large) version.
wasi-app-guest:
	GOOS=wasip1 GOARCH=wasm go build -o internal/wasiguest/app-guest.wasm ./cmd/sharecrop-wasi-app-guest

contracts:
	go run ./cmd/sharecrop generate elm-contracts

check-contracts:
	go run ./cmd/sharecrop generate elm-contracts
	git diff --exit-code -- web/elm/src/Sharecrop/Generated

openapi:
	go run ./cmd/sharecrop generate openapi
	deno task site:openapi:copy

check-openapi:
	go run ./cmd/sharecrop generate openapi
	deno task site:openapi:copy
	git diff --exit-code -- docs/openapi.json site/docs/openapi.json

wasi-bridge:
	go run ./cmd/sharecrop generate wasi-bridge

check-wasi-bridge:
	go run ./cmd/sharecrop generate wasi-bridge
	git diff --exit-code -- 'internal/wasibridge/*bridge/bridge_gen.go'

check-format:
	test -z "$$(gofmt -l cmd internal tests web | grep -E '\\.go$$')"
	deno fmt --check deno.json tools tests

check-policy:
	deno task check:policy

check-ts:
	deno task check:ts

check-copy-paste:
	deno run -A npm:jscpd@5.0.11 site/demo internal cmd tools web/elm/src tests

check-dead-code:
	go tool deadcode -test ./...

check-wasm-scenario-parity:
	deno task wasm:demo:build
	deno task check:scenario-parity:wasm -- --wasm site/demo/sharecrop-wasm-backend.wasm

ci: check-format check-contracts check-openapi check-policy check-ts check-copy-paste check-dead-code check-wasi-bridge lint vet test frontend build test-integration test-http e2e-ui check-wasm-scenario-parity

css:
	deno task css:build

db-checks:
	./tools/run_db_checks.sh

docker-up:
	docker compose up -d postgres

docker-down:
	docker compose down

e2e-ui:
	deno task wasm:demo:build
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
