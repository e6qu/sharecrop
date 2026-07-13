# Do Next

Prioritized queue. Reread [AGENTS.md](./AGENTS.md) before starting and update the
continuity files if task scope changes.

1. **MCP/SSE for concurrent streaming.** Move MCP/SSE toward HTTP/2 by default
   (HTTP/3-ready) to support ~100 concurrent streaming sessions, keeping HTTP/1.1
   as an explicit option for regular UI/API traffic. Note the current backend
   delivers SSE across replicas by DB polling and returns a bounded response over
   the WASI bridge (which cannot stream); a streaming transport would let it push.

2. **Stand up the AWS deployment.** The Terraform in `deploy/terraform/` provisions
   the ALB + ECS Fargate service + Aurora Serverless v2 + RDS Proxy + secrets/IAM
   into an existing VPC — fill `terraform.tfvars` (region, image, vpc/subnets),
   `terraform apply`, run the migrate task, and point DNS at the ALB. Then cut the
   first `feat`/`fix` release so the image publishes to ghcr. Requires
   GitHub-hosted arm64 runners for the release build. See
   [docs/deployment.md](./docs/deployment.md).

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
- Containerization for ECS Fargate: slim multi-arch (arm64) image with a baked
  wazero AOT cache (no build on startup) and a ghcr release workflow versioned by
  conventional commits. See [docs/deployment.md](./docs/deployment.md).
