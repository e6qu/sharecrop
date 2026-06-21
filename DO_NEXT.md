# Do Next

Prioritized queue:

1. Verify Docker Compose Postgres and `sharecrop migrate up` when the environment allows Docker approval.
2. Start PR 2 from [PLAN.md](./PLAN.md#pr-2-core-domain-type-system).
3. Add strong ID wrappers.
4. Add UUIDv7 generation behind `internal/core/id`.
5. Add domain error types.
6. Add per-type result patterns.
7. Add strong enum and tagged-union examples.
8. Add tests for ID parsing, enum parsing, result handling, and lifecycle transitions.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
