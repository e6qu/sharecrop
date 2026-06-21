# Bugs

No confirmed application defects were known.

Test gaps:

- Pull request 3 runtime HTTP end-to-end tests were not run locally because the environment rejected the required local listener and PostgreSQL approval after the usage limit was reached.
- Pull request 3 Playwright browser tests were not rerun locally because the user interface was not changed and the environment could not grant further browser/listener approval.

Known risks:

- The local Sharecrop schema parser had not been implemented.
- The Go-to-Elm contract generator had not been implemented.
- The exact UUIDv7 library behavior had not been verified in code.
