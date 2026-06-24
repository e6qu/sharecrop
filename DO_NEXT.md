# Do Next

Prioritized queue:

1. Finish decomposing the `Main.elm` monolith: the shared `Sharecrop.Types` module is now extracted; next, split the view functions into a `View` module and the HTTP commands into an `Api`/`Commands` module (their dependence on `Msg`/`Model` now resolves through `Types`, breaking the previous cycle). On the HTTP side, `server.go` is mostly task/submission/reservation handlers that could split next.
2. Minor demo review follow-ups deferred from the polish pass: the seed dashboard shows Credits available beside Held in escrow which can read as double-counting (present a single total or pre-deduct so it is unambiguous); pre-fill the review Settle/Tip inputs with their real defaults in normal text rather than muted placeholders; and cap or warn when a settle payout+tip exceeds the escrowed amount instead of silently clamping.
3. Smaller schema-designer follow-ups: array-length constraints, and normalizing field names to identifier-safe keys (the designer warns on duplicate/empty names but does not rewrite them).
4. Revisit collectible or inventory-based tips.
5. Redesign anonymous worker identity and payout.
6. Add user-issued or organization-issued tokens.
7. Add crypto reward metadata.
8. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
9. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
