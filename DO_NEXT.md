# Do Next

Prioritized queue:

1. Continue decomposing the HTTP and browser monoliths. `internal/http/server.go` and `web/elm/src/Main.elm` are still large. Auth handlers are in `internal/http/auth_handlers.go`, series handlers in `series.go`, and user-profile/work/submissions handlers in `users.go`; extract further cohesive handler groups (organizations, funding) and split `Main.elm` into view and API modules without behavior change.
2. Add team-assignee selection (organization-team and standalone-team) to the task creation form. Team and organization visibility scopes are wired; assignee scope is still user-only in the browser form.
3. Surface a team page and a team roster in the browser (standalone and organization teams are created and listed but have no detail page).
4. Revisit collectible or inventory-based tips.
5. Redesign anonymous worker identity and payout.
6. Add user-issued or organization-issued tokens.
7. Add crypto reward metadata.
8. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
9. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
