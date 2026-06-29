# Deletion Semantics

Sharecrop does not expose hard-delete features for core rows. Current lifecycle changes are modeled as explicit states or anonymization events, not row removal.

## Current Behavior

- Account removal is deactivation: email is anonymized, password credentials are removed, refresh tokens are revoked, and account-token rows are revoked.
- Users can create audited privacy requests for data export or sensitive-field deletion. These requests are queued audit records; they do not perform hard deletion, export generation, redaction, or retention work.
- Organization members are deactivated, not deleted.
- Tasks are cancelled, closed, refunded, unpublished, or left in their recorded lifecycle state.
- Submissions, comments, reservations, ledger entries, collectibles, audit events, notifications, and ownership rows remain as historical records.
- PostgreSQL foreign keys block deletion of referenced rows unless an explicit future migration defines different behavior.

## Rules For Future Delete Features

- Do not add hard deletes for users, organizations, tasks, submissions, ledger entries, comments, collectibles, audit events, or ownership rows without a written lifecycle design for every referencing table.
- Prefer explicit domain states, tombstone records, or value redaction over row removal when records are part of task, reward, audit, ownership, or financial history.
- Sensitive submitted fields may be redacted or removed according to schema retention metadata, but the deletion event must leave an audit/tombstone record that does not preserve the sensitive value.
- Any destructive or redaction workflow must fail loudly when a referenced record, ownership row, ledger entry, or sensitive-field index cannot be updated.
- Do not use nullable timestamps as hidden lifecycle state. Use explicit states and transition events.
- Do not add fallback deletion paths. If a row cannot be safely deleted or redacted, return an error and leave existing data unchanged.

## Implementation Gate

Before implementing a deletion executor, redaction executor, retention job, or export generator, add a design note that identifies:

- the lifecycle state or tombstone shape,
- every referencing table and foreign-key behavior,
- audit events to write,
- effects on ledger/accounting and collectible ownership,
- effects on task/submission/comment visibility,
- retention handling for sensitive response fields,
- migration and backfill requirements,
- HTTP and browser behavior for already-deleted or redacted records.
