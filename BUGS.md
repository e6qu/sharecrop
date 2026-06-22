# Bugs

Confirmed defects:

- None known.

Test gaps:

- Anonymous workers were removed. The anonymous worker identity and payout model is deferred; submissions are registered-users-only.
- The task funding form takes a task identifier as text; it is not yet wired to pick a task from the discovery or task list screens.
- The asset economy is platform-only: user-issued tokens, organization-issued tokens, crypto rewards, and external wallets are not implemented. Rewards are Sharecrop credits and platform collectibles.
- The MCP Streamable HTTP endpoint does not implement server-initiated SSE streams; a `GET` returns `405`, which the specification allows for a tools-only server.
- Request/command contracts and HTTP contract fixture tests still need to expand as the API grows.

Known risks:

- None known.
