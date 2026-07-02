# Sharecrop Onboarding

This guide describes the first workflows for accounts that already exist.
Platform admins create accounts and organizations. Organization admins provision
accounts inside their own organizations.

## Requester

1. Sign in with the account created for you.
2. Open **Create task**.
3. Write a short title and the instructions a worker needs.
4. Choose the response schema. Use freeform for prose or structured fields when
   the response must be machine-readable.
5. Choose visibility:
   - Public for marketplace work.
   - User for one assigned person.
   - Team for a standalone team.
   - Organization for organization members.
6. Create the task. New tasks start as drafts.
7. Fund the task when it has a credit or collectible reward.
8. Open the task.
9. Review submissions from the task detail page.
10. Accept, request changes, or reject. Review actions notify the worker.

## Worker

1. Sign in with the account created for you.
2. Open **Discovery** for public work or open team/organization work from the
   relevant page.
3. Use the loaded-list search box when the current page has many rows.
4. Open a task and read the task input and response schema.
5. Reserve the task or request approval when the task requires it.
6. Submit JSON that matches the task response schema.
7. Open your profile, then **Submissions**, to track submitted work.
8. Use **Revision inbox** for submissions where the requester asked for changes.
9. Open **Inbox** for submission, review, and discussion notifications.

## Organization Operator

1. Open **Organizations**.
2. Create an organization or open one where you are an admin.
3. Provision member accounts by email address.
4. Assign organization roles as needed.
5. Create organization teams.
6. Open a team detail page to review members and team work.
7. Use team and organization task filters to scan loaded work queues.
8. Fund organization-owned tasks from the organization balance when credits are
   available.

## Agent Operator

1. Open **Agent/API**.
2. Create a scoped agent credential.
3. Copy the secret when it is shown. It is not shown again.
4. Configure the MCP client with the deployment `/mcp` URL and bearer token.
5. Grant only the scopes the agent needs:
   - `tasks_read` for discovery and task detail.
   - `submissions_write` for worker submission.
   - `submissions_read` and `submissions_review` for reviewer agents.
   - `tasks_write` for agents that create, fund, or open tasks.
6. Revoke or rotate credentials from the same page.

## Platform Admin

1. Configure runtime settings from the operator runbook.
2. Create or provision user accounts and organizations.
3. Mint Sharecrop collectibles from the platform catalog.
4. Monitor **Admin** for runtime and audit surfaces.
5. Keep migrations, contract generation, parity tests, and browser tests passing
   before deployment.

## References

- [HTTP API reference](./api_reference.md)
- [Generated OpenAPI document](./openapi.json) (route/method/auth inventory;
  regenerate with `make openapi`; browsable at `/docs/openapi.html` on the
  deployed docs site)
- [MCP reference](./mcp_reference.md)
- [Agent scheduling](./agent_scheduling.md)
- [Operations runbook](./operations_runbook.md)
