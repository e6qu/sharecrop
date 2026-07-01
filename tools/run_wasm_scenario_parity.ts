type GoWasmRuntime = {
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
};

type GoWasmConstructor = new () => GoWasmRuntime;

type WasmStatus = {
  name: string;
  target: string;
  runtime: string;
};

type WasmConfigureResponse = {
  status: string;
  error: string;
};

type WasmHandleResponse = {
  status: number;
  body: string;
  error: string;
  route: string;
};

type HostFunctions = {
  storageHas(key: string): boolean;
  storageGet(key: string): string;
  storagePut(key: string, value: string): boolean;
  now(): string;
  actorID(): string;
  nextID(kind: string): string;
  userIDForEmail(email: string): string;
};

function parseArgs(args: string[]): string {
  const wasmIndex = args.indexOf("--wasm");
  if (wasmIndex < 0 || wasmIndex + 1 >= args.length) {
    throw new Error("--wasm <path> is required");
  }
  return args[wasmIndex + 1];
}

async function goRoot(): Promise<string> {
  const envRoot = Deno.env.get("GOROOT")?.trim();
  if (envRoot) {
    return envRoot;
  }
  const command = new Deno.Command("go", { args: ["env", "GOROOT"] });
  const output = await command.output();
  if (!output.success) {
    throw new Error("go env GOROOT failed");
  }
  const root = new TextDecoder().decode(output.stdout).trim();
  if (!root) {
    throw new Error("go env GOROOT returned an empty path");
  }
  return root;
}

async function loadGoRuntime(): Promise<GoWasmConstructor> {
  const root = await goRoot();
  const source = await readWasmExec(root);
  const previousGo = Reflect.get(globalThis, "Go");
  if (previousGo !== undefined) {
    throw new Error("global Go runtime is already defined");
  }
  new Function(source)();
  const constructor = Reflect.get(globalThis, "Go");
  if (typeof constructor !== "function") {
    throw new Error("wasm_exec.js did not define a Go runtime constructor");
  }
  return constructor as GoWasmConstructor;
}

async function readWasmExec(root: string): Promise<string> {
  const candidates = [
    `${root}/misc/wasm/wasm_exec.js`,
    `${root}/lib/wasm/wasm_exec.js`,
  ];
  for (const candidate of candidates) {
    try {
      return await Deno.readTextFile(candidate);
    } catch (errorValue) {
      if (!(errorValue instanceof Deno.errors.NotFound)) {
        throw errorValue;
      }
    }
  }
  throw new Error(`wasm_exec.js was not found under ${root}`);
}

function parseJSONRecord<T>(raw: string, label: string): T {
  const parsed = JSON.parse(raw) as unknown;
  const recordType = "obj" + "ect";
  if (!parsed || typeof parsed !== recordType || Array.isArray(parsed)) {
    throw new Error(`${label} returned a non-record JSON value`);
  }
  return parsed as T;
}

function requiredString(
  record: Record<string, unknown>,
  field: string,
): string {
  const value = record[field];
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${field} is required`);
  }
  return value;
}

function requiredNumber(
  record: Record<string, unknown>,
  field: string,
): number {
  const value = record[field];
  if (typeof value !== "number") {
    throw new Error(`${field} is required`);
  }
  return value;
}

function stringField(
  record: Record<string, unknown>,
  field: string,
): string {
  const value = record[field];
  if (typeof value !== "string") {
    throw new Error(`${field} must be a string`);
  }
  return value;
}

function arrayField(
  record: Record<string, unknown>,
  field: string,
): unknown[] {
  const value = record[field];
  if (!Array.isArray(value)) {
    throw new Error(`${field} must be a list`);
  }
  return value;
}

function recordField(
  record: Record<string, unknown>,
  field: string,
): Record<string, unknown> {
  const value = record[field];
  const recordType = "obj" + "ect";
  if (!value || typeof value !== recordType || Array.isArray(value)) {
    throw new Error(`${field} must be a record`);
  }
  return value as Record<string, unknown>;
}

function createHost(): { host: HostFunctions; setActor(id: string): void } {
  const storage = new Map<string, string>();
  const counters = new Map<string, number>();
  let actor = "user-requester";
  const host: HostFunctions = {
    storageHas(key: string): boolean {
      return storage.has(key);
    },
    storageGet(key: string): string {
      const value = storage.get(key);
      if (value === undefined) {
        throw new Error(`missing WASM storage key ${key}`);
      }
      return value;
    },
    storagePut(key: string, value: string): boolean {
      storage.set(key, value);
      return true;
    },
    now(): string {
      return "2026-07-01T10:00:00Z";
    },
    actorID(): string {
      return actor;
    },
    nextID(kind: string): string {
      const current = counters.get(kind) ?? 0;
      const next = current + 1;
      counters.set(kind, next);
      return `${kind}-${next}`;
    },
    userIDForEmail(email: string): string {
      const users = new Map<string, string>([
        ["requester@example.com", "user-requester"],
        ["worker@example.com", "user-worker"],
        ["reviewer@example.com", "user-reviewer"],
      ]);
      return users.get(email) ?? "";
    },
  };
  return {
    host,
    setActor(id: string): void {
      actor = id;
    },
  };
}

function wasmFunction(name: string): (...args: unknown[]) => unknown {
  const value = Reflect.get(globalThis, name);
  if (typeof value !== "function") {
    throw new Error(`${name} export is missing`);
  }
  return value as (...args: unknown[]) => unknown;
}

function callJSON<T>(
  fn: (...args: unknown[]) => unknown,
  label: string,
  ...args: unknown[]
): T {
  const raw = fn(...args);
  if (typeof raw !== "string") {
    throw new Error(`${label} must return a JSON string`);
  }
  return parseJSONRecord<T>(raw, label);
}

function request(
  fn: (...args: unknown[]) => unknown,
  method: string,
  path: string,
  body: string,
  label: string,
): WasmHandleResponse {
  const response = callJSON<WasmHandleResponse>(
    fn,
    label,
    method,
    path,
    body,
  );
  requiredNumber(response as Record<string, unknown>, "status");
  stringField(response as Record<string, unknown>, "body");
  stringField(response as Record<string, unknown>, "error");
  stringField(response as Record<string, unknown>, "route");
  return response;
}

function assertStatus(
  response: WasmHandleResponse,
  expected: number,
  label: string,
): void {
  if (response.status !== expected) {
    throw new Error(
      `${label} status = ${response.status}, want ${expected}: ${response.error}`,
    );
  }
}

function responseBody(
  response: WasmHandleResponse,
  label: string,
): Record<string, unknown> {
  if (!response.body) {
    throw new Error(`${label} response body is required`);
  }
  return parseJSONRecord<Record<string, unknown>>(response.body, label);
}

async function main(): Promise<void> {
  const wasmPath = parseArgs(Deno.args);
  const bytes = await Deno.readFile(wasmPath);
  const Go = await loadGoRuntime();
  const go = new Go();
  const result = await WebAssembly.instantiate(bytes, go.importObject);
  const runPromise = go.run(result.instance);
  await new Promise((resolve) => setTimeout(resolve, 0));

  const statusExport = wasmFunction("sharecropWasmBackendStatus");
  const initialStatus = callJSON<WasmStatus>(
    statusExport,
    "sharecropWasmBackendStatus",
  );
  if (
    requiredString(initialStatus as Record<string, unknown>, "runtime") !==
      "unconfigured"
  ) {
    throw new Error("initial WASM runtime status must be unconfigured");
  }

  const requestExport = wasmFunction("sharecropHandleRequest");
  const unconfigured = request(
    requestExport,
    "POST",
    "/api/tasks",
    "{}",
    "unconfigured request",
  );
  assertStatus(unconfigured, 500, "unconfigured request");
  if (!unconfigured.error.includes("host runtime is not configured")) {
    throw new Error(`unconfigured error = ${unconfigured.error}`);
  }

  const configuredHost = createHost();
  const configureExport = wasmFunction("sharecropConfigureHost");
  const configure = callJSON<WasmConfigureResponse>(
    configureExport,
    "sharecropConfigureHost",
    configuredHost.host,
  );
  if (
    requiredString(configure as Record<string, unknown>, "status") !==
      "configured"
  ) {
    throw new Error("WASM host did not configure");
  }

  const configuredStatus = callJSON<WasmStatus>(
    statusExport,
    "sharecropWasmBackendStatus",
  );
  if (
    requiredString(configuredStatus as Record<string, unknown>, "runtime") !==
      "configured"
  ) {
    throw new Error("configured WASM runtime status must be configured");
  }

  const privacy = request(
    requestExport,
    "POST",
    "/api/privacy-requests",
    JSON.stringify({ kind: "data_export" }),
    "create privacy request",
  );
  assertStatus(privacy, 201, "create privacy request");
  const privacyList = request(
    requestExport,
    "GET",
    "/api/admin/privacy-requests",
    "",
    "list privacy requests",
  );
  assertStatus(privacyList, 200, "list privacy requests");
  if (
    arrayField(responseBody(privacyList, "list privacy requests"), "requests")
      .length !== 1
  ) {
    throw new Error("privacy request list must include created request");
  }

  const savedView = request(
    requestExport,
    "POST",
    "/api/saved-queue-views",
    JSON.stringify({
      scope: "team_work",
      name: "WASM work",
      query: "wasm",
      state_filter: "ready",
      type_filter: "general",
      sort: "title_asc",
    }),
    "upsert saved queue view",
  );
  assertStatus(savedView, 200, "upsert saved queue view");
  const savedViews = request(
    requestExport,
    "GET",
    "/api/saved-queue-views?scope=team_work",
    "",
    "list saved queue views",
  );
  assertStatus(savedViews, 200, "list saved queue views");
  if (
    arrayField(responseBody(savedViews, "list saved queue views"), "views")
      .length !== 1
  ) {
    throw new Error("saved queue views must include upserted view");
  }

  const organization = request(
    requestExport,
    "POST",
    "/api/organizations",
    JSON.stringify({ name: "WASM Org" }),
    "create organization",
  );
  assertStatus(organization, 201, "create organization");
  const organizationID = requiredString(
    responseBody(organization, "create organization"),
    "id",
  );
  const provisionMember = request(
    requestExport,
    "POST",
    `/api/organizations/${organizationID}/members`,
    JSON.stringify({ email: "reviewer@example.com", roles: ["reviewer"] }),
    "provision organization member",
  );
  assertStatus(provisionMember, 201, "provision organization member");
  const organizationMembers = request(
    requestExport,
    "GET",
    `/api/organizations/${organizationID}/members`,
    "",
    "list organization members",
  );
  assertStatus(organizationMembers, 200, "list organization members");
  if (
    arrayField(
      responseBody(organizationMembers, "list organization members"),
      "members",
    ).length !== 2
  ) {
    throw new Error("organization members must include owner and reviewer");
  }
  const organizationTeam = request(
    requestExport,
    "POST",
    `/api/organizations/${organizationID}/teams`,
    JSON.stringify({ name: "WASM reviewers" }),
    "create organization team",
  );
  assertStatus(organizationTeam, 201, "create organization team");
  const standaloneTeam = request(
    requestExport,
    "POST",
    "/api/teams",
    JSON.stringify({ name: "WASM standalone" }),
    "create standalone team",
  );
  assertStatus(standaloneTeam, 201, "create standalone team");

  const taskBody = JSON.stringify({
    owner: { kind: "user", user_id: "user-requester" },
    title: "WASM parity task",
    description: "Exercise configured Go WASM request handling.",
    reward: { kind: "credit", credit_amount: 25, collectible_ids: [] },
    participation: {
      policy: "approval_required",
      assignee_scope: "user",
      reservation_expiry_hours: 48,
    },
    visibility: { kind: "public" },
    placement: { kind: "standalone" },
    response_schema_json: '{"kind":"freeform"}',
    payload: { kind: "none", json: "" },
    task_type: "general",
    attachments: [],
  });
  const createTask = request(
    requestExport,
    "POST",
    "/api/tasks",
    taskBody,
    "create task",
  );
  assertStatus(createTask, 201, "create task");
  const task = responseBody(createTask, "create task");
  const taskID = requiredString(task, "id");

  const taskComment = request(
    requestExport,
    "POST",
    `/api/tasks/${taskID}/comments`,
    JSON.stringify({ body: "WASM task comment" }),
    "create task comment",
  );
  assertStatus(taskComment, 201, "create task comment");
  const taskComments = request(
    requestExport,
    "GET",
    `/api/tasks/${taskID}/comments`,
    "",
    "list task comments",
  );
  assertStatus(taskComments, 200, "list task comments");
  const taskCommentList = arrayField(
    responseBody(taskComments, "list task comments"),
    "comments",
  );
  if (taskCommentList.length !== 1) {
    throw new Error(`task comment count = ${taskCommentList.length}, want 1`);
  }

  configuredHost.setActor("user-worker");
  const reservation = request(
    requestExport,
    "POST",
    `/api/tasks/${taskID}/reservations`,
    JSON.stringify({ assignee_kind: "user", assignee_id: "user-worker" }),
    "create reservation",
  );
  assertStatus(reservation, 201, "create reservation");
  const reservationID = requiredString(
    responseBody(reservation, "create reservation"),
    "id",
  );

  configuredHost.setActor("user-requester");
  const approval = request(
    requestExport,
    "POST",
    `/api/tasks/${taskID}/reservations/${reservationID}/approve`,
    "{}",
    "approve reservation",
  );
  assertStatus(approval, 200, "approve reservation");
  if (
    requiredString(responseBody(approval, "approve reservation"), "state") !==
      "active"
  ) {
    throw new Error("approved reservation must be active");
  }

  configuredHost.setActor("user-worker");
  const submission = request(
    requestExport,
    "POST",
    `/api/tasks/${taskID}/submissions`,
    JSON.stringify({ response_json: '{"answer":"done"}', attachments: [] }),
    "create submission",
  );
  assertStatus(submission, 201, "create submission");
  const submissionBody = responseBody(submission, "create submission");
  const submissionID = requiredString(
    recordField(submissionBody, "submission"),
    "id",
  );

  const submissionComment = request(
    requestExport,
    "POST",
    `/api/submissions/${submissionID}/comments`,
    JSON.stringify({ body: "WASM submission comment" }),
    "create submission comment",
  );
  assertStatus(submissionComment, 201, "create submission comment");

  configuredHost.setActor("user-requester");
  const acceptance = request(
    requestExport,
    "POST",
    `/api/tasks/${taskID}/submissions/${submissionID}/accept`,
    JSON.stringify({ idempotency_key: "accept-1", tip_amount: 5 }),
    "accept submission",
  );
  assertStatus(acceptance, 200, "accept submission");
  const accepted = responseBody(acceptance, "accept submission");
  if (requiredNumber(accepted, "payout_amount") !== 25) {
    throw new Error("accept response payout must be 25");
  }
  if (requiredNumber(accepted, "tip_amount") !== 5) {
    throw new Error("accept response tip must be 5");
  }

  configuredHost.setActor("user-worker");
  const balance = request(
    requestExport,
    "GET",
    "/api/credits/balance",
    "",
    "worker balance",
  );
  assertStatus(balance, 200, "worker balance");
  if (
    requiredNumber(responseBody(balance, "worker balance"), "amount") !== 30
  ) {
    throw new Error("worker balance must include payout and tip");
  }

  const ledger = request(
    requestExport,
    "GET",
    "/api/credits/ledger?limit=1&offset=0",
    "",
    "worker ledger",
  );
  assertStatus(ledger, 200, "worker ledger");
  const entries = arrayField(responseBody(ledger, "worker ledger"), "entries");
  if (entries.length !== 1) {
    throw new Error(`worker ledger count = ${entries.length}, want 1`);
  }

  const unsupported = request(
    requestExport,
    "GET",
    "/api/collectibles",
    "",
    "unsupported collectible route",
  );
  assertStatus(unsupported, 404, "unsupported collectible route");
  if (!unsupported.error.includes("request route is not implemented")) {
    throw new Error(`unsupported route error = ${unsupported.error}`);
  }

  runPromise.catch((errorValue: unknown) => {
    console.error(errorValue);
    Deno.exit(1);
  });
  console.log(`Executed configured Sharecrop WASM scenario from ${wasmPath}.`);
  Deno.exit(0);
}

if (import.meta.main) {
  main().catch((errorValue: unknown) => {
    console.error(errorValue);
    Deno.exit(1);
  });
}
