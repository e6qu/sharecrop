interface DemoRoute {
  method: string;
  pattern: string;
}

interface DemoResponse {
  status: number;
  body: string;
}

interface DemoBackend {
  routes: DemoRoute[];
  resolve(
    method: string,
    rawUrl: string,
    rawBody: string | undefined,
    headers: Record<string, string>,
  ): Promise<DemoResponse>;
}

function assert(condition: boolean, message: string): asserts condition {
  if (!condition) {
    throw new Error(message);
  }
}

function assertEquals<T>(actual: T, expected: T, message: string): void {
  if (actual !== expected) {
    throw new Error(
      `${message}: got ${String(actual)}, want ${String(expected)}`,
    );
  }
}

async function loadDemoBackend(): Promise<DemoBackend> {
  const source = await Deno.readTextFile("site/demo/backend.js");
  function RealXMLHttpRequest() {}
  const windowObject = {
    location: { origin: "http://demo.test" },
    XMLHttpRequest: RealXMLHttpRequest,
  };
  const loader = new Function(
    "window",
    "console",
    `${source}\nreturn window.__sharecropDemoBackend;`,
  ) as (windowValue: typeof windowObject, consoleValue: Console) => DemoBackend;
  return loader(windowObject, console);
}

function normalizeRoute(route: string): string {
  return route.replaceAll(/\{[^/]+?\}|:[^/]+/g, ":param");
}

async function realHTTPRoutes(): Promise<string[]> {
  const serverSource = await Deno.readTextFile("internal/http/server.go");
  const routes = [...serverSource.matchAll(/mux\.HandleFunc\("([^"]+)"/g)]
    .map((match) => normalizeRoute(match[1]));
  return routes;
}

function demoRoutes(backend: DemoBackend): string[] {
  return backend.routes.map((route) =>
    normalizeRoute(`${route.method} ${route.pattern}`)
  );
}

async function request(
  backend: DemoBackend,
  method: string,
  path: string,
  body: Record<string, unknown> | undefined,
  accessToken: string,
): Promise<{ status: number; json: Record<string, unknown> }> {
  const response = await backend.resolve(
    method,
    path,
    body === undefined ? undefined : JSON.stringify(body),
    accessToken === "" ? {} : { Authorization: `Bearer ${accessToken}` },
  );
  const parsed = response.body === "" ? {} : JSON.parse(response.body);
  assert(isRecord(parsed), `${method} ${path} returned a non-record body`);
  return { status: response.status, json: parsed as Record<string, unknown> };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  if (value == undefined || Array.isArray(value)) {
    return false;
  }
  const kind = typeof value;
  return kind !== "string" && kind !== "number" && kind !== "boolean" &&
    kind !== "function" && kind !== "symbol" && kind !== "bigint";
}

function requireString(value: Record<string, unknown>, key: string): string {
  const found = value[key];
  assert(typeof found === "string", `${key} must be a string`);
  return found;
}

function requireNumber(value: Record<string, unknown>, key: string): number {
  const found = value[key];
  assert(typeof found === "number", `${key} must be a number`);
  return found;
}

function requireArray(value: Record<string, unknown>, key: string): unknown[] {
  const found = value[key];
  assert(Array.isArray(found), `${key} must be an array`);
  return found;
}

function requireRecord(value: unknown, path: string): Record<string, unknown> {
  assert(isRecord(value), `${path} must be a record`);
  return value as Record<string, unknown>;
}

function assertTaskListItemShape(value: unknown): void {
  const task = requireRecord(value, "task");
  assertTaskSummaryShape(task);
  [
    "active_assignee_kind",
    "active_assignee_id",
  ].forEach((key) => requireString(task, key));
}

function assertTaskSummaryShape(task: Record<string, unknown>): void {
  [
    "id",
    "owner_kind",
    "title",
    "reward_kind",
    "participation_policy",
    "assignee_scope",
    "state",
    "visibility_kind",
    "availability_kind",
    "viewer_action",
    "reviewer_action",
    "created_by",
  ].forEach((key) => requireString(task, key));
  requireNumber(task, "reward_credit_amount");
  requireNumber(task, "reward_collectible_count");
  requireNumber(task, "reservation_expiry_hours");
}

function assertTaskDetailShape(value: Record<string, unknown>): void {
  assertTaskSummaryShape(value);
  [
    "owner_id",
    "description",
    "task_type",
    "reference_url",
    "visibility_id",
    "series_kind",
    "series_id",
    "response_schema_json",
    "payload_kind",
    "payload_json",
  ].forEach((key) => requireString(value, key));
  requireNumber(value, "series_position");
}

Deno.test("backendless demo route surface tracks real API routes", async () => {
  const backend = await loadDemoBackend();
  const realOnly = new Set([
    "GET /healthz",
    "POST /mcp",
    "GET /mcp",
    "DELETE /mcp",
    "GET /",
  ]);
  const real = (await realHTTPRoutes()).filter((route) => !realOnly.has(route));
  const demo = demoRoutes(backend);
  const demoSet = new Set(demo);
  const realSet = new Set(real);

  const missing = real.filter((route) => !demoSet.has(route));
  const extra = demo.filter((route) =>
    route.startsWith("GET /api/") || route.startsWith("POST /api/") ||
    route.startsWith("PATCH /api/") || route.startsWith("DELETE /api/")
  )
    .filter((route) => !realSet.has(route));

  assertEquals(missing.join("\n"), "", "demo is missing real API routes");
  assertEquals(
    extra.join("\n"),
    "",
    "demo exposes API routes that are not in the real router",
  );
});

Deno.test("backendless demo returns client-decodable shapes for account, directory, task, and reward flows", async () => {
  const backend = await loadDemoBackend();

  const auth = await request(
    backend,
    "POST",
    "/api/auth/refresh",
    undefined,
    "",
  );
  assertEquals(auth.status, 200, "refresh status");
  requireString(auth.json, "subject_kind");
  const subjectID = requireString(auth.json, "subject_id");
  const accessToken = requireString(auth.json, "access_token");

  const verification = await request(
    backend,
    "POST",
    "/api/account/email-verification",
    {},
    accessToken,
  );
  assertEquals(verification.status, 201, "email verification request status");
  const verificationToken = requireString(verification.json, "token");
  const verified = await request(
    backend,
    "POST",
    "/api/auth/email-verification/confirm",
    { token: verificationToken },
    "",
  );
  assertEquals(verified.status, 200, "email verification confirm status");
  assertEquals(
    requireString(verified.json, "status"),
    "verified",
    "email verification confirm body",
  );

  const privacyRequest = await request(
    backend,
    "POST",
    "/api/privacy-requests",
    { kind: "sensitive_field_deletion" },
    accessToken,
  );
  assertEquals(privacyRequest.status, 201, "privacy request status");
  assertEquals(
    requireString(privacyRequest.json, "requested_by"),
    subjectID,
    "privacy request actor",
  );
  assertEquals(
    requireString(privacyRequest.json, "status"),
    "queued",
    "privacy request queued status",
  );

  const privacyAudit = await request(
    backend,
    "GET",
    "/api/admin/audit-events?action=privacy_request_created&subject_kind=privacy_request",
    undefined,
    accessToken,
  );
  assertEquals(privacyAudit.status, 200, "privacy audit status");
  const events = requireArray(privacyAudit.json, "events");
  assert(events.length > 0, "privacy request must create an audit event");

  const savedView = await request(
    backend,
    "POST",
    "/api/saved-queue-views",
    {
      scope: "team_work",
      name: "Ready work",
      query: "review",
      state_filter: "ready",
      type_filter: "code_review",
      sort: "title_asc",
    },
    accessToken,
  );
  assertEquals(savedView.status, 200, "saved queue view status");
  requireString(savedView.json, "id");
  assertEquals(
    requireString(savedView.json, "scope"),
    "team_work",
    "saved queue view scope",
  );
  assertEquals(
    requireString(savedView.json, "name"),
    "Ready work",
    "saved queue view name",
  );

  const savedViews = await request(
    backend,
    "GET",
    "/api/saved-queue-views?scope=team_work",
    undefined,
    accessToken,
  );
  assertEquals(savedViews.status, 200, "saved queue views list status");
  const views = requireArray(savedViews.json, "views");
  assertEquals(views.length, 1, "saved queue views list size");
  const firstView = requireRecord(views[0], "saved queue view");
  assertEquals(
    requireString(firstView, "type_filter"),
    "code_review",
    "saved queue view type filter",
  );

  const reset = await request(
    backend,
    "POST",
    "/api/auth/password-reset/request",
    { email: "worker@example.com" },
    "",
  );
  assertEquals(reset.status, 201, "password reset request status");
  const resetToken = requireString(reset.json, "token");
  const resetConfirmed = await request(
    backend,
    "POST",
    "/api/auth/password-reset/confirm",
    { token: resetToken, password: "changed horse battery staple" },
    "",
  );
  assertEquals(resetConfirmed.status, 200, "password reset confirm status");
  assertEquals(
    requireString(resetConfirmed.json, "status"),
    "password_reset",
    "password reset confirm body",
  );

  const directory = await request(
    backend,
    "GET",
    "/api/users?query=mara",
    undefined,
    accessToken,
  );
  const users = requireArray(directory.json, "users");
  assert(users.length > 0, "user directory should include seeded demo users");
  const user = requireRecord(users[0], "users[0]");
  requireString(user, "id");
  requireString(user, "email");
  requireString(user, "status");

  const tasks = await request(
    backend,
    "GET",
    "/api/tasks",
    undefined,
    accessToken,
  );
  const taskItems = requireArray(tasks.json, "tasks");
  assert(taskItems.length > 0, "task list should include seeded tasks");
  assertTaskListItemShape(taskItems[0]);

  const detail = await request(
    backend,
    "GET",
    "/api/tasks/task-1",
    undefined,
    accessToken,
  );
  assertTaskDetailShape(detail.json);

  const notifications = await request(
    backend,
    "GET",
    "/api/notifications",
    undefined,
    accessToken,
  );
  const notificationItems = requireArray(notifications.json, "notifications");
  assert(
    notificationItems.length > 0,
    "notifications should include seeded inbox rows",
  );
  const notification = requireRecord(notificationItems[0], "notifications[0]");
  const notificationID = requireString(notification, "id");
  requireString(notification, "recipient_user_id");
  requireString(notification, "actor_user_id");
  requireString(notification, "kind");
  requireString(notification, "subject_kind");
  requireString(notification, "subject_id");
  requireString(notification, "metadata_json");
  requireString(notification, "created_at");
  assertEquals(
    requireString(notification, "state"),
    "unread",
    "seed notification state",
  );
  const readNotification = await request(
    backend,
    "POST",
    `/api/notifications/${notificationID}/read`,
    undefined,
    accessToken,
  );
  assertEquals(readNotification.status, 200, "mark notification read status");
  assertEquals(
    requireString(readNotification.json, "state"),
    "read",
    "read notification state",
  );

  const seededSubmissionComments = await request(
    backend,
    "GET",
    "/api/submissions/sub-4-sol/comments",
    undefined,
    accessToken,
  );
  assertEquals(
    seededSubmissionComments.status,
    200,
    "seeded submission comments status",
  );
  const seededComments = requireArray(
    seededSubmissionComments.json,
    "comments",
  );
  assert(seededComments.length > 0, "seeded submission comments should load");

  const strangerAuth = await request(
    backend,
    "POST",
    "/api/auth/register",
    {
      email: "stranger@example.com",
      password: "correct horse battery staple",
    },
    "",
  );
  const strangerToken = requireString(strangerAuth.json, "access_token");
  const deniedComments = await request(
    backend,
    "GET",
    "/api/submissions/sub-4-sol/comments",
    undefined,
    strangerToken,
  );
  assertEquals(
    deniedComments.status,
    403,
    "stranger submission comments status",
  );

  const addedSubmissionComment = await request(
    backend,
    "POST",
    "/api/submissions/sub-4-sol/comments",
    { body: "Thanks, checking that now." },
    accessToken,
  );
  assertEquals(
    addedSubmissionComment.status,
    201,
    "add seeded submission comment status",
  );
  assertEquals(
    requireString(addedSubmissionComment.json, "body"),
    "Thanks, checking that now.",
    "added submission comment body",
  );

  const minted = await request(backend, "POST", "/api/collectibles", {
    name: "Demo selector medal",
    kind: "badge",
    transfer_policy: "non_transferable_except_payout",
  }, accessToken);
  assertEquals(minted.status, 201, "mint collectible status");
  const collectibleID = requireString(minted.json, "id");

  const created = await request(backend, "POST", "/api/tasks", {
    owner: { kind: "user", user_id: "user-mara" },
    title: "Demo collectible reward task",
    description: "Created by the demo contract test.",
    visibility: { kind: "user", user_id: "user-tala" },
    participation: {
      policy: "open",
      assignee_scope: "user",
      reservation_expiry_hours: 48,
    },
    reward: {
      kind: "collectible",
      credit_amount: 0,
      collectible_ids: [collectibleID],
    },
    response_schema_json: '{"kind":"freeform"}',
    payload: { kind: "none", json: "" },
  }, accessToken);
  assertEquals(created.status, 201, "create collectible task status");
  assertEquals(
    requireString(created.json, "reward_kind"),
    "collectible",
    "created task reward kind",
  );
  assertEquals(
    requireNumber(created.json, "reward_collectible_count"),
    1,
    "created task collectible count",
  );

  const missing = await request(
    backend,
    "GET",
    "/api/not-a-real-route",
    undefined,
    "",
  );
  assertEquals(missing.status, 404, "unknown demo API route status");
  assertEquals(
    requireString(missing.json, "error"),
    "demo route not implemented",
    "unknown route error",
  );
});
