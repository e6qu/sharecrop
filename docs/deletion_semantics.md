# Deletion Semantics

Sharecrop does not use hard-delete features for core rows. Current lifecycle changes are modeled as explicit states, anonymization events, redaction, tombstones, and audit records, not row removal.

## Current Behavior

- Account removal is deactivation: email is anonymized, password credentials are removed, refresh tokens are revoked, and account-token rows are revoked.
- Users can create audited privacy requests for data export or sensitive-field redaction. Platform admins can resolve requests; resolution stores basic export JSON or marks delete-on-request sensitive-field metadata as redacted without removing core rows.
- Organization members are deactivated, not deleted.
- Tasks are cancelled, closed, refunded, unpublished, or left in their recorded lifecycle state.
- Submissions, comments, reservations, ledger entries, collectibles, audit events, notifications, and ownership rows remain as historical records.
- PostgreSQL foreign keys block deletion of referenced rows. Application lifecycle flows should not depend on deleting referenced core rows.

## Lifecycle And Redaction Rules

- Do not add hard deletes for users, organizations, tasks, submissions, ledger entries, comments, collectibles, audit events, notifications, or ownership rows.
- Prefer explicit domain states, tombstone records, or value redaction over row removal when records are part of task, reward, audit, ownership, or financial history.
- Sensitive submitted fields may be redacted according to schema retention metadata, but the redaction event must leave an audit/tombstone record that does not preserve the sensitive value.
- Any destructive or redaction workflow must fail loudly when a referenced record, ownership row, ledger entry, or sensitive-field index cannot be updated.
- Do not use nullable timestamps as hidden lifecycle state. Use explicit states and transition events.
- Do not add fallback redaction paths. If a row cannot be safely redacted or transitioned, return an error and leave existing data unchanged.

## Implementation Gate

Before extending redaction executors, retention jobs, lifecycle transitions, or export generators, add a design note that identifies:

- the lifecycle state or tombstone shape,
- every referencing table and foreign-key behavior,
- audit events to write,
- effects on ledger/accounting and collectible ownership,
- effects on task/submission/comment visibility,
- retention handling for sensitive response fields,
- migration and backfill requirements,
- HTTP and browser behavior for tombstoned or redacted records.
