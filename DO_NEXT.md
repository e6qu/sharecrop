# Do Next

Prioritized queue:

1. Continue decomposing the browser monolith. `web/elm/src/Main.elm` is still large even after extracting `Sharecrop.Labels`; split it into view, update, and API/command modules without behavior change. On the HTTP side, organization, team, funding, user, series, and credits handlers now live in their own files; the remaining `server.go` is mostly task, submission, and reservation handlers that could split next.
2. Add a standalone-team membership flow. Organization teams add members through provisioning, but standalone (user-owned) teams have no way to add members yet, so their rosters are always empty.
3. Revisit collectible or inventory-based tips.
5. Redesign anonymous worker identity and payout.
6. Add user-issued or organization-issued tokens.
7. Add crypto reward metadata.
8. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
9. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
