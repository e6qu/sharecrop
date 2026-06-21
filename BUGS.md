# Bugs

No confirmed application defects were known.

Test gaps:

- Pull request 5 generated contracts were limited to the first auth, error, identifier, organization, and team read models. Request/command contracts and HTTP contract fixture tests still need to expand as the API grows.
- Pull request 6 schema parsing and validation are not yet connected to task creation or submissions because those surfaces are scheduled for the task and submission pull requests.

Known risks:

- The exact UUIDv7 library behavior had not been verified in code.
