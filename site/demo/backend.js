// In-browser fake Sharecrop backend for the demo.
//
// The demo serves the REAL compiled Elm client (web/static/app.js). This script
// intercepts XMLHttpRequest and answers the /api/* calls the client makes from an
// in-memory, stateful store seeded with realistic agentic-work tasks — so the
// demo is the same code as the shipped client, with no backend, and cannot drift
// from the real UI. It is intentionally a demo fake: tokens are local demo
// bearer strings mapped to users, and behavior mirrors the Go backend's
// contracts closely enough to drive every screen.
(function () {
  "use strict";

  const ME = "user-mara";
  const DEMO_USERS = [
    { id: ME, email: "mara@sharecrop.demo", status: "active" },
    { id: "user-jules", email: "jules@sharecrop.demo", status: "active" },
    { id: "user-ren", email: "ren@sharecrop.demo", status: "active" },
    { id: "user-tala", email: "tala@sharecrop.demo", status: "active" },
    { id: "user-sol", email: "sol@sharecrop.demo", status: "active" },
  ];

  // The 25 default collectibles (mirrors internal/assets/catalog.go). All are
  // tradeable; kind doubles as rarity (badge=common, edition=rare, unique=legendary).
  const catalogEntry = (slug, name, kind) => ({
    slug,
    name,
    kind,
    transfer_policy: "transferable_between_users",
    art: slug,
  });
  const COLLECTIBLE_CATALOG = [
    catalogEntry("harvest-star", "Harvest Star", "badge"),
    catalogEntry("golden-sickle", "Golden Sickle", "badge"),
    catalogEntry("seedling", "Seedling", "badge"),
    catalogEntry("sun-token", "Sun Token", "badge"),
    catalogEntry("rain-drop", "Rain Drop", "badge"),
    catalogEntry("wheat-sheaf", "Wheat Sheaf", "badge"),
    catalogEntry("red-barn", "Red Barn", "badge"),
    catalogEntry("scarecrow", "Scarecrow", "badge"),
    catalogEntry("honey-pot", "Honey Pot", "badge"),
    catalogEntry("pumpkin", "Pumpkin", "badge"),
    catalogEntry("apple", "Apple", "badge"),
    catalogEntry("carrot", "Carrot", "badge"),
    catalogEntry("beehive", "Beehive", "badge"),
    catalogEntry("windmill", "Windmill", "badge"),
    catalogEntry("tractor", "Tractor", "badge"),
    catalogEntry("silver-plow", "Silver Plow", "edition"),
    catalogEntry("golden-egg", "Golden Egg", "edition"),
    catalogEntry("prize-cow", "Prize Cow", "edition"),
    catalogEntry("lucky-clover", "Lucky Clover", "edition"),
    catalogEntry("full-moon-harvest", "Full-Moon Harvest", "edition"),
    catalogEntry("cornucopia", "Cornucopia", "unique"),
    catalogEntry("first-harvest-trophy", "First-Harvest Trophy", "unique"),
    catalogEntry("founders-seed", "Founder's Seed", "unique"),
    catalogEntry("rainbow-field", "Rainbow Field", "unique"),
    catalogEntry("golden-combine", "Golden Combine", "unique"),
  ];

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
        items: {
          kind: "string",
          enum: ["bug", "billing", "feature_request", "account", "other"],
        },
      },
    },
  });
  const ledgerSchema = JSON.stringify({
    kind: "object",
    fields: {
      suspicious_ids: {
        kind: "array",
        required: true,
        items: { kind: "string" },
      },
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
            category: {
              kind: "string",
              required: true,
              enum: ["travel", "meals", "software", "office"],
            },
            total: { kind: "decimal_string", required: true },
          },
        },
      },
    },
  });
  const dateSchema = JSON.stringify({
    kind: "object",
    fields: {
      iso_dates: {
        kind: "array",
        required: true,
        minItems: 8,
        maxItems: 8,
        items: { kind: "string" },
      },
    },
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

  function positiveIntParam(url, name, defaultValue) {
    const raw = url.searchParams.get(name);
    if (raw === null || raw === "") return defaultValue;
    const value = parseInt(raw, 10);
    if (!Number.isInteger(value) || String(value) !== raw || value < 1) {
      throw new Error(`invalid ${name} query parameter`);
    }
    return value;
  }

  function nonNegativeIntParam(url, name) {
    const raw = url.searchParams.get(name);
    if (raw === null || raw === "") return 0;
    const value = parseInt(raw, 10);
    if (!Number.isInteger(value) || String(value) !== raw || value < 0) {
      throw new Error(`invalid ${name} query parameter`);
    }
    return value;
  }

  function selectorPage(url, items, matches) {
    const query = (url.searchParams.get("query") || "").trim().toLowerCase();
    const limit = Math.min(100, positiveIntParam(url, "limit", 50));
    const offset = nonNegativeIntParam(url, "offset");
    return items
      .filter((item) => query === "" || matches(item, query))
      .slice(offset, offset + limit);
  }

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
    balance: 1250,
    ledger: [
      { id: "entry-1", kind: "signup_grant", amount: 1500, task_id: "" },
      { id: "entry-2", kind: "task_escrow", amount: -260, task_id: "task-5" },
      { id: "entry-3", kind: "task_payout", amount: -30, task_id: "task-7" },
      { id: "entry-4", kind: "task_tip", amount: -5, task_id: "task-7" },
      { id: "entry-5", kind: "task_refund", amount: 45, task_id: "task-8" },
    ],
    collectibles: [
      {
        id: "col-1",
        name: "Harvest Star",
        kind: "badge",
        state: "minted",
        transfer_policy: "transferable_between_users",
        owner_id: ME,
        owner_kind: "user",
        organization_id: "",
        art: "harvest-star",
      },
      {
        id: "col-2",
        name: "Golden Sickle",
        kind: "badge",
        state: "minted",
        transfer_policy: "transferable_between_users",
        owner_id: ME,
        owner_kind: "user",
        organization_id: "",
        art: "golden-sickle",
      },
    ],
    organizations: [{
      id: "org-lattice",
      name: "Lattice Field Co",
      created_by: ME,
    }],
    orgBalances: { "org-lattice": 7200 },
    members: {
      "org-lattice": [{
        id: "mem-1",
        organization_id: "org-lattice",
        user_id: ME,
        status: "active",
        roles: ["owner"],
      }],
    },
    orgTeams: {
      "org-lattice": [{
        id: "team-survey",
        owner_kind: "organization",
        organization_id: "org-lattice",
        owner_user_id: "",
        name: "Survey crew",
        created_by: ME,
      }],
    },
    standaloneTeams: [{
      id: "team-field",
      owner_kind: "user",
      organization_id: "",
      owner_user_id: ME,
      name: "Field hands",
      created_by: ME,
    }],
    teamMembers: {
      "team-survey": ["user-jules", "user-tala"],
      "team-field": ["user-tala"],
    },
    credentials: [
      {
        id: "cred-1",
        label: "Sol's field agent",
        scopes: ["tasks_read", "submissions_write"],
        state: "active",
      },
      {
        id: "cred-2",
        label: "Lattice reviewer agent",
        scopes: ["tasks_read", "submissions_review"],
        state: "active",
      },
    ],
    series: [{
      id: "series-orchard",
      owner_kind: "user",
      title: "Orchard intake",
      description: "A multi-step orchard onboarding with review rounds.",
      state: "published",
      created_by: ME,
      position: 0,
    }],
    seriesComments: {
      "series-orchard": [{
        id: "scom-1",
        series_id: "series-orchard",
        author_user_id: ME,
        body: "Kicking off round one — add the intake tasks here.",
        created_at: "2026-06-20T10:00:00Z",
      }],
    },
    taskComments: {
      "task-7": [{
        id: "tcom-seed",
        task_id: "task-7",
        author_user_id: "user-jules",
        body: "Keep each note to one sentence; link the PR for each entry.",
        created_at: "2026-06-22T09:00:00Z",
      }],
    },
    submissionComments: {
      "sub-4-sol": [{
        id: "subcom-seed",
        submission_id: "sub-4-sol",
        author_user_id: ME,
        body:
          "Totals look right — can you double-check the meals category rounding?",
        created_at: "2026-06-22T11:00:00Z",
      }],
    },
    users: DEMO_USERS.slice(),
    accountTokens: {},
    accessTokens: { "demo-access-token": ME },
    appliedFunding: {},
    balances: {
      [ME]: 1250,
      "user-jules": 100,
      "user-ren": 100,
      "user-tala": 100,
      "user-sol": 100,
    },
    notifications: [{
      id: "notif-seed",
      recipient_user_id: ME,
      actor_user_id: "user-sol",
      kind: "submission_created",
      subject_kind: "submission",
      subject_id: "sub-4-sol",
      state: "unread",
      metadata_json: '{"task_id":"task-4"}',
      created_at: "2026-06-22T10:45:00Z",
    }],
    tasks: [],
  };

  // Every task is self-contained: all data needed to solve it is embedded in the
  // task itself (rendered as the "Task input" block), with no external lookups,
  // attachments, or live data required.
  db.tasks = [
    task({
      id: "task-1",
      title: "Extract line items from 6 vendor invoices",
      owner_id: "user-jules",
      created_by: "user-jules",
      description:
        'The Task input below contains the OCR\'d text of 6 vendor invoices (one string per invoice). For each invoice, return invoice_id, vendor, total, and due_date. Rules: total is the numeric grand total with the currency symbol and thousands separators removed (e.g. "$1,240.55" -> "1240.55"); due_date is the written date converted to YYYY-MM-DD. Keep the invoices in the given order.',
      reward_credit_amount: 80,
      escrow: 80,
      response_schema_json: invoiceSchema,
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
      id: "task-2",
      title: "Classify 8 support tickets by category",
      owner_id: "user-ren",
      created_by: "user-ren",
      description:
        "Assign each of the 8 support tickets in the Task input exactly one category from this set: bug, billing, feature_request, account, other. Return a labels array with one label per ticket, in the same order as the input.",
      reward_credit_amount: 45,
      escrow: 45,
      participation_policy: "open",
      response_schema_json: ticketSchema,
      payload_kind: "inline",
      payload_json: pretty({
        tickets: [
          "1. You charged my card again after I cancelled my plan last month",
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
      id: "task-3",
      title: "Verify 10 ledger transfers for fraud signals",
      description:
        "Review the 10 transfers in the Task input. Flag a transfer's id as suspicious if EITHER (a) its item_id appears in more than one transfer, OR (b) its to_account is in banned_accounts. Return suspicious_ids (the flagged transfer ids, in input order) and reviewed_count (the number of transfers you reviewed, which is 10).",
      reward_credit_amount: 60,
      escrow: 60,
      visibility_kind: "organization",
      visibility_id: "org-lattice",
      response_schema_json: ledgerSchema,
      assignee_scope: "user",
      payload_kind: "inline",
      payload_json: pretty({
        banned_accounts: ["ACC-666", "ACC-999"],
        transfers: [
          {
            id: "TR-01",
            item_id: "ITM-A",
            from_account: "ACC-100",
            to_account: "ACC-200",
            amount: "120.00",
          },
          {
            id: "TR-02",
            item_id: "ITM-B",
            from_account: "ACC-101",
            to_account: "ACC-201",
            amount: "75.50",
          },
          {
            id: "TR-03",
            item_id: "ITM-C",
            from_account: "ACC-102",
            to_account: "ACC-666",
            amount: "300.00",
          },
          {
            id: "TR-04",
            item_id: "ITM-D",
            from_account: "ACC-103",
            to_account: "ACC-202",
            amount: "44.00",
          },
          {
            id: "TR-05",
            item_id: "ITM-A",
            from_account: "ACC-104",
            to_account: "ACC-203",
            amount: "120.00",
          },
          {
            id: "TR-06",
            item_id: "ITM-E",
            from_account: "ACC-105",
            to_account: "ACC-204",
            amount: "210.10",
          },
          {
            id: "TR-07",
            item_id: "ITM-F",
            from_account: "ACC-106",
            to_account: "ACC-205",
            amount: "9.99",
          },
          {
            id: "TR-08",
            item_id: "ITM-G",
            from_account: "ACC-107",
            to_account: "ACC-999",
            amount: "1500.00",
          },
          {
            id: "TR-09",
            item_id: "ITM-H",
            from_account: "ACC-108",
            to_account: "ACC-206",
            amount: "62.00",
          },
          {
            id: "TR-10",
            item_id: "ITM-E",
            from_account: "ACC-109",
            to_account: "ACC-207",
            amount: "210.10",
          },
        ],
      }),
      reservations: [{
        id: "res-3-tala",
        task_id: "task-3",
        assignee_kind: "user",
        assignee_id: "user-tala",
        state: "requested",
        requested_by: "user-tala",
      }],
      participation_policy: "approval_required",
      availability_kind: "awaiting_approval",
    }),
    task({
      id: "task-4",
      title: "Categorize and total 6 expense lines",
      description:
        "Assign each of the 6 expense lines in the Task input to exactly one category (travel, meals, software, office), then return category_totals: for every category that has at least one expense, the sum of its amounts as a decimal string with two places. Amounts are given in the input; sum them exactly.",
      reward_credit_amount: 25,
      reward_kind: "bundle",
      reward_collectible_count: 1,
      escrow: 25,
      response_schema_json: expenseSchema,
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
      reservations: [{
        id: "res-4-sol",
        task_id: "task-4",
        assignee_kind: "user",
        assignee_id: "user-sol",
        state: "active",
        requested_by: "user-sol",
      }],
      submissions: [{
        id: "sub-4-sol",
        task_id: "task-4",
        submitter_id: "user-sol",
        state: "submitted",
        response_json: pretty({
          category_totals: [
            { category: "travel", total: "78.60" },
            { category: "meals", total: "100.75" },
            { category: "software", total: "144.00" },
            { category: "office", total: "23.40" },
          ],
        }),
        review_note: "",
        validation_errors: [],
        via_agent: true,
      }],
      availability_kind: "reserved",
    }),
    task({
      id: "task-5",
      title: "Normalize 8 dates to ISO 8601",
      series_kind: "existing_series",
      series_id: "series-orchard",
      series_position: 1,
      description:
        'Convert each of the 8 dates in the Task input to ISO 8601 (YYYY-MM-DD) and return them, in order, as an iso_dates array. Rules: if a date leads with a 4-digit number, treat it as the year (YYYY/MM/DD or YYYY.MM.DD); otherwise, for ambiguous all-numeric dates, treat the format as month-first (so "03/04/2026" is March 4, 2026 -> "2026-03-04"). Slash, dash, and dot separators all appear.',
      reward_credit_amount: 30,
      escrow: 30,
      response_schema_json: dateSchema,
      payload_kind: "inline",
      payload_json: pretty({
        dates: [
          "March 5, 2026",
          "03/04/2026",
          "2026/12/01",
          "7 Jan 2026",
          "11-02-2026",
          "Apr 30 2026",
          "2026.06.18",
          "09/10/2026",
        ],
      }),
    }),
    task({
      id: "task-6",
      title: "Extract product and rating from 5 reviews",
      participation_policy: "open",
      series_kind: "existing_series",
      series_id: "series-orchard",
      series_position: 2,
      description:
        'Each of the 5 review lines in the Task input starts with "Rating: N/5" and then, after an em dash, names a product. The product name is the proper-noun phrase between the em dash and the next colon (e.g. "Orchard Boots"). Return an items array, in order, with that product name and the rating N as an integer (1-5).',
      reward_credit_amount: 36,
      escrow: 36,
      response_schema_json: reviewSchema,
      payload_kind: "inline",
      payload_json: pretty({
        reviews: [
          "Rating: 4/5 — Orchard Boots: great grip, runs a half size small",
          "Rating: 2/5 — Field Gloves: wore through at the seams in a month",
          "Rating: 5/5 — Sun Hat: the best I've owned, packs flat",
          "Rating: 3/5 — Canvas Tote: sturdy but the strap is too short",
          "Rating: 1/5 — Rain Shell: soaked through in light drizzle",
        ],
      }),
    }),
    task({
      id: "task-7",
      title: "Write release notes for 5 changelog entries",
      owner_id: "user-jules",
      created_by: "user-jules",
      participation_policy: "open",
      task_type: "product_review",
      reference_url: "https://github.com/example/app/releases/tag/v2.0",
      description:
        "Turn each of the 5 raw changelog entries in the Task input into one customer-facing release note: a single sentence, plain language, no internal ticket ids. Return a JSON object with a notes array of 5 strings, in the same order as the input. (This task uses a freeform response schema, so the exact shape is up to you — just keep it to the 5 notes in order.)",
      reward_credit_amount: 20,
      escrow: 20,
      response_schema_json: '{"kind":"freeform"}',
      payload_kind: "inline",
      payload_json: pretty({
        changelog: [
          "PROj-412: fix null deref when exporting empty report",
          "PROj-418: add CSV export button to the Reports tab",
          "PROj-421: bump session timeout from 15m to 60m",
          "PROj-430: dark mode for the dashboard",
          "PROj-435: 2x faster invoice search via new index",
        ],
      }),
    }),
  ];

  // --- shape helpers (mirror the Go DTOs the Elm client decodes) ---
  function listItem(t, actorId) {
    const viewer = actorId || ME;
    return {
      id: t.id,
      owner_kind: t.owner_kind,
      title: t.title,
      reward_kind: t.reward_kind,
      reward_credit_amount: t.reward_credit_amount,
      reward_collectible_count: t.reward_collectible_count,
      participation_policy: t.participation_policy,
      assignee_scope: t.assignee_scope,
      reservation_expiry_hours: t.reservation_expiry_hours,
      state: t.state,
      visibility_kind: t.visibility_kind,
      availability_kind: t.availability_kind,
      viewer_action: viewerAction(t),
      reviewer_action: reviewerAction(t, viewer),
      created_by: t.created_by,
      active_assignee_kind: activeAssignee(t) ? "user" : "",
      active_assignee_id: activeAssignee(t),
    };
  }
  function detail(t, actorId) {
    const viewer = actorId || ME;
    return {
      id: t.id,
      owner_kind: t.owner_kind,
      owner_id: t.owner_id,
      title: t.title,
      description: t.description,
      task_type: t.task_type || "general",
      reference_url: t.reference_url || "",
      reward_kind: t.reward_kind,
      reward_credit_amount: t.reward_credit_amount,
      reward_collectible_count: t.reward_collectible_count,
      participation_policy: t.participation_policy,
      assignee_scope: t.assignee_scope,
      reservation_expiry_hours: t.reservation_expiry_hours,
      state: t.state,
      visibility_kind: t.visibility_kind,
      visibility_id: t.visibility_id,
      series_kind: t.series_kind,
      series_id: t.series_id,
      series_position: t.series_position,
      response_schema_json: t.response_schema_json,
      payload_kind: t.payload_kind,
      payload_json: t.payload_json,
      created_by: t.created_by,
      availability_kind: t.availability_kind,
      viewer_action: viewerAction(t),
      reviewer_action: reviewerAction(t, viewer),
    };
  }
  function activeAssignee(t) {
    const active = t.reservations.find((r) => r.state === "active");
    return active ? active.assignee_id : "";
  }
  // Mirrors the real backend's taskViewerAction: a pure function of state +
  // participation policy (viewer-independent), so the demo matches production.
  function viewerAction(t) {
    if (t.state !== "open") return "none";
    if (t.participation_policy === "open") return "submit";
    if (t.participation_policy === "reservation_required") return "reserve";
    if (t.participation_policy === "approval_required") {
      return "request_approval";
    }
    return "none";
  }
  function reviewerAction(t, actorId) {
    if (t.created_by === actorId) return "review";
    if (t.owner_kind !== "organization") return "none";
    const membership = (db.members[t.owner_id] || []).find((m) =>
      m.user_id === actorId && m.status === "active"
    );
    if (!membership) return "none";
    return membership.roles.some((role) =>
        ["owner", "admin", "reviewer"].includes(role)
      )
      ? "review"
      : "none";
  }

  // Minimal validator matching the response-schema format the designer emits
  // (kind: object|array|string|integer|decimal_string|freeform). Returns the
  // same {path, message} validation errors the real backend produces.
  function validateValue(schema, value, path) {
    if (!schema || schema.kind === "freeform") return [];
    switch (schema.kind) {
      case "object": {
        if (
          typeof value !== "object" || value === null || Array.isArray(value)
        ) return [{ path, message: "value must be an object" }];
        const errs = [];
        // Support both schema dialects: the designer/seed format (fields is a
        // map of name -> subschema with a `required` flag) and the canonical
        // contract format (fields is an array of {name, presence, schema}).
        const fields = Array.isArray(schema.fields)
          ? schema.fields.map((f) => ({
            name: f.name,
            required: f.presence === "required" || f.required === true,
            sub: f.schema || f,
          }))
          : Object.entries(schema.fields || {}).map(([name, f]) => ({
            name,
            required: f.required === true || f.presence === "required",
            sub: f,
          }));
        for (const field of fields) {
          const fp = path ? path + "." + field.name : field.name;
          if (!(field.name in value)) {
            if (field.required) {
              errs.push({ path: fp, message: "required field is missing" });
            }
            continue;
          }
          errs.push(...validateValue(field.sub, value[field.name], fp));
        }
        return errs;
      }
      case "array": {
        if (!Array.isArray(value)) {
          return [{ path, message: "value must be an array" }];
        }
        const errs = [];
        if (schema.minItems != null && value.length < schema.minItems) {
          errs.push({
            path,
            message: `expected at least ${schema.minItems} items`,
          });
        }
        if (schema.maxItems != null && value.length > schema.maxItems) {
          errs.push({
            path,
            message: `expected at most ${schema.maxItems} items`,
          });
        }
        const itemSchema = schema.item || schema.items;
        value.forEach((v, i) =>
          errs.push(...validateValue(itemSchema, v, `${path}[${i}]`))
        );
        return errs;
      }
      case "enum": {
        if (typeof value !== "string") {
          return [{ path, message: "value must be a string" }];
        }
        return (schema.values || []).includes(value) ? [] : [{
          path,
          message: `must be one of: ${(schema.values || []).join(", ")}`,
        }];
      }
      case "string":
      case "decimal_string": {
        if (typeof value !== "string") {
          return [{ path, message: "value must be a string" }];
        }
        if (schema.enum && !schema.enum.includes(value)) {
          return [{
            path,
            message: `must be one of: ${schema.enum.join(", ")}`,
          }];
        }
        if (
          schema.kind === "decimal_string" && !/^-?\d+(\.\d+)?$/.test(value)
        ) return [{ path, message: "must be a decimal string" }];
        return [];
      }
      case "integer":
        return (typeof value === "number" && Number.isInteger(value))
          ? []
          : [{ path, message: "value must be an integer" }];
      default:
        return [];
    }
  }

  // --- routing ---
  const routes = [];
  const on = (method, pattern, handler) =>
    routes.push({ method, parts: pattern.split("/"), handler });
  const ok = (body, status) => ({
    status: status || 200,
    body: JSON.stringify(body),
  });
  const empty = (status) => ({ status: status || 204, body: "" });
  const err = (status, message) => ({
    status,
    body: JSON.stringify({ error: message }),
  });
  const auth = (userId, token) => ({
    subject_kind: "user",
    subject_id: userId,
    access_token: token,
    role: "admin",
  });
  const findTask = (id) => db.tasks.find((t) => t.id === id);
  function accessTokenForUser(userId) {
    const token = "demo-access-" + userId + "-" + nextId("tok");
    db.accessTokens[token] = userId;
    return token;
  }
  function actorFromHeaders(headers) {
    const authorization = headers &&
      (headers.Authorization || headers.authorization);
    if (!authorization) return null;
    const prefix = "Bearer ";
    if (authorization.slice(0, prefix.length) !== prefix) return null;
    return db.accessTokens[authorization.slice(prefix.length)] || null;
  }
  function ensureActor(actorId) {
    return actorId && db.users.some((u) => u.id === actorId) ? actorId : null;
  }
  function balanceFor(userId) {
    if (!Object.prototype.hasOwnProperty.call(db.balances, userId)) {
      db.balances[userId] = 100;
    }
    return db.balances[userId];
  }
  function adjustUserBalance(userId, amount) {
    db.balances[userId] = balanceFor(userId) + amount;
  }
  function findOrCreateUserByEmail(email) {
    const clean = String(email || "").trim().toLowerCase();
    if (clean === "") return null;
    let user = db.users.find((u) => u.email.toLowerCase() === clean);
    if (!user) {
      user = { id: nextId("user"), email: clean, status: "active" };
      db.users.push(user);
      db.balances[user.id] = 100;
    }
    return user;
  }
  function accountToken(kind, actorId) {
    const token = "demo-" + kind + "-" + nextId("tok");
    db.accountTokens[token] = { kind, user_id: actorId || ME };
    return token;
  }
  function consumeAccountToken(token, kind) {
    const record = db.accountTokens[token || ""];
    if (!record || record.kind !== kind) return false;
    delete db.accountTokens[token];
    return true;
  }
  function notify(
    recipientUserId,
    actorUserId,
    kind,
    subjectKind,
    subjectId,
    metadata,
  ) {
    if (recipientUserId === actorUserId) return null;
    const notification = {
      id: nextId("notif"),
      recipient_user_id: recipientUserId,
      actor_user_id: actorUserId,
      kind,
      subject_kind: subjectKind,
      subject_id: subjectId,
      state: "unread",
      metadata_json: JSON.stringify(metadata || {}),
      created_at: new Date().toISOString(),
    };
    db.notifications.unshift(notification);
    return notification;
  }

  function match(method, path) {
    const segs = path.split("?")[0].split("/").filter((s) => s !== "");
    for (const route of routes) {
      if (route.method !== method) continue;
      const rp = route.parts.filter((s) => s !== "");
      if (rp.length !== segs.length) continue;
      const params = {};
      let hit = true;
      for (let i = 0; i < rp.length; i++) {
        if (rp[i].startsWith(":")) {
          params[rp[i].slice(1)] = decodeURIComponent(segs[i]);
        } else if (rp[i] !== segs[i]) {
          hit = false;
          break;
        }
      }
      if (hit) return { handler: route.handler, params };
    }
    return null;
  }
  function allowsAnonymousRoute(method, pathname) {
    return (method === "POST" && (
      pathname === "/api/auth/refresh" ||
      pathname === "/api/auth/login" ||
      pathname === "/api/auth/register" ||
      pathname === "/api/auth/guest" ||
      pathname === "/api/auth/email-verification/confirm" ||
      pathname === "/api/auth/password-reset/request" ||
      pathname === "/api/auth/password-reset/confirm"
    ));
  }

  // Auth: refresh auto-succeeds so the demo boots straight into the seeded app.
  on("POST", "/api/auth/refresh", () => ok(auth(ME, "demo-access-token")));
  on("POST", "/api/auth/login", (_p, _url, body) => {
    const user = findOrCreateUserByEmail(body && body.email);
    if (!user) return err(400, "email is required");
    return ok(auth(user.id, accessTokenForUser(user.id)));
  });
  on("POST", "/api/auth/register", (_p, _url, body) => {
    const user = findOrCreateUserByEmail(body && body.email);
    if (!user) return err(400, "email is required");
    return ok(auth(user.id, accessTokenForUser(user.id)), 201);
  });
  on("POST", "/api/auth/guest", () => {
    const user = findOrCreateUserByEmail(
      "guest-" + nextId("user") + "@sharecrop.demo",
    );
    return ok(auth(user.id, accessTokenForUser(user.id)), 201);
  });
  on("POST", "/api/auth/logout", () => empty());
  on("POST", "/api/auth/email-verification/confirm", (_p, _url, body) => {
    return consumeAccountToken(body && body.token, "email_verification")
      ? ok({ status: "verified" })
      : err(400, "account token is invalid");
  });
  on("POST", "/api/auth/password-reset/request", (_p, _url, body, actorId) => {
    findOrCreateUserByEmail(body && body.email);
    return ok({ token: accountToken("password_reset", actorId) }, 201);
  });
  on("POST", "/api/auth/password-reset/confirm", (_p, _url, body) => {
    return consumeAccountToken(body && body.token, "password_reset")
      ? ok({ status: "password_reset" })
      : err(400, "account token is invalid");
  });
  on(
    "POST",
    "/api/account/email-verification",
    (_p, _url, _body, actorId) =>
      ok({ token: accountToken("email_verification", actorId) }, 201),
  );
  on(
    "PATCH",
    "/api/account/password",
    () => ok({ status: "password_changed" }),
  );
  on("PATCH", "/api/account/profile", (_p, _url, body, actorId) => {
    const me = db.users.find((u) => u.id === actorId);
    if (me && body && body.email) {
      me.email = String(body.email).trim().toLowerCase();
    }
    return ok({ status: "profile_updated" });
  });
  on("DELETE", "/api/account", () => ok({ status: "deactivated" }));

  on(
    "GET",
    "/api/credits/balance",
    (_p, _url, _body, actorId) => ok({ amount: balanceFor(actorId) }),
  );
  on("GET", "/api/credits/ledger", () => ok({ entries: db.ledger }));

  on("GET", "/api/tasks", (_p, url, _body, actorId) => {
    const scope = url.searchParams.get("scope") || "";
    const includeReserved = url.searchParams.get("include_reserved") === "true";
    const stateFilter = url.searchParams.get("state") || "";
    const orgId = url.searchParams.get("organization_id") || "";
    let list = db.tasks; // default (empty scope): everything visible to the demo user
    if (scope === "user") {
      list = db.tasks.filter((t) => t.created_by === actorId);
    } else if (scope === "public") {
      // Discovery shows open public tasks only (mirrors the real public scope).
      list = db.tasks.filter((t) =>
        t.visibility_kind === "public" && t.state === "open" &&
        (includeReserved || !activeAssignee(t))
      );
    } else if (scope === "organization") {
      list = db.tasks.filter((t) =>
        t.visibility_kind === "organization" &&
        (!orgId || t.visibility_id === orgId)
      );
    }
    if (stateFilter) list = list.filter((t) => t.state === stateFilter);
    return ok({ tasks: list.map((t) => listItem(t, actorId)) });
  });
  on("GET", "/api/tasks/:id", (p, _url, _body, actorId) => {
    const t = findTask(p.id);
    return t ? ok(detail(t, actorId)) : err(404, "task not found");
  });
  on("POST", "/api/tasks", async (_p, _url, body, actorId) => {
    // Honor the owner / visibility scope / assignee / reservation fields the
    // client sends, so an org-owned or org-scoped task is stored as such (and
    // shows up under its org), matching the real backend.
    const owner = (body && body.owner) || {};
    const ownerKind = owner.kind || "user";
    const ownerId = ownerKind === "organization"
      ? (owner.organization_id || "")
      : (owner.user_id || actorId);
    const visibility = (body && body.visibility) || {};
    const visibilityKind = visibility.kind || "public";
    const visibilityId = visibilityKind === "organization"
      ? (visibility.organization_id || "")
      : visibilityKind === "team"
      ? (visibility.team_id || "")
      : visibilityKind === "user"
      ? (visibility.user_id || "")
      : "";
    const participation = (body && body.participation) || {};
    const t = task({
      title: body.title || "Untitled task",
      description: body.description || "",
      owner_kind: ownerKind,
      owner_id: ownerId,
      reward_kind: (body.reward && body.reward.kind) || "none",
      reward_credit_amount: (body.reward && body.reward.credit_amount) || 0,
      participation_policy: participation.policy || "open",
      assignee_scope: participation.assignee_scope || "user",
      reservation_expiry_hours: participation.reservation_expiry_hours || 48,
      visibility_kind: visibilityKind,
      visibility_id: visibilityId,
      response_schema_json: body.response_schema_json || '{"kind":"freeform"}',
      task_type: body.task_type || "general",
      reference_url: body.reference_url || "",
      payload_kind:
        (body.payload && body.payload.kind === "json" && body.payload.json)
          ? "inline"
          : "none",
      payload_json: (body.payload && body.payload.kind === "json")
        ? body.payload.json
        : "",
      state: "draft",
      availability_kind: "available",
      created_by: actorId,
    });
    ((body.reward && body.reward.collectible_ids) || []).forEach((id) => {
      const c = db.collectibles.find((x) =>
        x.id === id && x.owner_id === actorId
      );
      if (c) {
        c.state = "escrowed";
        t.reward_collectible_count += 1;
        t.reward_kind = t.reward_credit_amount > 0 ? "bundle" : "collectible";
      }
    });
    db.tasks.unshift(t);
    return ok(detail(t), 201);
  });
  // Lifecycle transitions are state-machine guarded exactly like the real backend
  // (open: draft only; cancel: draft/open; unpublish: open only).
  on("POST", "/api/tasks/:id/open", (p) => {
    const t = findTask(p.id);
    if (!t) return err(404, "task not found");
    if (t.state !== "draft") return err(409, "only draft tasks can be opened");
    t.state = "open";
    return ok(detail(t));
  });
  on("POST", "/api/tasks/:id/cancel", (p) => {
    const t = findTask(p.id);
    if (!t) return err(404, "task not found");
    if (t.state !== "draft" && t.state !== "open") {
      return err(409, "only draft or open tasks can be cancelled");
    }
    if (
      (t.escrow || 0) > 0 ||
      db.collectibles.some((c) =>
        c.state === "escrowed" && c.owner_id === t.created_by
      )
    ) return err(409, "refund the task's held escrow before cancelling");
    t.state = "cancelled";
    t.availability_kind = "closed";
    return ok(detail(t));
  });
  on("POST", "/api/tasks/:id/unpublish", (p) => {
    const t = findTask(p.id);
    if (!t) return err(404, "task not found");
    if (t.state !== "open") {
      return err(409, "only open tasks can be unpublished");
    }
    t.state = "draft";
    return ok(detail(t));
  });
  on(
    "GET",
    "/api/tasks/:id/comments",
    (p) => ok({ comments: db.taskComments[p.id] || [] }),
  );
  on("POST", "/api/tasks/:id/comments", (p, _url, body, actorId) => {
    const t = findTask(p.id);
    if (!t) return err(404, "task not found");
    const comment = {
      id: nextId("tcom"),
      task_id: t.id,
      author_user_id: actorId,
      body: (body && body.body) || "",
      created_at: "2026-06-25T12:00:00Z",
    };
    (db.taskComments[t.id] = db.taskComments[t.id] || []).push(comment);
    return ok(comment, 201);
  });
  on("POST", "/api/tasks/:id/refund", (p, _url, _body, actorId) => {
    // Unified refund: release held credits AND any escrowed collectibles (so a
    // bundle reward refunds in one shot), then cancel. Mirrors the real backend's
    // /refund, which calls refundHeldCollectibleReward inside the same tx.
    const t = findTask(p.id);
    if (!t) return err(404, "task not found");
    if (t.state !== "draft" && t.state !== "open") {
      return err(409, "only draft or open tasks can be refunded");
    }
    const released = t.escrow || 0;
    if (released === 0) return err(409, "task has no escrow to refund");
    funderAdjust(t, released, actorId);
    db.ledger.push({
      id: nextId("entry"),
      kind: "task_refund",
      amount: released,
      task_id: t.id,
    });
    db.collectibles.filter((c) =>
      c.state === "escrowed" && c.owner_id === t.created_by
    ).forEach((c) => {
      c.state = "minted";
    });
    if (t.reward_collectible_count > 0) {
      t.reward_collectible_count = 0;
      t.reward_kind = "none";
    }
    t.escrow = 0;
    t.state = "cancelled";
    t.availability_kind = "closed";
    return ok({ task_id: t.id, amount: released, state: "refunded" });
  });
  // Refund escrowed collectible rewards back to the requester's holdings
  // (mirrors the real backend's POST /collectible-refund -> RefundReward).
  on("POST", "/api/tasks/:id/collectible-refund", (p, _url, _body, actorId) => {
    const t = findTask(p.id);
    if (!t) return err(404, "task not found");
    if (t.created_by !== actorId) {
      return err(403, "only the task requester can refund rewards");
    }
    if (t.state !== "draft" && t.state !== "open") {
      return err(409, "only draft or open tasks can be refunded");
    }
    const escrowed = db.collectibles.filter((c) =>
      c.state === "escrowed" && c.owner_id === actorId
    );
    if (escrowed.length === 0) {
      return err(409, "task has no escrowed collectibles to refund");
    }
    escrowed.forEach((c) => {
      c.state = "minted";
    });
    t.reward_collectible_count = 0;
    t.reward_kind = t.reward_credit_amount > 0 ? "credit" : "none";
    return ok({ collectibles: escrowed });
  });
  on("POST", "/api/tasks/:id/funding", (p, _url, body, actorId) => {
    const t = findTask(p.id);
    if (!t) return err(404, "task not found");
    const amount = (body && body.amount) || 0;
    // Idempotent funding: replaying the same key does not double-charge (mirrors
    // the real backend's idempotency_key guard).
    const key = body && body.idempotency_key;
    if (key && db.appliedFunding[key]) {
      return ok({ task_id: t.id, amount: t.escrow, state: "held" }, 201);
    }
    // Escrow is a single hold, not additive: re-funding an already-funded task is
    // rejected (matches the real backend's "task is already funded").
    if ((t.escrow || 0) > 0) return err(409, "task is already funded");
    // Org-owned tasks fund from the organization wallet; personal tasks from the
    // user balance. The funder is remembered so refunds/tips return to it.
    const orgId = (body && body.organization_id) || "";
    if (orgId) {
      if (amount > (db.orgBalances[orgId] || 0)) {
        return err(409, "insufficient organization credits to fund the task");
      }
      db.orgBalances[orgId] -= amount;
      t.fundedOrg = orgId;
    } else {
      if (amount > balanceFor(actorId)) {
        return err(409, "insufficient credits to fund the task");
      }
      adjustUserBalance(actorId, -amount);
    }
    // Funding only moves credits into escrow; task state is unchanged (the /open
    // route moves draft -> open). "funded" is not a valid TaskState.
    t.escrow += amount;
    if (key) db.appliedFunding[key] = true;
    return ok({ task_id: t.id, amount: t.escrow, state: "held" }, 201);
  });
  on("POST", "/api/tasks/:id/reservations", (p, _url, body, actorId) => {
    const t = findTask(p.id);
    if (!t) return err(404, "task not found");
    if (t.state !== "open") return err(409, "only open tasks can be reserved");
    if (t.participation_policy === "open") {
      return err(409, "task does not require reservation");
    }
    const assigneeKind = (body && body.assignee_kind) || "user";
    if ((t.assignee_scope || "user") !== assigneeKind) {
      return err(
        409,
        assigneeKind === "organization_team"
          ? "this task does not accept organization team reservations"
          : "this task does not accept user reservations",
      );
    }
    if (t.created_by === actorId) {
      return err(409, "task requester cannot reserve their own task");
    }
    let assigneeId = actorId;
    if (assigneeKind === "organization_team") {
      const orgId = (body && body.organization_id) || "";
      const teamId = (body && body.team_id) || "";
      const team = (db.orgTeams[orgId] || []).find((value) =>
        value.id === teamId
      );
      if (!team || !(db.teamMembers[teamId] || []).includes(actorId)) {
        return err(403, "organization team membership denied");
      }
      assigneeId = teamId;
    }
    const state = t.participation_policy === "approval_required"
      ? "requested"
      : "active";
    const r = {
      id: nextId("res"),
      task_id: t.id,
      assignee_kind: assigneeKind,
      assignee_id: assigneeId,
      state,
      requested_by: actorId,
    };
    t.reservations.push(r);
    if (state === "active") t.availability_kind = "reserved";
    return ok(r, 201);
  });
  on("GET", "/api/tasks/:id/reservations", (p) => {
    const t = findTask(p.id);
    return t
      ? ok({ reservations: t.reservations })
      : err(404, "task not found");
  });
  const reservationChange =
    (state, availability) => (p, _url, _body, actorId) => {
      const t = findTask(p.id);
      if (!t) return err(404, "task not found");
      // Mirrors the real backend: approve/decline/cancel are requester-only.
      if (t.created_by !== actorId) {
        return err(403, "only the task requester can change reservations");
      }
      const r = t.reservations.find((x) => x.id === p.rid);
      if (!r) return err(404, "reservation not found");
      // Only pending/active reservations can transition (matches the store guard).
      if (r.state !== "requested" && r.state !== "active") {
        return err(409, "reservation is not pending or active");
      }
      r.state = state;
      if (availability) t.availability_kind = availability;
      return ok(r);
    };
  on(
    "POST",
    "/api/tasks/:id/reservations/:rid/approve",
    reservationChange("active", "reserved"),
  );
  on(
    "POST",
    "/api/tasks/:id/reservations/:rid/decline",
    reservationChange("declined", "available"),
  );
  on(
    "POST",
    "/api/tasks/:id/reservations/:rid/cancel",
    reservationChange("cancelled_by_requester", "available"),
  );
  on(
    "GET",
    "/api/submissions/:id/comments",
    (p, _url, _body, actorId) => {
      const context = findSubmissionContext(p.id);
      if (!context) return err(404, "submission not found");
      if (!canAccessSubmissionThread(context, actorId)) {
        return err(
          403,
          "only the submitter or reviewer can read submission comments",
        );
      }
      return ok({ comments: db.submissionComments[p.id] || [] });
    },
  );
  on("POST", "/api/submissions/:id/comments", (p, _url, body, actorId) => {
    const context = findSubmissionContext(p.id);
    if (!context) return err(404, "submission not found");
    if (!canAccessSubmissionThread(context, actorId)) {
      return err(
        403,
        "only the submitter or reviewer can add submission comments",
      );
    }
    const c = {
      id: nextId("scom"),
      submission_id: p.id,
      author_user_id: actorId,
      body: (body && body.body) || "",
      created_at: "2026-06-24T10:00:00Z",
    };
    (db.submissionComments[p.id] = db.submissionComments[p.id] || []).push(c);
    const recipient = context.submission.submitter_id === actorId
      ? context.task.created_by
      : context.submission.submitter_id;
    notify(recipient, actorId, "submission_commented", "submission", p.id, {
      task_id: context.task.id,
    });
    return ok(c, 201);
  });
  on("GET", "/api/tasks/:id/submissions", (p) => {
    const t = findTask(p.id);
    return t ? ok({ submissions: t.submissions }) : err(404, "task not found");
  });
  on("POST", "/api/tasks/:id/submissions", (p, _url, body, actorId) => {
    const t = findTask(p.id);
    if (!t) return err(404, "task not found");
    // Reservation eligibility: non-open tasks require an active reservation first
    // (mirrors CheckSubmissionEligibility in the real backend).
    const mineActive = t.reservations.find((r) =>
      r.state === "active" &&
      (r.assignee_id === actorId ||
        (r.assignee_kind === "organization_team" &&
          (db.teamMembers[r.assignee_id] || []).includes(actorId)))
    );
    if (t.participation_policy !== "open" && !mineActive) {
      return err(409, "reserve the task before submitting");
    }
    // Validate the response against the task's response schema; an invalid
    // submission is recorded with state "invalid" + validation_errors (the
    // designer's strict schemas are the whole point, so the demo enforces them).
    const raw = (body && body.response_json) || "";
    let parsed = null, parseFailed = false;
    try {
      parsed = JSON.parse(raw || "null");
    } catch (_) {
      parseFailed = true;
    }
    let schema = null;
    try {
      schema = JSON.parse(t.response_schema_json);
    } catch (_) {
      schema = null;
    }
    const errors = parseFailed
      ? [{ path: "response", message: "response must be valid JSON" }]
      : validateValue(schema, parsed, "response");
    const state = errors.length ? "invalid" : "submitted";
    const s = {
      id: nextId("sub"),
      task_id: t.id,
      submitter_id: actorId,
      state,
      response_json: raw || "{}",
      review_note: "",
      validation_errors: errors,
    };
    t.submissions.push(s);
    if (state === "submitted") {
      t.availability_kind = "reserved"; // availability stays reserved; the submission state carries "submitted"
      if (mineActive) mineActive.state = "submitted";
    }
    notify(t.created_by, actorId, "submission_created", "submission", s.id, {
      task_id: t.id,
    });
    return ok({ submission: s, receipt_token: nextId("receipt") }, 201);
  });
  function pushLedger(kind, amount, taskId) {
    db.ledger.push({ id: nextId("entry"), kind, amount, task_id: taskId });
  }
  // Refunds and tips return to whichever wallet funded the task (org or personal).
  function funderAdjust(t, amount, actorId) {
    if (t.fundedOrg) {
      db.orgBalances[t.fundedOrg] = (db.orgBalances[t.fundedOrg] || 0) + amount;
    } else adjustUserBalance(actorId || t.created_by, amount);
  }
  function activeOrgMember(orgId, userId) {
    return (db.members[orgId] || []).some((member) =>
      member.user_id === userId && member.status === "active"
    );
  }
  function hasOrgReviewPermission(orgId, userId) {
    return (db.members[orgId] || []).some((member) =>
      member.user_id === userId && member.status === "active" &&
      member.roles.some((role) =>
        role === "owner" || role === "admin" || role === "reviewer"
      )
    );
  }
  function canReviewTask(t, actorId) {
    if (t.created_by === actorId) return true;
    if (t.owner_kind === "organization" && t.owner_id) {
      return hasOrgReviewPermission(t.owner_id, actorId);
    }
    return false;
  }
  function findSubmissionContext(submissionId) {
    for (const t of db.tasks) {
      const submission = t.submissions.find((s) => s.id === submissionId);
      if (submission) return { task: t, submission };
    }
    return null;
  }
  function canAccessSubmissionThread(context, actorId) {
    return context.submission.submitter_id === actorId ||
      canReviewTask(context.task, actorId);
  }
  const decide = (state, availability) => (p, _url, body, actorId) => {
    const t = findTask(p.id);
    if (!t) return err(404, "task not found");
    if (!canReviewTask(t, actorId)) {
      return err(403, "only the task owner or an organization reviewer can review submissions");
    }
    const s = t.submissions.find((x) => x.id === p.sid);
    if (!s) return err(404, "submission not found");
    if (s.state !== "submitted") {
      return err(409, "only submitted work can be reviewed");
    }
    // Mirrors the real backend: review actions require an open task.
    if (t.state !== "open") return err(409, "only open tasks can be reviewed");
    s.state = state;
    s.review_note = (body && body.review_note) || "";
    const reservation = t.reservations.find((r) =>
      r.assignee_id === s.submitter_id
    );
    let payout = 0;
    let tip = 0;
    let payoutKind = "none";
    if (state === "changes_requested") {
      // Return the worker to an active reservation so they can resubmit; escrow
      // stays held and no credits move. The task stays open and reserved.
      if (reservation) reservation.state = "active";
      t.availability_kind = availability;
    } else {
      // Rejected: optional partial credit to the worker, the rest refunded, and
      // an optional tip charged from the requester balance. Unlike accept, a
      // rejected task stays OPEN (mirrors the real backend's closeTask: false)
      // and the rejected worker's reservation is released.
      const escrow = t.escrow || 0;
      payout = Math.min((body && body.partial_credit_amount) || 0, escrow);
      const refund = Math.max(0, escrow - payout);
      tip = (body && body.tip_amount) || 0;
      if (payout > 0) {
        adjustUserBalance(s.submitter_id, payout);
        pushLedger("task_payout", -payout, t.id);
        payoutKind = "credit";
      }
      if (refund > 0) {
        funderAdjust(t, refund, actorId);
        pushLedger("task_refund", refund, t.id);
      }
      if (tip > 0) {
        funderAdjust(t, -tip, actorId);
        adjustUserBalance(s.submitter_id, tip);
        pushLedger("task_tip", -tip, t.id);
      }
      t.escrow = 0;
      if (reservation) reservation.state = "cancelled_by_requester";
      t.availability_kind = availability;
    }
    notify(
      s.submitter_id,
      actorId,
      state === "changes_requested"
        ? "submission_changes_requested"
        : "submission_rejected",
      "submission",
      s.id,
      { task_id: t.id },
    );
    return ok({
      task_id: t.id,
      submission_id: s.id,
      state,
      review_note: s.review_note,
      payout_kind: payoutKind,
      payout_amount: payout,
      worker_user_id: payout > 0 ? s.submitter_id : "",
      tip_amount: tip,
    });
  };
  on(
    "POST",
    "/api/tasks/:id/submissions/:sid/accept",
    (p, _url, body, actorId) => {
      const t = findTask(p.id);
      if (!t) return err(404, "task not found");
      if (!canReviewTask(t, actorId)) {
        return err(403, "only the task owner or an organization reviewer can accept submissions for the task");
      }
      const s = t.submissions.find((x) => x.id === p.sid);
      if (!s) return err(404, "submission not found");
      if (s.state !== "submitted") {
        return err(409, "only submitted work can be accepted");
      }
      if (t.state !== "open") {
        return err(
          409,
          "only open tasks can be reviewed",
        );
      }
      if (t.submissions.some((x) => x.state === "accepted")) {
        return err(409, "task already has an accepted submission");
      }
      const escrow = t.escrow || 0;
      const payout = Math.min(
        (body && body.payout_amount) || t.reward_credit_amount || 0,
        escrow,
      );
      const refund = Math.max(0, escrow - payout);
      const tip = (body && body.tip_amount) || 0;
      let payoutKind = "none";
      // The escrow was deducted from the requester wallet at funding time, so a
      // payout leaves the held escrow (no balance change), an unpaid remainder is
      // refunded, and a tip is charged from the current balance. Ledger entries
      // make all of this visible.
      if (payout > 0) {
        adjustUserBalance(s.submitter_id, payout);
        pushLedger("task_payout", -payout, t.id);
        payoutKind = "credit";
      }
      if (refund > 0) {
        funderAdjust(t, refund, actorId);
        pushLedger("task_refund", refund, t.id);
      }
      if (tip > 0) {
        funderAdjust(t, -tip, actorId);
        adjustUserBalance(s.submitter_id, tip);
        pushLedger("task_tip", -tip, t.id);
      }
      // Optional collectible tip: transfer a held collectible from the requester's
      // inventory to the worker (mirrors the real backend's GiftCollectible on accept).
      const tippedIds = [];
      const tipCollectibleId = (body && body.tip_collectible_id) || "";
      if (tipCollectibleId) {
        const c = db.collectibles.find((x) =>
          x.id === tipCollectibleId && x.owner_kind === "user" &&
          x.owner_id === actorId
        );
        if (!c || c.state !== "minted") {
          return err(409, "collectible is not available to tip");
        }
        if (
          c.transfer_policy !== "transferable_between_users" &&
          c.transfer_policy !== "transferable_within_organization"
        ) {
          return err(409, "collectible cannot be tipped");
        }
        if (c.transfer_policy === "transferable_within_organization") {
          if (!c.organization_id) {
            return err(409, "within-organization collectible has no organization");
          }
          if (
            !activeOrgMember(c.organization_id, actorId) ||
            !activeOrgMember(c.organization_id, s.submitter_id)
          ) {
            return err(
              403,
              "within-organization collectible can only be tipped between organization members",
            );
          }
        }
        c.owner_id = s.submitter_id;
        c.owner_kind = "user";
        tippedIds.push(c.id);
        if (payoutKind === "none") payoutKind = "collectible";
        else if (payoutKind === "credit") payoutKind = "bundle";
      }
      s.state = "accepted";
      t.escrow = 0;
      t.availability_kind = "closed";
      t.state = "closed";
      notify(
        s.submitter_id,
        actorId,
        "submission_accepted",
        "submission",
        s.id,
        {
          task_id: t.id,
        },
      );
      return ok({
        task_id: t.id,
        submission_id: s.id,
        payout_kind: payoutKind,
        payout_amount: payout,
        worker_user_id: (payout > 0 || tippedIds.length > 0)
          ? s.submitter_id
          : "",
        collectible_ids: tippedIds,
        tip_amount: tip,
      });
    },
  );
  on(
    "POST",
    "/api/tasks/:id/submissions/:sid/reject",
    decide("rejected", "available"),
  );
  on(
    "POST",
    "/api/tasks/:id/submissions/:sid/request-changes",
    decide("changes_requested", "reserved"),
  );
  on(
    "POST",
    "/api/tasks/:id/capability-tokens",
    (p) =>
      ok({
        id: nextId("cap"),
        task_id: p.id,
        state: "active",
        token: "demo-capability-" + nextId("tok"),
      }, 201),
  );

  on(
    "GET",
    "/api/agent-credentials",
    () => ok({ credentials: db.credentials }),
  );
  on("POST", "/api/agent-credentials", (_p, _url, body) => {
    const cred = {
      id: nextId("cred"),
      label: (body && body.label) || "Agent",
      scopes: (body && body.scopes) || [],
      state: "active",
    };
    db.credentials.push(cred);
    return ok({ credential: cred, secret: "demo-secret-" + nextId("s") }, 201);
  });
  on("POST", "/api/agent-credentials/:id/revoke", (p) => {
    const c = db.credentials.find((x) => x.id === p.id);
    if (!c) return err(404, "credential not found");
    c.state = "revoked";
    return ok(c);
  });

  on(
    "GET",
    "/api/collectibles",
    (_p, _url, _body, actorId) =>
      ok({
        collectibles: db.collectibles.filter((c) => c.owner_id === actorId),
      }),
  );
  on("POST", "/api/collectibles", (_p, _url, body, actorId) => {
    const c = {
      id: nextId("col"),
      name: (body && body.name) || "Collectible",
      kind: (body && body.kind) || "badge",
      state: "minted",
      transfer_policy: (body && body.transfer_policy) ||
        "transferable_between_users",
      owner_id: actorId,
      owner_kind: "user",
      organization_id: "",
      art: (body && body.art) || "",
    };
    db.collectibles.push(c);
    return ok(c, 201);
  });
  // The default-collectible catalog (mirrors internal/assets/catalog.go). Kind
  // doubles as rarity: badge=common, edition=rare, unique=legendary.
  on(
    "GET",
    "/api/collectibles/catalog",
    () => ok({ entries: COLLECTIBLE_CATALOG }),
  );
  // Admin award: mint a fresh copy of a default collectible owned by the
  // recipient (a user, team, or organization). Awarding to yourself makes the
  // copy show up in your holdings immediately.
  on("POST", "/api/collectibles/award", (_p, _url, body) => {
    const slug = body && body.slug;
    const entry = COLLECTIBLE_CATALOG.find((e) => e.slug === slug);
    if (!entry) return err(400, "unknown default collectible");
    const recipientKind = (body && body.recipient_kind) || "user";
    const recipientId = (body && body.recipient_id) || "";
    if (recipientId === "") return err(400, "recipient is required");
    const c = {
      id: nextId("col"),
      name: entry.name,
      kind: entry.kind,
      state: "awarded",
      transfer_policy: entry.transfer_policy,
      owner_id: recipientId,
      owner_kind: recipientKind,
      organization_id: (body && body.organization_id) ||
        (recipientKind === "organization" ? recipientId : ""),
      art: entry.art,
    };
    db.collectibles.push(c);
    return ok(c, 201);
  });
  // Holdings of an organization or team (e.g. defaults an admin awarded to them).
  on(
    "GET",
    "/api/organizations/:id/collectibles",
    (p) =>
      ok({
        collectibles: db.collectibles.filter((c) =>
          c.owner_kind === "organization" && c.owner_id === p.id
        ),
      }),
  );
  on(
    "GET",
    "/api/teams/:id/collectibles",
    (p) =>
      ok({
        collectibles: db.collectibles.filter((c) =>
          c.owner_kind === "team" && c.owner_id === p.id
        ),
      }),
  );
  // Trade: move a collectible to another user.
  on("POST", "/api/collectibles/:id/transfer", (p, _url, body) => {
    const c = db.collectibles.find((x) => x.id === p.id);
    if (!c) return err(404, "collectible not found");
    const recipientId = (body && body.recipient_id) || "";
    if (recipientId === "") return err(400, "recipient is required");
    if (c.transfer_policy !== "transferable_between_users") {
      return err(409, "this collectible cannot be traded");
    }
    c.owner_id = recipientId;
    c.owner_kind = "user";
    return ok(c);
  });
  on("GET", "/api/admin/operations", () =>
    ok({
      status: "ok",
      account_token_delivery: "api",
      mcp_storage: "process_memory",
      rate_limit_storage: "process_memory",
      active_mcp_sessions: 0,
      active_ip_rate_buckets: 0,
      active_subject_rate_buckets: 0,
      secure_cookies: "disabled",
    }));
  on("GET", "/api/admin/audit-events", () =>
    ok({
      events: [],
    }));
  on("GET", "/api/notifications", (_p, _url, _body, actorId) =>
    ok({
      notifications: db.notifications.filter((n) =>
        n.recipient_user_id === actorId
      ),
    }));
  on("POST", "/api/notifications/:id/read", (p, _url, _body, actorId) => {
    const notification = db.notifications.find((n) =>
      n.id === p.id && n.recipient_user_id === actorId
    );
    if (!notification) return err(404, "notification not found");
    notification.state = "read";
    return ok(notification);
  });
  on("POST", "/api/tasks/:id/collectible-reward", (p, _url, body) => {
    const t = findTask(p.id);
    if (!t) return err(404, "task not found");
    const c = db.collectibles.find((x) =>
      x.id === (body && body.collectible_id)
    );
    if (!c) return err(404, "collectible not found");
    c.state = "escrowed";
    t.reward_collectible_count += 1;
    t.reward_kind = t.reward_credit_amount > 0 ? "bundle" : "collectible";
    return ok(c, 201);
  });

  on(
    "GET",
    "/api/organizations",
    (_p, url) =>
      ok({
        organizations: selectorPage(
          url,
          db.organizations,
          (organization, query) =>
            organization.name.toLowerCase().includes(query) ||
            organization.id.toLowerCase().includes(query),
        ),
      }),
  );
  on("POST", "/api/organizations", (_p, _url, body, actorId) => {
    const o = {
      id: nextId("org"),
      name: (body && body.name) || "Org",
      created_by: actorId,
    };
    db.organizations.push(o);
    db.members[o.id] = [{
      id: nextId("mem"),
      organization_id: o.id,
      user_id: actorId,
      status: "active",
      roles: ["owner"],
    }];
    db.orgTeams[o.id] = [];
    db.orgBalances[o.id] = 100;
    return ok(o, 201);
  });
  on(
    "GET",
    "/api/organizations/:id/credits/balance",
    (p) => ok({ amount: db.orgBalances[p.id] || 0 }),
  );
  on(
    "GET",
    "/api/organizations/:id/members",
    (p) => ok({ members: db.members[p.id] || [] }),
  );
  on("POST", "/api/organizations/:id/members", (p, _url, body) => {
    const user = findOrCreateUserByEmail(body && body.email);
    const m = {
      id: nextId("mem"),
      organization_id: p.id,
      user_id: user ? user.id : nextId("user"),
      status: "active",
      roles: (body && body.roles) || ["member"],
    };
    (db.members[p.id] = db.members[p.id] || []).push(m);
    return ok(m, 201);
  });
  on(
    "PATCH",
    "/api/organizations/:id/members/:userId/roles",
    (p, _url, body) => {
      const member = (db.members[p.id] || []).find((m) =>
        m.user_id === p.userId && m.status === "active"
      );
      if (!member) return err(404, "member not found");
      member.roles = (body && body.roles && body.roles.length > 0)
        ? body.roles
        : ["member"];
      return ok(member);
    },
  );
  on("PATCH", "/api/organizations/:id/members/:userId/deactivate", (p) => {
    const member = (db.members[p.id] || []).find((m) =>
      m.user_id === p.userId && m.status === "active"
    );
    if (!member) return err(404, "member not found");
    member.status = "inactive";
    return empty();
  });
  on(
    "GET",
    "/api/organizations/:id/teams",
    (p, url) =>
      ok({
        teams: selectorPage(
          url,
          db.orgTeams[p.id] || [],
          (team, query) =>
            team.name.toLowerCase().includes(query) ||
            team.id.toLowerCase().includes(query),
        ),
      }),
  );
  on("POST", "/api/organizations/:id/teams", (p, _url, body, actorId) => {
    const team = {
      id: nextId("team"),
      owner_kind: "organization",
      organization_id: p.id,
      owner_user_id: "",
      name: (body && body.name) || "Team",
      created_by: actorId,
    };
    (db.orgTeams[p.id] = db.orgTeams[p.id] || []).push(team);
    return ok(team, 201);
  });

  on("GET", "/api/teams", (_p, url) =>
    ok({
      teams: selectorPage(
        url,
        db.standaloneTeams,
        (team, query) =>
          team.name.toLowerCase().includes(query) ||
          team.id.toLowerCase().includes(query),
      ),
    }));
  on("POST", "/api/teams", (_p, _url, body, actorId) => {
    const team = {
      id: nextId("team"),
      owner_kind: "user",
      organization_id: "",
      owner_user_id: actorId,
      name: (body && body.name) || "Team",
      created_by: actorId,
    };
    db.standaloneTeams.push(team);
    db.teamMembers[team.id] = [];
    return ok(team, 201);
  });
  on("GET", "/api/teams/:id", (p) => {
    const team = db.standaloneTeams.concat(...Object.values(db.orgTeams)).find((
      x,
    ) => x.id === p.id);
    if (!team) return err(404, "team not found");
    return ok({ team, members: db.teamMembers[p.id] || [] });
  });
  on("GET", "/api/teams/:id/work", (p) =>
    ok({
      tasks: db.tasks
        .filter((t) =>
          t.visibility_id === p.id || t.active_assignee_id === p.id
        )
        .map((t) => listItem(t)),
    }));
  on("POST", "/api/teams/:id/members", (p, _url, body) => {
    const team = db.standaloneTeams.concat(...Object.values(db.orgTeams)).find((
      x,
    ) => x.id === p.id);
    if (!team) return err(404, "team not found");
    const user = findOrCreateUserByEmail(body && body.email);
    (db.teamMembers[p.id] = db.teamMembers[p.id] || []).push(
      user ? user.id : nextId("user"),
    );
    return ok({ team, members: db.teamMembers[p.id] }, 201);
  });

  const findSeries = (id) => db.series.find((x) => x.id === id);
  function seriesDetail(s) {
    const tasks = db.tasks
      .filter((t) => t.series_id === s.id)
      .sort((a, b) => (a.series_position || 0) - (b.series_position || 0))
      .map((t) => ({ id: t.id, title: t.title, state: t.state }));
    return { series: s, tasks, comments: db.seriesComments[s.id] || [] };
  }
  on("GET", "/api/task-series", () => ok({ series: db.series }));
  on("POST", "/api/task-series", (_p, _url, body, actorId) => {
    const s = {
      id: nextId("series"),
      owner_kind: "user",
      title: (body && body.title) || "Series",
      description: (body && body.description) || "",
      state: "draft",
      created_by: actorId,
    };
    db.series.unshift(s);
    db.seriesComments[s.id] = [];
    return ok(seriesDetail(s), 201);
  });
  on("GET", "/api/task-series/:id", (p) => {
    const s = findSeries(p.id);
    return s ? ok(seriesDetail(s)) : err(404, "series not found");
  });
  on("PATCH", "/api/task-series/:id", (p, _url, body) => {
    const s = findSeries(p.id);
    if (!s) return err(404, "series not found");
    s.title = (body && body.title) || s.title;
    s.description =
      body && Object.prototype.hasOwnProperty.call(body, "description")
        ? body.description
        : s.description;
    return ok(seriesDetail(s));
  });
  const setSeriesState = (state) => (p) => {
    const s = findSeries(p.id);
    if (!s) return err(404, "series not found");
    s.state = state;
    return ok(seriesDetail(s));
  };
  on("POST", "/api/task-series/:id/publish", setSeriesState("published"));
  on("POST", "/api/task-series/:id/unpublish", setSeriesState("draft"));
  on("POST", "/api/task-series/:id/close", setSeriesState("closed"));
  on("POST", "/api/task-series/:id/reopen", setSeriesState("draft"));
  on("POST", "/api/task-series/:id/tasks", (p, _url, body) => {
    const s = findSeries(p.id);
    if (!s) return err(404, "series not found");
    const t = findTask(body && body.task_id);
    if (!t) return err(404, "task not found");
    const max = db.tasks.filter((x) => x.series_id === s.id).reduce(
      (m, x) => Math.max(m, x.series_position || 0),
      0,
    );
    t.series_id = s.id;
    t.series_position = max + 1;
    t.series_kind = "existing_series";
    return ok(seriesDetail(s));
  });
  on("DELETE", "/api/task-series/:id/tasks/:taskId", (p) => {
    const s = findSeries(p.id);
    if (!s) return err(404, "series not found");
    const t = findTask(p.taskId);
    if (t && t.series_id === s.id) {
      t.series_id = "";
      t.series_position = 0;
      t.series_kind = "standalone";
    }
    return ok(seriesDetail(s));
  });
  on("POST", "/api/task-series/:id/reorder", (p, _url, body) => {
    const s = findSeries(p.id);
    if (!s) return err(404, "series not found");
    const ids = (body && body.task_ids) || [];
    ids.forEach((id, index) => {
      const t = findTask(id);
      if (t && t.series_id === s.id) t.series_position = index + 1;
    });
    return ok(seriesDetail(s));
  });
  on(
    "GET",
    "/api/task-series/:id/comments",
    (p) => ok({ comments: db.seriesComments[p.id] || [] }),
  );
  on("POST", "/api/task-series/:id/comments", (p, _url, body, actorId) => {
    const s = findSeries(p.id);
    if (!s) return err(404, "series not found");
    const comment = {
      id: nextId("scom"),
      series_id: s.id,
      author_user_id: actorId,
      body: (body && body.body) || "",
      created_at: "2026-06-25T12:00:00Z",
    };
    (db.seriesComments[s.id] = db.seriesComments[s.id] || []).push(comment);
    return ok(comment, 201);
  });
  on("GET", "/api/users", (_p, url) => {
    const users = selectorPage(
      url,
      db.users,
      (user, query) =>
        user.email.toLowerCase().includes(query) ||
        user.id.toLowerCase().includes(query),
    );
    return ok({ users });
  });
  on(
    "GET",
    "/api/users/:id",
    (p) =>
      ok({
        id: p.id,
        tasks: db.tasks.filter((t) => t.created_by === p.id).map(listItem),
      }),
  );
  on(
    "GET",
    "/api/users/:id/work",
    (p) =>
      ok({
        tasks: db.tasks.filter((t) => activeAssignee(t) === p.id).map(listItem),
      }),
  );
  on(
    "GET",
    "/api/users/:id/submissions",
    (p) =>
      ok({
        submissions: db.tasks.flatMap((t) => t.submissions).filter((s) =>
          s.submitter_id === p.id
        ),
      }),
  );
  on(
    "GET",
    "/api/submission-receipts/:token",
    () =>
      ok({
        submission: {
          id: "sub-receipt",
          task_id: "task-1",
          submitter_id: ME,
          state: "submitted",
          response_json: "{}",
          review_note: "",
          validation_errors: [],
        },
      }),
  );

  const base = (window.location.origin && window.location.origin !== "null")
    ? window.location.origin
    : "http://demo.local";
  function resolve(method, rawUrl, rawBody, rawHeaders) {
    let url;
    try {
      url = new URL(rawUrl, base);
    } catch (_) {
      return null;
    }
    if (!url.pathname.startsWith("/api/")) return null;
    const found = match(method, url.pathname);
    if (!found) {
      console.warn("[demo-backend] unhandled", method, url.pathname);
      return Promise.resolve(err(404, "demo route not implemented"));
    }
    const actorId = ensureActor(actorFromHeaders(rawHeaders || {}));
    if (!actorId && !allowsAnonymousRoute(method, url.pathname)) {
      return Promise.resolve(err(401, "valid bearer access token is required"));
    }
    let body = null;
    if (rawBody && typeof rawBody === "string") {
      try {
        body = JSON.parse(rawBody);
      } catch (_) {
        body = null;
      }
    }
    try {
      return Promise.resolve(
        found.handler(found.params, url, body, actorId || ME),
      );
    } catch (e) {
      console.error("[demo-backend] handler error", method, url.pathname, e);
      return Promise.resolve(
        err(500, e && e.message ? e.message : "demo backend error"),
      );
    }
  }

  window.__sharecropDemoBackend = {
    routes: routes.map((route) => ({
      method: route.method,
      pattern: route.parts.join("/"),
    })),
    resolve,
  };

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
    this._headers = {};
  }
  DemoXHR.prototype.open = function (method, url) {
    this._method = (method || "GET").toUpperCase();
    this._url = url;
    this._intercept = (function () {
      try {
        return new URL(url, base).pathname.startsWith("/api/");
      } catch (_) {
        return false;
      }
    })();
    if (!this._intercept) {
      this._real = new RealXHR();
      this._real.open.apply(this._real, arguments);
    }
  };
  DemoXHR.prototype.setRequestHeader = function (k, v) {
    if (this._real) this._real.setRequestHeader(k, v);
    else this._headers[k] = v;
  };
  DemoXHR.prototype.getAllResponseHeaders = function () {
    return this._real
      ? this._real.getAllResponseHeaders()
      : "content-type: application/json\r\n";
  };
  DemoXHR.prototype.getResponseHeader = function (name) {
    if (this._real) return this._real.getResponseHeader(name);
    return name.toLowerCase() === "content-type" ? "application/json" : null;
  };
  DemoXHR.prototype.addEventListener = function (type, fn) {
    (this._listeners[type] = this._listeners[type] || []).push(fn);
    if (this._real) this._real.addEventListener(type, fn);
  };
  DemoXHR.prototype.removeEventListener = function (type, fn) {
    if (this._real) this._real.removeEventListener(type, fn);
  };
  DemoXHR.prototype.abort = function () {
    if (this._real) this._real.abort();
  };
  DemoXHR.prototype._emit = function (type) {
    const ev = { type: type, target: this, currentTarget: this };
    if (typeof this["on" + type] === "function") this["on" + type](ev);
    (this._listeners[type] || []).forEach((fn) => {
      try {
        fn.call(this, ev);
      } catch (e) {
        console.error(e);
      }
    });
  };
  DemoXHR.prototype.send = function (body) {
    if (this._real) {
      // mirror real XHR back onto this façade for non-/api requests
      const self = this;
      ["load", "error", "abort", "timeout", "loadend", "readystatechange"]
        .forEach((t) =>
          self._real.addEventListener(t, function () {
            self.readyState = self._real.readyState;
            self.status = self._real.status;
            self.statusText = self._real.statusText;
            self.responseText = self._real.responseText;
            self.response = self._real.response;
          })
        );
      return this._real.send(body);
    }
    const self = this;
    resolve(this._method, this._url, body, this._headers).then(
      function (result) {
        const res = result || ok({});
        self.status = res.status;
        self.statusText = res.status >= 400 ? "Error" : "OK";
        self.responseText = res.body || "";
        self.response = res.body || "";
        self.readyState = 4;
        self._emit("readystatechange");
        self._emit("load");
        self._emit("loadend");
      },
    );
  };
  window.XMLHttpRequest = DemoXHR;
})();
