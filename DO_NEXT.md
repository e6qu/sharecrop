# Do Next

Prioritized queue:

1. Open pull request 10 and wait for continuous integration to pass.
2. After pull request 10 is merged, sync local `main` with `origin/main`.
3. Start the next task branch from synced `origin/main`.
4. Continue with user interface polish and shadcn-inspired Elm/Tailwind components from [PLAN.md](./PLAN.md#pr-11-ui-polish-and-shadcn-inspired-elmtailwind-components).
5. Add task-series read APIs and the remaining MCP tools (`list_task_series`, `get_task_series`), which were left out because no task-series read surface exists yet.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
