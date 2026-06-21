# Bugs

No confirmed application defects were known.

Test gaps:

- Pull request 7 generated task contracts cover compact task list items and capability-token responses. Full task detail form/view models remain handwritten Elm work because the current generator emits `Decode.mapN` decoders and should not be used for large product records.
- Pull request 7 added task API endpoints without visible task list or task detail screens. Browser coverage remains an app-shell smoke test until task user interface screens are added.
- Request/command contracts and HTTP contract fixture tests still need to expand as the API grows.
- Schema validation is connected to task creation but not yet to submissions because submissions are scheduled for pull request 8.

Known risks:

- The exact UUIDv7 library behavior had not been verified in code.
- `make build` passed locally with a non-fatal Go global module stat-cache warning from the sandbox.
