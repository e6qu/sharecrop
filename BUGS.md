# Bugs

Confirmed defects:

- None known.

Test gaps:

- Anonymous workers were removed. The anonymous worker identity and payout model is deferred; submissions are registered-users-only.
- Reward bundles are planned but not implemented; current rewards are no reward, credits, or platform collectibles through separate flows.
- Browser task creation supports user assignees but does not yet expose organization-team assignee selection.
- Browser task detail pages show task-specific REST and MCP examples, but MCP workflow tools for reservation and approval are not implemented yet.
- Task-local implementor bans have storage but do not yet have requester review actions that write bans.
- Request-changes, partial rewards, and tips are planned but not implemented.
- Reservation availability is exposed on task responses, but the current read model does not yet expose the active reservation assignee on each task list item.
- The asset economy is platform-only: user-issued tokens, organization-issued tokens, crypto rewards, and external wallets are not implemented. Current implemented rewards are Sharecrop credits and platform collectibles.
- The MCP Streamable HTTP endpoint does not implement full server-initiated SSE streams yet; full SSE is planned.
- MCP workflow tools for reservation, approval, request changes, rejection, partial payout, tips, and task-local bans are not implemented yet.
- Request/command contracts and HTTP contract fixture tests still need to expand as the API grows.

Known risks:

- None known.
