# Bugs

Confirmed defects:

- None known.

Test gaps:

- GitHub Pages deployment cannot be observed from pull request CI because the Pages workflow publishes after pushes to `main` or manual dispatch.
- Anonymous workers were removed. The anonymous worker identity and payout model is deferred; submissions are registered-users-only.
- Browser task creation supports user assignees but does not yet expose organization-team assignee selection.
- Review tips are credit-only. Collectible or inventory-based tips remain deferred.
- Reservation availability is exposed on task responses, but the current read model does not yet expose the active reservation assignee on each task list item.
- The asset economy is platform-only: user-issued tokens, organization-issued tokens, crypto rewards, and external wallets are not implemented. Current implemented rewards are Sharecrop credits and platform collectibles.
- Request/command contracts and HTTP contract fixture tests still need to expand as the API grows.
- Manual screenshot review for reward-bundle browser labels was skipped; Playwright UI coverage passed on the branch.
- Manual screenshot review for MCP instruction snippet changes was skipped; Playwright UI coverage passed on the branch.

Known risks:

- MCP HTTP sessions and SSE replay buffers are in-memory. They are not shared across server restarts or multiple app processes.
