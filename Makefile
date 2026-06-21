APP := bin/sharecrop

.PHONY: build css docker-down docker-up e2e-ui elm frontend migrate-up serve test test-deno test-go test-http

build: frontend
	go build -o $(APP) ./cmd/sharecrop

css:
	deno task css:build

docker-up:
	docker compose up -d postgres

docker-down:
	docker compose down

e2e-ui:
	deno task e2e:ui

elm:
	deno task elm:build

frontend:
	deno task frontend:build

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
