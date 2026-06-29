# Raw-ID Browser Flow Audit

## Current Finding

No confirmed high-traffic browser workflow requires a user to type a raw ID when
directory data is already available.

Picker-backed flows are used for common user, team, organization, and task
selection:

- task owner and visibility selection
- reservation assignee selection
- organization/team work queues
- task funding
- series task selection
- collectible award recipient selection
- collectible transfer recipient selection

## Remaining ID Surfaces

Raw IDs still appear where they are protocol or operator data:

- audit event filters and audit rows
- admin moderation report rows
- API/MCP examples and copyable integration payloads
- links and route fragments
- stored event metadata

These are not currently classified as replacement candidates because they expose
identifiers for inspection or integration rather than asking a normal workflow
user to find and type an ID.

## Follow-Up Trigger

Replace a raw-ID field with a paginated/typeahead selector when all of these are
true:

1. The browser already has an API for discovering that entity type.
2. The flow is a normal user workflow rather than an operator/audit/protocol
   surface.
3. The selected ID is not meant to be copied into an external integration.
