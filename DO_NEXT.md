# Do Next

Prioritized queue:

1. Make the real-Elm demo base-path aware for GitHub Pages: pass a base (the demo's path prefix) via flags and have `pageToPath`/`pageFromUrl` honor it, plus add a Pages SPA fallback (e.g. a `404.html`), so hard-refresh and deep-links on demo sub-routes work and the URL stays under `/demo/`. Today only in-app click navigation works (see BUGS.md).
2. Out-of-process session/rate-limiter store (Postgres). Put the MCP session store and the rate limiter behind interfaces (the in-memory implementations stay the default) and add Postgres-backed implementations: a `rate_limit_buckets` table (atomic token-bucket via a row-locked upsert) and an `mcp_sessions` table for session existence/eviction. The hard part is cross-process SSE replay fan-out, which needs `LISTEN/NOTIFY` (or a polling relay) — design that explicitly. This was scoped out of the collectible-tips/arcade PR to avoid shipping fragile pubsub.
5. Redesign anonymous worker identity and payout.
6. Add user-issued or organization-issued tokens.
7. Add crypto reward metadata.
8. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
9. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
