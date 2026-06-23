# Do Next

Prioritized queue:

1. Verify GitHub Pages deploys the task-detail navigation demo after the demo pull requests are merged.
2. Finish organization context in the browser: organization switcher, organization-scoped task and credit views, member and team management screens, and organization-owned task creation. The browser currently lists organizations and creates them; the rest is not built.
3. Add standalone (user-owned) teams. Teams are organization-only. A first attempt modeled the team owner as a tagged union and added a migration, but it was reverted because the store and HTTP layers were left inconsistent. Redo it as a focused change: domain tagged union for the team owner, store methods, HTTP create/list endpoints, and tests.
4. Continue decomposing the HTTP and browser monoliths. `internal/http/server.go` and `web/elm/src/Main.elm` are still large. The auth handlers were extracted into `internal/http/auth_handlers.go`; extract further cohesive handler groups and split `Main.elm` view and API layers into modules without behavior change.
5. Expose collectible reward counts and the held collectible set in the browser so the multi-collectible reward feature is visible in the UI.
6. Revisit collectible or inventory-based tips.
7. Redesign anonymous worker identity and payout.
8. Add user-issued or organization-issued tokens.
9. Add crypto reward metadata.
10. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store now evicts idle sessions after a TTL but is still per-process.
11. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
