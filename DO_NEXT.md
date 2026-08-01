# Do Next

Prioritized queue. Reread [AGENTS.md](./AGENTS.md) before starting and update the
continuity files if task scope changes.

1. **Edge architecture for true streaming.** The server now supports HTTP/2
   cleartext behind `SHARECROP_HTTP_PROTOCOL=h2c`, but the Amazon API Gateway
   HTTP API edge speaks HTTP/1.1 to the integration, buffers responses, and
   caps every request at 30 seconds, so MCP/browser SSE remain
   replay-and-reconnect in production. Real end-to-end streaming for ~100
   concurrent sessions is an edge decision (an ALB with an h2c target protocol
   or an API Gateway replacement) that also conflicts with the current
   Terraform no-load-balancer policy gate; decide deliberately before
   implementing. The WASI bridge still cannot stream; native mode already
   pushes.

2. **Remaining durability edges.** Host-side expiry sweeps still emit their
   events post-commit (outside the expiry transaction) — the one emission
   path not yet on the outbox. Webhook secrets are stored as written because
   HMAC needs the plaintext; at-rest encryption needs a key-management
   decision.

3. **Deliberate product decisions still open.** Sybil stance for signup
   grants (per-IP throttle is the only guard); auto-accept policy
   (validation + escrow + limits exist as building blocks); collectible
   scarcity (anyone can mint); API versioning (`/v1`); socket-mode webhooks
   for local agents (outbound connection, Slack-style) — gated on the edge
   decision in item 1, with agent event polling/long-poll as the shipped
   interim; local-timezone expiry input (needs a JS port or dependency).

4. **Maintain the AWS deployment.** The Terraform in `deploy/terraform/`
   provisions private Amazon ECS Fargate tasks and an Amazon API Gateway HTTP
   API private integration through AWS Cloud Map in an existing VPC. Keep the
   API route throttles, access logs, container health checks, private task
   addressing, image and module pins current. Keep Sharecrop in its distinct
   database inside the shared PostgreSQL service, keep the AWS Step Functions
   workflow running the standalone migration task before every service rollout,
   and verify both direct entry and the Shauth Apps-catalog launch after every
   authentication change. The migration task used database-only configuration,
   and serve/MCP refused to start against a schema older than the image. Keep
   the Shauth confidential client registered with
   `https://sharecrop.dev.e6qu.dev/api/auth/shauth/backchannel-logout` as its
   Back-Channel Logout URI and
   `https://sharecrop.dev.e6qu.dev/auth/shauth/logout/complete` as its allowed
   post-logout redirect URI, with
   `https://sharecrop.dev.e6qu.dev/api/auth/signed-out` retained as the
   application-local signed-out destination Shauth registered for the app.
   Keep the registered validation URL at
   `https://sharecrop.dev.e6qu.dev/auth/validation` and bind its expected
   release revision to the same immutable image revision. Keep the
   authorization callback registered as
   `https://sharecrop.dev.e6qu.dev/api/auth/shauth/callback`.
   Keep Front-Channel Logout registered as
   `https://sharecrop.dev.e6qu.dev/api/auth/shauth/frontchannel-logout`.
   Keep environment image pins on the immutable 12-character commit-SHA generic
   manifest published by the release workflow. Keep the package retention gate
   deleting untagged, incomplete, mixed-tag, unrecognized, and older versions
   while preserving at most 20 complete release triplets.
   See [docs/deployment.md](./docs/deployment.md).

5. Keep expanding shared scenario parity as new user-visible API surfaces are
   added, and keep running it against both SQL engines and the real backend as
   behavior changes.

6. Keep expanding generated/fixture-level HTTP contract coverage as the API
   surface grows.

7. Audit remaining raw-ID browser flows and replace high-traffic fields with
   selectors where directory data exists. No confirmed high-traffic raw-ID input
   remains after the latest audit in
   [docs/raw_id_browser_flow_audit.md](./docs/raw_id_browser_flow_audit.md).

8. Do not add anonymous worker identity or provider email delivery unless the
   product direction changes. Registered-user submissions remain the model;
   account and organization setup stays admin/org-admin driven.

UI minors queue:

- Add `type_ "button"` to any remaining secondary buttons that move into forms;
  continue replacing raw-id fields as directory-backed selectors become available.

Recently finished (details in [WHAT_WE_DID.md](./WHAT_WE_DID.md)):

- Agent loop hardening: the in-transaction event outbox with idempotent
  dispatch and crash-recovery sweep, replay-aware emission, superseded
  competing submissions, structured disputes, agent/org event-feed access
  with long-polling and MCP `list_events`, funded-state discovery, list
  totals, handler-derived OpenAPI status codes and error schema, forced
  sign-out on `unauthenticated`, the garden-gnome identity, the
  test-hardening pass (fan-out scale, multi-replica exactly-once, two-actor
  live flow, reference webhook receiver, migration rehearsal).
- Agent loop completion: marketplace webhook audience with filters (push
  discovery of new public tasks), mintable webhook scopes, reviewer
  submission reads over MCP, validation errors + kept reservations on
  invalid submissions, required explicit task visibility, admin credit
  grants, agent-credential public task listing over REST with optional
  `scope` and `created_after`, OpenAPI parameter declarations, display
  names end to end, needs-review signals, humane inbox/feed, and the
  first-run explainer.
- The review-driven platform upgrade: domain event stream with service-layer
  emission (MCP and REST now produce identical notifications/events), sealed
  notification kinds with unread filter/count, outbound webhooks (signed,
  retried, SSRF-guarded), the per-user event feed + SSE, the lifecycle runner
  (reservation/task expiry, retention, rate-limit and MCP-session sweeps,
  webhook pump), real task expiration with refunds, per-account ledger
  idempotency, the REST org-credential scope gate, machine-readable error
  codes, `next_offset` pagination, MCP create/accept/list parity plus credits
  tools, opt-in h2c, and the farm/pixel retheme of the shipped app.
- The Shauth relying-party recovery boundary: an accessible branded
  light/dark app-local signed-out page, explicit same-origin recovery, and a
  real PostgreSQL/Ory Hydra/browser matrix for catalog and direct SSO,
  relying-party and provider logout, reload, and retained-credential rejection.
- The Shauth application-owned logout bridge and release-validation boundary:
  destination-free completion, durable username/email/role identity, exact
  release reporting, hostile-input checks, and a browser matrix pinned to the
  released provider commit.
- The single-store-implementation program: one engine-neutral store (`internal/db`,
  Postgres + SQLite), `internal/wasmdemo` deleted, the browser demo runs the real
  backend over in-browser SQLite.
- WASI production hosting is the default and the browser/backend now build from the
  same source; the pooled guest reaches Postgres through the bridge.
- Hardening of the production-default WASI path (randomness, clock, MCP SSE
  pool-exhaustion DoS, request-bridge fidelity for rate limiting and the MCP origin
  check, payload/frame size limits).
- Containerization for ECS Fargate: slim multi-architecture (arm64) image with a
  baked wazero AOT cache (no build on startup) and an immutable commit-SHA
  GitHub Container Registry release workflow. See
  [docs/deployment.md](./docs/deployment.md).
