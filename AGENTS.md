# Agent Instructions

This file defines how AI agents should work in this repository.

Start here, then use the continuity files:

- [PLAN.md](./PLAN.md)
- [STATUS.md](./STATUS.md)
- [WHAT_WE_DID.md](./WHAT_WE_DID.md)
- [DO_NEXT.md](./DO_NEXT.md)
- [BUGS.md](./BUGS.md)

## Continuity Files

This project uses continuity files so work can survive context loss, compaction, branch changes, and fresh AI-agent sessions.

The continuity files are:

- [PLAN.md](./PLAN.md): product plan, architecture, stack, modeling rules, roadmap.
- [STATUS.md](./STATUS.md): current project status and active work state.
- [WHAT_WE_DID.md](./WHAT_WE_DID.md): completed work, written in past tense.
- [DO_NEXT.md](./DO_NEXT.md): next tasks to pick up.
- [BUGS.md](./BUGS.md): known defects, risks, regressions, and test gaps.

Update these files before each task and after each task.

Before a task:

- Read the continuity files.
- Remove stale or irrelevant content when it no longer helps.
- Update the files so they describe the task as it is currently understood.
- Keep the wording timeless. Do not narrate the history of how the task was discovered.

After a task:

- Update the continuity files again.
- Write completed work in past tense.
- Keep the files accurate after the task branch is merged into `main`.
- Record bugs, test gaps, and remaining work.
- Remove side threads that are no longer relevant.

The continuity files should be useful to someone who starts from a fresh session and needs to understand what exists, what changed, what is broken, and what to do next.

Specific rules:

- Continuity files are updated before and after each task, not necessarily every commit.
- [STATUS.md](./STATUS.md) summarizes the implementation status precisely and factually.
- [WHAT_WE_DID.md](./WHAT_WE_DID.md) is generally append-oriented, but old or irrelevant parts may be removed or compressed to keep it streamlined.
- [DO_NEXT.md](./DO_NEXT.md) is a prioritized queue. It may be reordered as implementation changes the project direction or understanding.
- [BUGS.md](./BUGS.md) includes confirmed defects, test gaps, and open risks.
- Generated-file work is not automatically excluded from continuity summaries. If generated output is too large or noisy, decide based on the task.
- [STATUS.md](./STATUS.md) should stay short: current implemented surface, current test status, current active task, and blocking issues.
- Continuity updates should usually be in the same task branch. The final after-task update should happen near the end of the branch. A separate final docs commit is acceptable when the code diff is large.
- Create one commit at the end of each task.
- Use one commit per task unless the user explicitly asks for a different commit structure.
- The task commit should include code, tests, and continuity-file updates for that task.
- Create one pull request for each task.
- Each task should happen on its own task branch.
- The task branch should be pushed and opened as a pull request after the task commit is created.
- **At most one open pull request at any time. No exceptions.** Before opening a new PR, check for any other open PR (`gh pr list`) and confirm it is merged first. This applies even to small, low-risk, or doc-only changes — there is no size or risk threshold that excuses opening a second PR while one is still open.
- Do not start a new task branch while the previous task pull request remains open.
- After a task pull request is merged, sync local `main` with `origin/main`.
- Create the next task branch from synced `origin/main`.
- CI should run only for pull requests targeting `main`.
- CI should not run on direct pushes to `main` or on bare branch pushes.

## Documentation Style

Use simple, factual language.

Do not use inflated or promotional words such as:

- complete
- comprehensive
- robust
- powerful
- seamless
- world-class

Avoid speculation. Do not guess. Do not assume intent, motivation, or hidden requirements.

Prefer facts from:

- Actual source code in this repository.
- Official specifications.
- Official documentation.
- Behavior observed by running commands or tests.
- Explicit user instructions.

When facts are not available, say what is unknown.

Prefer links over duplicated information. Cross-link related continuity docs when that makes navigation easier.

Keep continuity docs streamlined:

- Keep current task context.
- Keep useful decisions.
- Keep known bugs and next steps.
- Remove old side discussions.
- Remove outdated alternatives after a decision has been made.

## Coding Principles

Follow [PLAN.md](./PLAN.md) for the current architecture and roadmap.

The project uses Go, Elm, PostgreSQL, Tailwind, plain SQL migrations, generated Elm contracts from Go contract definitions, and an API-first UI.

The frontend is an API client. It must be replaceable without rewriting domain services or core API contracts.

### Domain Modeling

Model the domain with strong types.

Rules:

- Domain models are the center of the application.
- DB rows, API DTOs, and domain objects are separate.
- Do not expose third-party types from domain packages.
- Avoid magic strings and magic numbers.
- Avoid relying on Go zero values as meaningful defaults.
- Prefer explicit constructors over exported mutable structs.
- Prefer unexported fields with validated constructors.
- Prefer product types for required grouped values.
- Prefer tagged unions or sealed-interface-style variants for real variants.
- Prefer strong enums over raw strings.
- Limit optional fields; absence should be explicit and domain-meaningful.
- Avoid `nil` in domain code.
- Do not use `any`.
- Do not use generic `Result[T]` or `Option[T]`.
- Use handwritten or generated per-type result, option, and sum types.
- Do not use type-ignore comments or intentional lint/type-safety escapes.
- Do not use stringly typed domain values.
- Do not use dictly typed domain values such as generic maps for meaningful business concepts.
- Avoid booleans in domain and contract models.
- Use strong enums, tagged unions, explicit commands, or separate methods for separate intentions.
- Avoid nullable timestamp fields that secretly encode lifecycle state such as activated, deleted, revoked, or disabled.
- Model lifecycle through explicit states and transition events.
- Timestamps should record facts or event times, not act as hidden status flags.
- Do not add fallbacks, workarounds, fake behavior, false behavior, or disabled tests instead of fixing the underlying issue.
- Do not introduce fallback behavior unless it is required for an explicit reliability mechanism such as retry/backoff.
- If fallback behavior seems necessary, stop and ask the user before adding it.
- Treat fallbacks as risky because they can hide bugs, dead code, and functionally dead code.

Low-level dependencies may expose booleans, nils, generic data, or weak shapes. Keep those at boundaries and convert them into Sharecrop domain types before application logic runs.

### Dependency Boundaries

Third-party dependencies must be segregated behind small boundary packages:

- `internal/db`: `pgx` and generated `sqlc` access.
- `internal/auth`: JWT and Argon2id usage.
- `internal/core/id`: UUID usage.
- `internal/schema`: local Sharecrop schema parser and validator.
- `internal/http`: raw HTTP and JSON decoding/encoding.

No external JSON Schema dependency is used. Sharecrop schema parsing and validation are local code built from Sharecrop domain types.

### Testing

Work in a TDD style where practical.

Use the test pyramid:

- Unit tests for domain constructors, enums, tagged unions, lifecycle transitions, schema parsing, validation, redaction, token generation, and ledger arithmetic.
- Integration tests for repositories, migrations, Postgres transactions, auth session storage, escrow, acceptance, visibility checks, and sensitive-field indexing.
- HTTP end-to-end tests for the API.
- Playwright end-to-end tests for browser UI workflows.
- Manual screenshot review for UI changes.

CI and local checks should also include:

- Strict format checks.
- Linters.
- Type checks.
- Project-specific weak-typing checks.
- Dead-code detection.
- Copy-paste detection.
- Dependency-boundary checks.

Any behavior required by the UI should have API-level coverage before or alongside Playwright coverage.

When the UI changes:

- Run the app locally when practical.
- Capture screenshots of affected screens or states.
- Inspect screenshots for layout, spacing, text overflow, broken styling, missing content, and obvious accessibility issues.
- Record skipped screenshot checks in [BUGS.md](./BUGS.md) with the reason.
- Add or update Playwright tests as the UI matures and workflows stabilize.

## Task Workflow

For each task:

1. Read the continuity files.
2. Check the working tree.
3. Update continuity docs if the task scope or state needs clarification.
4. Make the smallest coherent set of changes for the task.
5. Add or update tests throughout the task according to the test pyramid.
6. Run relevant checks throughout the task and before finishing.
7. Update continuity docs after the task, in past tense.
8. Record known bugs or skipped checks in [BUGS.md](./BUGS.md).
9. Commit the task as one commit.
10. Push the task branch and create one pull request for the task.
11. Do not start the next task until the open pull request is merged or the user explicitly changes direction.

Do not overwrite unrelated user changes.

Do not use destructive git commands unless the user explicitly asks for them.

## PR Descriptions

PR descriptions should be precise and timeless.

They should describe the changes as a whole in the branch, not the evolution of understanding during the branch.

They should not reproduce the actual code.

They may link to continuity files when useful.

## Open Questions

No agent-practice questions were open.
