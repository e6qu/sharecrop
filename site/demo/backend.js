// In-browser fake Sharecrop backend for the demo.
//
// The demo serves the REAL compiled Elm client (web/static/app.js). This script
// overrides window.fetch and answers the /api/* calls the client makes from an
// in-memory, stateful store seeded with realistic agentic-work tasks — so the
// demo is the same code as the shipped client, with no backend, and cannot drift
// from the real UI. It is intentionally a demo fake: tokens are not validated,
// the single seeded "you" is the requester Mara, and behavior mirrors the Go
// backend's contracts closely enough to drive every screen.
(function () {
  "use strict";

  const ME = "user-mara";

  const users = {
    "user-mara": { id: "user-mara", name: "Mara Chen" },
    "user-jules": { id: "user-jules", name: "Jules Park" },
    "user-ren": { id: "user-ren", name: "Ren Ito" },
    "user-sol": { id: "user-sol", name: "Sol Rivera" },
    "user-tala": { id: "user-tala", name: "Tala Stone" },
  };

  // Realistic, deep example tasks: concrete instructions, real-looking input
  // payloads, and strict response schemas a requester would actually post.
  const invoiceSchema = JSON.stringify({
    kind: "object",
    fields: {
      invoices: {
        kind: "array",
        required: true,
        items: {
          kind: "object",
          fields: {
            invoice_id: { kind: "string", required: true },
            vendor: { kind: "string", required: true },
            total: { kind: "decimal_string", required: true },
            due_date: { kind: "string", required: true },
          },
        },
      },
    },
  });
  const ticketSchema = JSON.stringify({
    kind: "object",
    fields: {
      labels: {
        kind: "array",
        required: true,
        minItems: 8,
        maxItems: 8,
        items: { kind: "string", enum: ["bug", "billing", "feature_request", "account", "other"] },
      },
    },
  });
  const ledgerSchema = JSON.stringify({
    kind: "object",
    fields: {
      suspicious_ids: { kind: "array", required: true, items: { kind: "string" } },
      reviewed_count: { kind: "integer", required: true },
    },
  });
  const expenseSchema = JSON.stringify({
    kind: "object",
    fields: {
      category_totals: {
        kind: "array",
        required: true,
        items: {
          kind: "object",
          fields: {
            category: { kind: "string", required: true, enum: ["travel", "meals", "software", "office"] },
            total: { kind: "decimal_string", required: true },
          },
        },
      },
    },
  });
  const dateSchema = JSON.stringify({
    kind: "object",
    fields: { iso_dates: { kind: "array", required: true, minItems: 8, maxItems: 8, items: { kind: "string" } } },
  });
  const reviewSchema = JSON.stringify({
    kind: "object",
    fields: {
      items: {
        kind: "array",
        required: true,
        minItems: 5,
        maxItems: 5,
        items: {
          kind: "object",
          fields: {
            product: { kind: "string", required: true },
            rating: { kind: "integer", required: true },
          },
        },
      },
    },
  });
  const pretty = (value) => JSON.stringify(value, null, 2);

  let seq = 100;
  const nextId = (prefix) => `${prefix}-${++seq}`;

  function task(overrides) {
    return Object.assign({
      id: nextId("task"),
      owner_kind: "user",
      owner_id: ME,
      created_by: ME,
      reward_kind: "credit",
      reward_credit_amount: 0,
      reward_collectible_count: 0,
      participation_policy: "reservation_required",
      assignee_scope: "user",
      reservation_expiry_hours: 48,
      state: "open",
      visibility_kind: "public",
      visibility_id: "",
      series_kind: "standalone",
      series_id: "",
      series_position: 0,
      payload_kind: "none",
      payload_json: "",
      availability_kind: "available",
      description: "",
      response_schema_json: '{"kind":"freeform"}',
      reservations: [],
      submissions: [],
      escrow: 0,
    }, overrides);
  }

  const db = {
    balance: 1240,
    ledger: [
      { id: "entry-1", kind: "signup_grant", amount: 1500, task_id: "" },
      { id: "entry-2", kind: "task_escrow", amount: -260, task_id: "task-5" },
      { id: "entry-3", kind: "task_payout", amount: -30, task_id: "task-7" },
      { id: "entry-4", kind: "task_tip", amount: -5, task_id: "task-7" },
      { id: "entry-5", kind: "task_refund", amount: 45, task_id: "task-8" },
    ],
    collectibles: [
      { id: "col-1", name: "Harvest Star", kind: "badge", state: "minted", transfer_policy: "transferable_between_users", owner_id: ME },
      { id: "col-2", name: "Golden Sickle", kind: "badge", state: "minted", transfer_policy: "non_transferable_except_payout", owner_id: ME },
    ],
    organizations: [{ id: "org-lattice", name: "Lattice Field Co", created_by: ME }],
    members: { "org-lattice": [{ id: "mem-1", organization_id: "org-lattice", user_id: ME, status: "active", roles: ["owner"] }] },
    orgTeams: { "org-lattice": [{ id: "team-survey", owner_kind: "organization", organization_id: "org-lattice", owner_user_id: "", name: "Survey crew", created_by: ME }] },
    standaloneTeams: [{ id: "team-field", owner_kind: "user", organization_id: "", owner_user_id: ME, name: "Field hands", created_by: ME }],
    teamMembers: { "team-survey": ["user-jules", "user-tala"], "team-field": ["user-tala"] },
    credentials: [
      { id: "cred-1", label: "Sol's field agent", scopes: ["tasks_read", "submissions_write"], state: "active" },
      { id: "cred-2", label: "Lattice reviewer agent", scopes: ["tasks_read", "submissions_review"], state: "active" },
    ],
    series: [{ id: "series-orchard", title: "Orchard intake", position: 0 }],
    tasks: [],
  };

  // Every task is self-contained: all data needed to solve it is embedded in the
  // task itself (rendered as the "Task input" block), with no external lookups,
  // attachments, or live data required.
  db.tasks = [
    task({
      id: "task-1", title: "Extract line items from 6 vendor invoices", owner_id: "user-jules", created_by: "user-jules",
      description:
        "The Task input below contains the OCR'd text of 6 vendor invoices (one string per invoice). For each invoice, return invoice_id, vendor, total, and due_date. Rules: total is the numeric grand total with the currency symbol and thousands separators removed (e.g. \"$1,240.55\" -> \"1240.55\"); due_date is the written date converted to YYYY-MM-DD. Keep the invoices in the given order.",
      reward_credit_amount: 80, escrow: 80, response_schema_json: invoiceSchema,
      payload_kind: "inline",
      payload_json: pretty({
        invoices: [
          "INV-1041 | Birch Supply Co | Grand total: $1,240.55 | Due 12 Jul 2026",
          "INV-1042 | Cedar Freight | Grand total: $88.10 | Due 3 Jul 2026",
          "INV-1043 | Delta Print | Grand total: $146.20 | Due 19 Jul 2026",
          "INV-1044 | Meadow Labs | Grand total: $902.75 | Due 22 Jul 2026",
          "INV-1045 | Grove Cafe | Grand total: $41.25 | Due 5 Jul 2026",
          "INV-1046 | North Mill | Grand total: $5,310.00 | Due 1 Aug 2026",
        ],
      }),
    }),
    task({
      id: "task-2", title: "Classify 8 support tickets by category", owner_id: "user-ren", created_by: "user-ren",
      description:
        "Assign each of the 8 support tickets in the Task input exactly one category from this set: bug, billing, feature_request, account, other. Return a labels array with one label per ticket, in the same order as the input.",
      reward_credit_amount: 45, escrow: 45, participation_policy: "open", response_schema_json: ticketSchema,
      payload_kind: "inline",
      payload_json: pretty({
        tickets: [
          "1. Card declined at checkout even though it has funds",
          "2. App crashes every time I open the Reports tab",
          "3. How do I export my data to CSV?",
          "4. I was double charged for this month's plan",
          "5. Can't log in — need to reset my 2FA device",
          "6. Please add a dark mode",
          "7. Invoice shows the wrong VAT amount",
          "8. Just wanted to say the new dashboard looks great",
        ],
      }),
    }),
    task({
      id: "task-3", title: "Verify 10 ledger transfers for fraud signals",
      description:
        "Review the 10 transfers in the Task input. Flag a transfer's id as suspicious if EITHER (a) its item_id appears in more than one transfer, OR (b) its to_account is in banned_accounts. Return suspicious_ids (the flagged transfer ids) and reviewed_count (the number of transfers you reviewed, which is 10).",
      reward_credit_amount: 60, escrow: 60, visibility_kind: "organization", visibility_id: "org-lattice",
      response_schema_json: ledgerSchema, assignee_scope: "user",
      payload_kind: "inline",
      payload_json: pretty({
        banned_accounts: ["ACC-666", "ACC-999"],
        transfers: [
          { id: "TR-01", item_id: "ITM-A", from_account: "ACC-100", to_account: "ACC-200", amount: "120.00" },
          { id: "TR-02", item_id: "ITM-B", from_account: "ACC-101", to_account: "ACC-201", amount: "75.50" },
          { id: "TR-03", item_id: "ITM-C", from_account: "ACC-102", to_account: "ACC-666", amount: "300.00" },
          { id: "TR-04", item_id: "ITM-D", from_account: "ACC-103", to_account: "ACC-202", amount: "44.00" },
          { id: "TR-05", item_id: "ITM-A", from_account: "ACC-104", to_account: "ACC-203", amount: "120.00" },
          { id: "TR-06", item_id: "ITM-E", from_account: "ACC-105", to_account: "ACC-204", amount: "210.10" },
          { id: "TR-07", item_id: "ITM-F", from_account: "ACC-106", to_account: "ACC-205", amount: "9.99" },
          { id: "TR-08", item_id: "ITM-G", from_account: "ACC-107", to_account: "ACC-999", amount: "1500.00" },
          { id: "TR-09", item_id: "ITM-H", from_account: "ACC-108", to_account: "ACC-206", amount: "62.00" },
          { id: "TR-10", item_id: "ITM-E", from_account: "ACC-109", to_account: "ACC-207", amount: "210.10" },
        ],
      }),
      reservations: [{ id: "res-3-tala", task_id: "task-3", assignee_kind: "user", assignee_id: "user-tala", state: "requested", requested_by: "user-tala" }],
      participation_policy: "approval_required", availability_kind: "awaiting_approval",
    }),
    task({
      id: "task-4", title: "Categorize and total 6 expense lines",
      description:
        "Assign each of the 6 expense lines in the Task input to exactly one category (travel, meals, software, office), then return category_totals: for every category that has at least one expense, the sum of its amounts as a decimal string with two places. Amounts are given in the input; sum them exactly.",
      reward_credit_amount: 25, reward_kind: "bundle", reward_collectible_count: 1, escrow: 25, response_schema_json: expenseSchema,
      payload_kind: "inline",
      payload_json: pretty({
        categories: ["travel", "meals", "software", "office"],
        expenses: [
          { merchant: "Uber — ride to airport", amount: "32.50" },
          { merchant: "Olive Bistro — team lunch", amount: "88.00" },
          { merchant: "Figma — annual seat", amount: "144.00" },
          { merchant: "Staples — printer paper, 5 reams", amount: "23.40" },
          { merchant: "Rail — Lisbon to Porto ticket", amount: "46.10" },
          { merchant: "Cafe — coffee with client", amount: "12.75" },
        ],
      }),
      reservations: [{ id: "res-4-sol", task_id: "task-4", assignee_kind: "user", assignee_id: "user-sol", state: "active", requested_by: "user-sol" }],
      submissions: [{ id: "sub-4-sol", task_id: "task-4", submitter_id: "user-sol", state: "submitted", response_json: pretty({ category_totals: [{ category: "travel", total: "78.60" }, { category: "meals", total: "100.75" }, { category: "software", total: "144.00" }, { category: "office", total: "23.40" }] }), review_note: "", validation_errors: [], via_agent: true }],
      availability_kind: "reserved",
    }),
    task({
      id: "task-5", title: "Normalize 8 dates to ISO 8601",
      description:
        "Convert each of the 8 dates in the Task input to ISO 8601 (YYYY-MM-DD), preserving order. For ambiguous all-numeric dates, treat the format as day-first (so \"03/04/2026\" is 4 March 2026 -> \"2026-03-04\"). Slash, dash, and dot separators all appear.",
      reward_credit_amount: 30, escrow: 30, response_schema_json: dateSchema,
      payload_kind: "inline",
      payload_json: pretty({
        dates: ["March 5, 2026", "03/04/2026", "2026/12/01", "7 Jan 2026", "11-02-2026", "Apr 30 2026", "2026.06.18", "09/10/2026"],
      }),
    }),
    task({
      id: "task-6", title: "Extract product and rating from 5 reviews", participation_policy: "open",
      description:
        "Each of the 5 review lines in the Task input names a product and gives a rating out of 5. Return an items array, in order, with the product name and the rating as an integer (1-5).",
      reward_credit_amount: 36, escrow: 36, response_schema_json: reviewSchema,
      payload_kind: "inline",
      payload_json: pretty({
        reviews: [
          "Rating: 4/5 — Orchard Boots: great grip, runs a half size small",
          "Rating: 2/5 — Field Gloves wore through at the seams in a month",
          "Rating: 5/5 — Sun Hat is the best I've owned, packs flat",
          "Rating: 3/5 — Canvas Tote is sturdy but the strap is too short",
          "Rating: 1/5 — Rain Shell soaked through in light drizzle",
        ],
      }),
    }),
  ];

  // --- shape helpers (mirror the Go DTOs the Elm client decodes) ---
  function listItem(t) {
    return {
      id: t.id, owner_kind: t.owner_kind, title: t.title, reward_kind: t.reward_kind,
      reward_credit_amount: t.reward_credit_amount, reward_collectible_count: t.reward_collectible_count,
      participation_policy: t.participation_policy, assignee_scope: t.assignee_scope,
      reservation_expiry_hours: t.reservation_expiry_hours, state: t.state, visibility_kind: t.visibility_kind,
      availability_kind: t.availability_kind, viewer_action: viewerAction(t), created_by: t.created_by,
      active_assignee_kind: activeAssignee(t) ? "user" : "", active_assignee_id: activeAssignee(t),
    };
  }
  function detail(t) {
    return {
      id: t.id, owner_kind: t.owner_kind, owner_id: t.owner_id, title: t.title, description: t.description,
      reward_kind: t.reward_kind, reward_credit_amount: t.reward_credit_amount, reward_collectible_count: t.reward_collectible_count,
      participation_policy: t.participation_policy, assignee_scope: t.assignee_scope, reservation_expiry_hours: t.reservation_expiry_hours,
      state: t.state, visibility_kind: t.visibility_kind, visibility_id: t.visibility_id, series_kind: t.series_kind,
      series_id: t.series_id, series_position: t.series_position, response_schema_json: t.response_schema_json,
      payload_kind: t.payload_kind, payload_json: t.payload_json, created_by: t.created_by,
      availability_kind: t.availability_kind, viewer_action: viewerAction(t),
    };
  }
  function activeAssignee(t) {
    const active = t.reservations.find((r) => r.state === "active");
    return active ? active.assignee_id : "";
  }
  function viewerAction(t) {
    if (t.created_by === ME) return "none";
    if (t.state !== "open" || ["accepted", "rejected", "closed"].includes(t.availability_kind)) return "none";
    const mine = t.reservations.find((r) => r.assignee_id === ME);
    const sub = t.submissions.find((s) => s.submitter_id === ME);
    if (sub && sub.state === "submitted") return "none";
    if (sub && sub.state === "changes_requested") return "submit";
    if (activeAssignee(t) && activeAssignee(t) !== ME) return "none";
    if (t.participation_policy === "approval_required" && (!mine || mine.state === "requested")) return "request_approval";
    if (t.participation_policy === "reservation_required" && !mine) return "reserve";
    return "submit";
  }

  // --- routing ---
  const routes = [];
  const on = (method, pattern, handler) => routes.push({ method, parts: pattern.split("/"), handler });
  const ok = (body, status) => ({ status: status || 200, body: JSON.stringify(body) });
  const empty = (status) => ({ status: status || 204, body: "" });
  const err = (status, message) => ({ status, body: JSON.stringify({ message }) });
  const auth = () => ({ subject_kind: "user", subject_id: ME, access_token: "demo-access-token" });
  const findTask = (id) => db.tasks.find((t) => t.id === id);

  function match(method, path) {
    const segs = path.split("?")[0].split("/").filter((s) => s !== "");
    for (const route of routes) {
      if (route.method !== method) continue;
      const rp = route.parts.filter((s) => s !== "");
      if (rp.length !== segs.length) continue;
      const params = {};
      let hit = true;
      for (let i = 0; i < rp.length; i++) {
        if (rp[i].startsWith(":")) params[rp[i].slice(1)] = decodeURIComponent(segs[i]);
        else if (rp[i] !== segs[i]) { hit = false; break; }
      }
      if (hit) return { handler: route.handler, params };
    }
    return null;
  }

  // Auth: refresh auto-succeeds so the demo boots straight into the seeded app.
  on("POST", "/api/auth/refresh", () => ok(auth()));
  on("POST", "/api/auth/login", () => ok(auth()));
  on("POST", "/api/auth/register", () => ok(auth(), 201));
  on("POST", "/api/auth/guest", () => ok(auth(), 201));
  on("POST", "/api/auth/logout", () => empty());

  on("GET", "/api/credits/balance", () => ok({ amount: db.balance }));
  on("GET", "/api/credits/ledger", () => ok({ entries: db.ledger }));

  on("GET", "/api/tasks", (_p, url) => {
    const scope = url.searchParams.get("scope") || "";
    const includeReserved = url.searchParams.get("include_reserved") === "true";
    let list = db.tasks; // default (empty scope): everything visible to the demo user
    if (scope === "user") list = db.tasks.filter((t) => t.created_by === ME);
    else if (scope === "public") list = db.tasks.filter((t) => t.visibility_kind === "public" && (includeReserved || !activeAssignee(t)));
    else if (scope === "organization") list = db.tasks.filter((t) => t.visibility_kind === "organization");
    return ok({ tasks: list.map(listItem) });
  });
  on("GET", "/api/tasks/:id", (p) => { const t = findTask(p.id); return t ? ok(detail(t)) : err(404, "task not found"); });
  on("POST", "/api/tasks", async (_p, _url, body) => {
    const t = task({
      title: body.title || "Untitled task", description: body.description || "",
      reward_kind: (body.reward && body.reward.kind) || "none",
      reward_credit_amount: (body.reward && body.reward.credit_amount) || 0,
      participation_policy: (body.participation && body.participation.policy) || "reservation_required",
      visibility_kind: (body.visibility && body.visibility.kind) || "public",
      response_schema_json: body.response_schema_json || '{"kind":"freeform"}',
      state: "draft", availability_kind: "available",
    });
    db.tasks.unshift(t);
    return ok(detail(t), 201);
  });
  on("POST", "/api/tasks/:id/open", (p) => { const t = findTask(p.id); if (!t) return err(404, "task not found"); t.state = "open"; return ok(detail(t)); });
  on("POST", "/api/tasks/:id/cancel", (p) => { const t = findTask(p.id); if (!t) return err(404, "task not found"); t.state = "cancelled"; return ok(detail(t)); });
  on("POST", "/api/tasks/:id/funding", (p, _url, body) => {
    const t = findTask(p.id); if (!t) return err(404, "task not found");
    const amount = (body && body.amount) || 0;
    if (amount > db.balance) return err(409, "insufficient credits to fund the task");
    // Funding only moves credits into escrow; task state is unchanged (the /open
    // route moves draft -> open). "funded" is not a valid TaskState.
    db.balance -= amount; t.escrow += amount;
    return ok({ task_id: t.id, amount: t.escrow, state: "held" }, 201);
  });
  on("POST", "/api/tasks/:id/reservations", (p) => {
    const t = findTask(p.id); if (!t) return err(404, "task not found");
    const state = t.participation_policy === "approval_required" ? "requested" : "active";
    const r = { id: nextId("res"), task_id: t.id, assignee_kind: "user", assignee_id: ME, state, requested_by: ME };
    t.reservations.push(r);
    if (state === "active") t.availability_kind = "reserved";
    return ok(r, 201);
  });
  on("GET", "/api/tasks/:id/reservations", (p) => { const t = findTask(p.id); return t ? ok({ reservations: t.reservations }) : err(404, "task not found"); });
  const reservationChange = (state, availability) => (p) => {
    const t = findTask(p.id); if (!t) return err(404, "task not found");
    const r = t.reservations.find((x) => x.id === p.rid);
    if (!r) return err(404, "reservation not found");
    r.state = state; if (availability) t.availability_kind = availability;
    return ok(r);
  };
  on("POST", "/api/tasks/:id/reservations/:rid/approve", reservationChange("active", "reserved"));
  on("POST", "/api/tasks/:id/reservations/:rid/decline", reservationChange("declined", "available"));
  on("POST", "/api/tasks/:id/reservations/:rid/cancel", reservationChange("cancelled_by_requester", "available"));
  on("GET", "/api/tasks/:id/submissions", (p) => { const t = findTask(p.id); return t ? ok({ submissions: t.submissions }) : err(404, "task not found"); });
  on("POST", "/api/tasks/:id/submissions", (p, _url, body) => {
    const t = findTask(p.id); if (!t) return err(404, "task not found");
    const s = { id: nextId("sub"), task_id: t.id, submitter_id: ME, state: "submitted", response_json: (body && body.response_json) || "{}", review_note: "", validation_errors: [] };
    t.submissions.push(s); t.availability_kind = "reserved";
    return ok({ submission: s, receipt_token: nextId("receipt") }, 201);
  });
  const decide = (state, availability) => (p, _url, body) => {
    const t = findTask(p.id); if (!t) return err(404, "task not found");
    const s = t.submissions.find((x) => x.id === p.sid);
    if (!s) return err(404, "submission not found");
    s.state = state; s.review_note = (body && body.review_note) || "";
    t.availability_kind = availability; if (state !== "changes_requested") t.state = "closed";
    const worker = s.submitter_id;
    const payout = state === "accepted" ? (t.reward_credit_amount || 0) : ((body && body.payout_amount) || 0);
    return ok({ task_id: t.id, submission_id: s.id, state, review_note: s.review_note, payout_kind: t.reward_kind, payout_amount: payout, worker_user_id: worker, tip_amount: (body && body.tip_amount) || 0 });
  };
  on("POST", "/api/tasks/:id/submissions/:sid/accept", (p, url, body) => {
    const t = findTask(p.id); if (!t) return err(404, "task not found");
    const s = t.submissions.find((x) => x.id === p.sid); if (!s) return err(404, "submission not found");
    s.state = "accepted"; t.availability_kind = "closed"; t.state = "closed";
    return ok({ task_id: t.id, submission_id: s.id, payout_kind: t.reward_kind, payout_amount: t.reward_credit_amount || 0, worker_user_id: s.submitter_id, collectible_ids: [], tip_amount: (body && body.tip_amount) || 0 });
  });
  on("POST", "/api/tasks/:id/submissions/:sid/reject", decide("rejected", "closed"));
  on("POST", "/api/tasks/:id/submissions/:sid/request-changes", decide("changes_requested", "reserved"));
  on("POST", "/api/tasks/:id/capability-tokens", (p) => ok({ id: nextId("cap"), task_id: p.id, state: "active", token: "demo-capability-" + nextId("tok") }, 201));

  on("GET", "/api/agent-credentials", () => ok({ credentials: db.credentials }));
  on("POST", "/api/agent-credentials", (_p, _url, body) => {
    const cred = { id: nextId("cred"), label: (body && body.label) || "Agent", scopes: (body && body.scopes) || [], state: "active" };
    db.credentials.push(cred);
    return ok({ credential: cred, secret: "demo-secret-" + nextId("s") }, 201);
  });
  on("DELETE", "/api/agent-credentials/:id", (p) => { const c = db.credentials.find((x) => x.id === p.id); if (c) c.state = "revoked"; return ok(c || {}); });

  on("GET", "/api/collectibles", () => ok({ collectibles: db.collectibles.filter((c) => c.owner_id === ME) }));
  on("POST", "/api/collectibles", (_p, _url, body) => {
    const c = { id: nextId("col"), name: (body && body.name) || "Collectible", kind: (body && body.kind) || "badge", state: "minted", transfer_policy: (body && body.transfer_policy) || "transferable_between_users", owner_id: ME };
    db.collectibles.push(c);
    return ok(c, 201);
  });
  on("POST", "/api/tasks/:id/collectible-reward", (p, _url, body) => {
    const t = findTask(p.id); if (!t) return err(404, "task not found");
    const c = db.collectibles.find((x) => x.id === (body && body.collectible_id));
    if (!c) return err(404, "collectible not found");
    c.state = "escrowed"; t.reward_collectible_count += 1; t.reward_kind = t.reward_credit_amount > 0 ? "bundle" : "collectible";
    return ok(c, 201);
  });

  on("GET", "/api/organizations", () => ok({ organizations: db.organizations }));
  on("POST", "/api/organizations", (_p, _url, body) => { const o = { id: nextId("org"), name: (body && body.name) || "Org", created_by: ME }; db.organizations.push(o); db.members[o.id] = [{ id: nextId("mem"), organization_id: o.id, user_id: ME, status: "active", roles: ["owner"] }]; db.orgTeams[o.id] = []; return ok(o, 201); });
  on("GET", "/api/organizations/:id/members", (p) => ok({ members: db.members[p.id] || [] }));
  on("POST", "/api/organizations/:id/members", (p, _url, body) => { const m = { id: nextId("mem"), organization_id: p.id, user_id: nextId("user"), status: "active", roles: (body && body.roles) || ["member"] }; (db.members[p.id] = db.members[p.id] || []).push(m); return ok(m, 201); });
  on("GET", "/api/organizations/:id/teams", (p) => ok({ teams: db.orgTeams[p.id] || [] }));
  on("POST", "/api/organizations/:id/teams", (p, _url, body) => { const team = { id: nextId("team"), owner_kind: "organization", organization_id: p.id, owner_user_id: "", name: (body && body.name) || "Team", created_by: ME }; (db.orgTeams[p.id] = db.orgTeams[p.id] || []).push(team); return ok(team, 201); });

  on("GET", "/api/teams", () => ok({ teams: db.standaloneTeams }));
  on("POST", "/api/teams", (_p, _url, body) => { const team = { id: nextId("team"), owner_kind: "user", organization_id: "", owner_user_id: ME, name: (body && body.name) || "Team", created_by: ME }; db.standaloneTeams.push(team); db.teamMembers[team.id] = []; return ok(team, 201); });
  on("GET", "/api/teams/:id", (p) => {
    const team = db.standaloneTeams.concat(...Object.values(db.orgTeams)).find((x) => x.id === p.id);
    if (!team) return err(404, "team not found");
    return ok({ team, members: db.teamMembers[p.id] || [] });
  });
  on("POST", "/api/teams/:id/members", (p, _url, body) => {
    const team = db.standaloneTeams.concat(...Object.values(db.orgTeams)).find((x) => x.id === p.id);
    if (!team) return err(404, "team not found");
    (db.teamMembers[p.id] = db.teamMembers[p.id] || []).push(nextId("user"));
    return ok({ team, members: db.teamMembers[p.id] }, 201);
  });

  on("GET", "/api/task-series", () => ok({ series: db.series }));
  on("GET", "/api/task-series/:id", (p) => { const s = db.series.find((x) => x.id === p.id); return s ? ok(s) : err(404, "series not found"); });
  on("GET", "/api/users/:id", (p) => ok({ id: p.id, tasks: db.tasks.filter((t) => t.created_by === p.id).map(listItem) }));
  on("GET", "/api/users/:id/work", (p) => ok({ tasks: db.tasks.filter((t) => activeAssignee(t) === p.id).map(listItem) }));
  on("GET", "/api/users/:id/submissions", () => ok({ submissions: [] }));
  on("GET", "/api/submission-receipts/:token", () => ok({ submission: { id: "sub-receipt", task_id: "task-1", submitter_id: ME, state: "submitted", response_json: "{}", review_note: "", validation_errors: [] } }));

  const base = (window.location.origin && window.location.origin !== "null") ? window.location.origin : "http://demo.local";
  function resolve(method, rawUrl, rawBody) {
    let url;
    try { url = new URL(rawUrl, base); } catch (_) { return null; }
    if (!url.pathname.startsWith("/api/")) return null;
    const found = match(method, url.pathname);
    if (!found) {
      console.warn("[demo-backend] unhandled", method, url.pathname);
      return Promise.resolve(ok({})); // degrade gracefully, never crash the client
    }
    let body = null;
    if (rawBody && typeof rawBody === "string") { try { body = JSON.parse(rawBody); } catch (_) { body = null; } }
    try {
      return Promise.resolve(found.handler(found.params, url, body));
    } catch (e) {
      console.error("[demo-backend] handler error", method, url.pathname, e);
      return Promise.resolve(err(500, "demo backend error"));
    }
  }

  // elm/http uses XMLHttpRequest, so we intercept XHR (not fetch). For /api/*
  // requests we synthesize a response from the in-memory backend; anything else
  // delegates to the real XHR (e.g. nothing, in practice).
  const RealXHR = window.XMLHttpRequest;
  function DemoXHR() {
    this._listeners = {};
    this._real = null;
    this.readyState = 0;
    this.status = 0;
    this.statusText = "";
    this.responseText = "";
    this.response = "";
    this.responseType = "";
  }
  DemoXHR.prototype.open = function (method, url) {
    this._method = (method || "GET").toUpperCase();
    this._url = url;
    this._intercept = (function () { try { return new URL(url, base).pathname.startsWith("/api/"); } catch (_) { return false; } })();
    if (!this._intercept) { this._real = new RealXHR(); this._real.open.apply(this._real, arguments); }
  };
  DemoXHR.prototype.setRequestHeader = function (k, v) { if (this._real) this._real.setRequestHeader(k, v); };
  DemoXHR.prototype.getAllResponseHeaders = function () { return this._real ? this._real.getAllResponseHeaders() : "content-type: application/json\r\n"; };
  DemoXHR.prototype.getResponseHeader = function (name) {
    if (this._real) return this._real.getResponseHeader(name);
    return name.toLowerCase() === "content-type" ? "application/json" : null;
  };
  DemoXHR.prototype.addEventListener = function (type, fn) { (this._listeners[type] = this._listeners[type] || []).push(fn); if (this._real) this._real.addEventListener(type, fn); };
  DemoXHR.prototype.removeEventListener = function (type, fn) { if (this._real) this._real.removeEventListener(type, fn); };
  DemoXHR.prototype.abort = function () { if (this._real) this._real.abort(); };
  DemoXHR.prototype._emit = function (type) {
    const ev = { type: type, target: this, currentTarget: this };
    if (typeof this["on" + type] === "function") this["on" + type](ev);
    (this._listeners[type] || []).forEach((fn) => { try { fn.call(this, ev); } catch (e) { console.error(e); } });
  };
  DemoXHR.prototype.send = function (body) {
    if (this._real) {
      // mirror real XHR back onto this façade for non-/api requests
      const self = this;
      ["load", "error", "abort", "timeout", "loadend", "readystatechange"].forEach((t) =>
        self._real.addEventListener(t, function () {
          self.readyState = self._real.readyState; self.status = self._real.status;
          self.statusText = self._real.statusText; self.responseText = self._real.responseText;
          self.response = self._real.response;
        }));
      return this._real.send(body);
    }
    const self = this;
    resolve(this._method, this._url, body).then(function (result) {
      const res = result || ok({});
      self.status = res.status;
      self.statusText = res.status >= 400 ? "Error" : "OK";
      self.responseText = res.body || "";
      self.response = res.body || "";
      self.readyState = 4;
      self._emit("readystatechange");
      self._emit("load");
      self._emit("loadend");
    });
  };
  window.XMLHttpRequest = DemoXHR;
})();
