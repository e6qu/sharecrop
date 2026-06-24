# Do Next

Prioritized queue:

1. Demo UI/UX minors deferred from the review: unify the two neutral-chip styles (list `.mission-meta span` vs detail `.badge-row span`) into one shared class so attributes read as one system; replace the `site/docs` "Documentation placeholder" with a minimal real quickstart (post → fund → reserve → submit → settle, plus the REST/MCP tool list the Agent/API console already shows); and add a lightweight worker/agent trust signal (e.g. a per-persona completed-tasks stat near the submitter) so authorizing an external agent has a credibility cue.
2. Security follow-ups (see [BUGS.md](./BUGS.md)): add per-subject/per-IP rate limiting (HTTP 429) on MCP tool calls and the unauthenticated receipt/login/refresh endpoints; add a derived `idempotency_key` to the two `task_tip` ledger inserts for consistency with payout/refund entries.
3. Revisit collectible or inventory-based tips.
5. Redesign anonymous worker identity and payout.
6. Add user-issued or organization-issued tokens.
7. Add crypto reward metadata.
8. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
9. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
