# Bugs

Confirmed defects:

- None known.

Test gaps:

- Anonymous workers were removed. The anonymous worker identity and payout model is deferred; submissions are registered-users-only.
- Browser task creation supports user assignees but does not yet expose organization-team assignee selection.
- Browser task detail pages show task-specific REST and MCP examples, but MCP workflow tools for reservation and approval are not implemented yet.
- Review tips are credit-only. Collectible or inventory-based tips remain deferred.
- Reservation availability is exposed on task responses, but the current read model does not yet expose the active reservation assignee on each task list item.
- The asset economy is platform-only: user-issued tokens, organization-issued tokens, crypto rewards, and external wallets are not implemented. Current implemented rewards are Sharecrop credits and platform collectibles.
- The MCP Streamable HTTP endpoint does not implement full server-initiated SSE streams yet; full SSE is planned.
- MCP workflow tools for request changes, rejection, partial payout, credit tips, and task-local bans are implemented. MCP reservation and approval workflow tools are still not implemented.
- Request/command contracts and HTTP contract fixture tests still need to expand as the API grows.
- Manual screenshot review for reward-bundle browser labels was skipped; Playwright UI coverage passed on the branch.

Known risks:

- None known.
