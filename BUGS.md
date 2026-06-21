# Bugs

No confirmed application defects were known.

Test gaps:

- Docker Compose Postgres startup was not verified because the environment rejected the required Docker approval request.
- `sharecrop migrate up` against a live Postgres database was not verified for the same reason.
- Final rerun of `deno task e2e:ui` was not performed because local-network/browser permissions had already been exhausted in this environment after an earlier successful run.
- `make build` with both `GOCACHE` and `GOMODCACHE` isolated inside the workspace could not fetch `pgx` because network access was restricted. The build had passed earlier with the existing module cache.

Known risks:

- The local Sharecrop schema parser had not been implemented.
- The Go-to-Elm contract generator had not been implemented.
- The exact UUIDv7 library behavior had not been verified in code.
- The migration runner had not been exercised against a live database in this environment.
