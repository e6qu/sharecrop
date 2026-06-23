# Do Next

Prioritized queue:

1. Verify GitHub Pages deploys the task-detail navigation demo after the demo pull requests are merged.
2. Finish the remaining organization-context gaps in the browser. The browser now switches organizations, shows the organization credit balance and organization-scoped task list, creates teams, provisions members, and creates organization-owned tasks. Still missing: a member listing view (no `GET` members endpoint exists yet; add the endpoint and the view), funding an organization-owned task from the organization credit account through the browser, and organization or team visibility and team-assignee selection in the task creation form.
3. Wire standalone (user-owned) teams into task assignee and visibility logic. Standalone teams can be created and listed; they are not yet selectable as a task assignee or visibility scope.
4. Continue decomposing the HTTP and browser monoliths. `internal/http/server.go` and `web/elm/src/Main.elm` are still large. Auth handlers were extracted into `internal/http/auth_handlers.go`; extract further cohesive handler groups and split `Main.elm` into view and API modules without behavior change.
5. Revisit collectible or inventory-based tips.
6. Redesign anonymous worker identity and payout.
7. Add user-issued or organization-issued tokens.
8. Add crypto reward metadata.
9. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
10. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
