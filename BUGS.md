# Bugs

No confirmed application defects were known.

Test gaps:

- Pull request 7 generated task contracts cover compact task list items and capability-token responses. Full task detail form/view models remain handwritten Elm work because the current generator emits `Decode.mapN` decoders and should not be used for large product records.
- Pull request 7 added task API endpoints without visible task list or task detail screens. Browser coverage remains an app-shell smoke test until task user interface screens are added.
- Pull request 8 added submission API endpoints without visible submission review or anonymous submission screens. Browser coverage remains an app-shell smoke test until submission user interface screens are added.
- Pull request 9 added credit balance, ledger, and task funding screens but no submission review or task discovery screens. The task funding form takes a task identifier as text because there is no task list screen yet.
- Credit accounts and the ledger are modeled for user accounts only. Organization credit accounts, token ledgers, and crypto reward payouts are not implemented.
- Accepting an anonymous submission for a credit-reward task is rejected because anonymous submitters have no credit account. Wallet payout for anonymous workers is not implemented.
- The MCP surface implements the task, submission, and ledger tools. The `sharecrop.list_task_series` and `sharecrop.get_task_series` tools are not implemented because no task-series read API exists yet.
- The MCP endpoint handles single JSON-RPC requests; JSON-RPC batch requests are not implemented.
- The browser task list shows user-visible tasks only. Public task discovery and task detail screens beyond the agent curl examples are not built yet.
- Request/command contracts and HTTP contract fixture tests still need to expand as the API grows.

Known risks:

- The exact UUIDv7 library behavior had not been verified in code.
- `make build` passed locally with a non-fatal Go global module stat-cache warning from the sandbox.
