# Sharecrop

Sharecrop is a coordination layer for requested work, submissions, validation, scoped access, and rewards.

See:

- [PLAN.md](./PLAN.md)
- [AGENTS.md](./AGENTS.md)
- [STATUS.md](./STATUS.md)
- [DO_NEXT.md](./DO_NEXT.md)
- [BUGS.md](./BUGS.md)
- [docs/operations_runbook.md](./docs/operations_runbook.md)
- [docs/api_reference.md](./docs/api_reference.md)
- [docs/mcp_reference.md](./docs/mcp_reference.md)
- [docs/agent_scheduling.md](./docs/agent_scheduling.md)

## Local Commands

```sh
export ELM_BIN=/opt/homebrew/bin/elm
export SHARECROP_HTTP_ADDR=:18080
export SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901
export DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable
export SHARECROP_MIGRATIONS_DIR=$PWD/migrations
make build
make test
make serve
make migrate-up
```

The default app address is `http://127.0.0.1:18080`.

Local Postgres is configured through Docker Compose on port `15432`.
