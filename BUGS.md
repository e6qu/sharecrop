# Bugs

No confirmed application defects were known.

Test gaps:

- Pull request 5 generated contracts were limited to the first auth, error, identifier, organization, and team read models. Request/command contracts and HTTP contract fixture tests still need to expand as the API grows.

Known risks:

- The local Sharecrop schema parser had not been implemented.
- The exact UUIDv7 library behavior had not been verified in code.
