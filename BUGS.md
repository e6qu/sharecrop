# Bugs

No confirmed application defects were known.

Test gaps:

- The contract generator emits `Decode.mapN` decoders, so large product records such as task detail use handwritten Elm decoders instead of generated ones.
- Anonymous submission screens are not built. Anonymous submission remains API-only.
- The task funding form takes a task identifier as text; it is not yet wired to pick a task from the discovery or task list screens.
- Credit accounts and the ledger are modeled for user accounts only. Organization credit accounts, token ledgers, crypto reward payout, collectibles, and anonymous wallet payout remain future asset-economy work.
- Accepting an anonymous submission for a credit-reward task is rejected because anonymous submitters have no credit account.
- The MCP Streamable HTTP endpoint does not implement server-initiated SSE streams; a `GET` returns `405`, which the specification allows for a tools-only server.
- Request/command contracts and HTTP contract fixture tests still need to expand as the API grows.

Known risks:

- `make build` passed locally with a non-fatal Go global module stat-cache warning from the sandbox.
