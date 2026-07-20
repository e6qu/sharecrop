# Do Next

Prioritized queue. Reread [AGENTS.md](./AGENTS.md) before starting and update the
continuity files if task scope changes.

1. **MCP/SSE for concurrent streaming.** Move MCP/SSE toward HTTP/2 by default
   (HTTP/3-ready) to support ~100 concurrent streaming sessions, keeping HTTP/1.1
   as an explicit option for regular UI/API traffic. Note the current backend
   delivers SSE across replicas by DB polling and returns a bounded response over
   the WASI bridge (which cannot stream); a streaming transport would let it push.

2. **Maintain the AWS deployment.** The Terraform in `deploy/terraform/`
   provisions private Amazon ECS Fargate tasks and an Amazon API Gateway HTTP
   API private integration through AWS Cloud Map in an existing VPC. Keep the
   API route throttles, access logs, container health checks, private task
   addressing, image and module pins current. Keep Sharecrop in its distinct
   database inside the shared PostgreSQL service, run migrations before an
   image requires them, and verify both direct entry and the Shauth Apps-catalog
   launch after every authentication change. The migration task used
   database-only configuration, and serve/MCP refused to start against a schema
   older than the image. Keep
   the Shauth confidential client registered with
   `https://sharecrop.dev.e6qu.dev/api/auth/shauth/backchannel-logout` as its
   Back-Channel Logout URI and
   `https://sharecrop.dev.e6qu.dev/api/auth/signed-out` as its allowed
   post-logout redirect URI. Keep the authorization callback registered as
   `https://sharecrop.dev.e6qu.dev/api/auth/shauth/callback`.
   Keep Front-Channel Logout registered as
   `https://sharecrop.dev.e6qu.dev/api/auth/shauth/frontchannel-logout`.
   Keep environment image pins on the immutable 12-character commit-SHA generic
   manifest published by the release workflow. Keep the package retention gate
   deleting untagged, incomplete, mixed-tag, unrecognized, and older versions
   while preserving at most 20 complete release triplets.
   See [docs/deployment.md](./docs/deployment.md).

3. Keep expanding shared scenario parity as new user-visible API surfaces are
   added, and keep running it against both SQL engines and the real backend as
   behavior changes.

4. Keep expanding generated/fixture-level HTTP contract coverage as the API
   surface grows.

5. Audit remaining raw-ID browser flows and replace high-traffic fields with
   selectors where directory data exists. No confirmed high-traffic raw-ID input
   remains after the latest audit in
   [docs/raw_id_browser_flow_audit.md](./docs/raw_id_browser_flow_audit.md).

6. Do not add anonymous worker identity or provider email delivery unless the
   product direction changes. Registered-user submissions remain the model;
   account and organization setup stays admin/org-admin driven.

UI minors queue:

- Add `type_ "button"` to any remaining secondary buttons that move into forms;
  continue replacing raw-id fields as directory-backed selectors become available.

Recently finished (details in [WHAT_WE_DID.md](./WHAT_WE_DID.md)):

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
