import {
  type JsonRecord,
  noScenarioBody,
  runSharedScenarioParity,
  type ScenarioBody,
  type ScenarioClient,
  type ScenarioResponse,
} from "../tests/scenario_parity/scenario.ts";
import {
  arrayField,
  assertStatus,
  callJSON,
  createHost,
  instantiateWasm,
  parseJSONRecord,
  recordField,
  request,
  requiredNumber,
  requiredString,
  responseBody,
  type WasmConfigureResponse,
  wasmFunction,
  type WasmStatus,
} from "./wasm_runtime_loader.ts";

function parseArgs(args: string[]): string {
  const wasmIndex = args.indexOf("--wasm");
  if (wasmIndex < 0 || wasmIndex + 1 >= args.length) {
    throw new Error("--wasm <path> is required");
  }
  return args[wasmIndex + 1];
}

class WasmScenarioClient implements ScenarioClient {
  private readonly requestExport: (...args: unknown[]) => unknown;
  private readonly configuredHost: {
    setActor(id: string): void;
    rememberUser(email: string, userID: string): void;
  };
  private readonly accessToken: string;

  constructor(
    requestExport: (...args: unknown[]) => unknown,
    configuredHost: {
      setActor(id: string): void;
      rememberUser(email: string, userID: string): void;
    },
    accessToken: string,
  ) {
    this.requestExport = requestExport;
    this.configuredHost = configuredHost;
    this.accessToken = accessToken;
  }

  request(
    method: string,
    path: string,
    body: ScenarioBody,
  ): Promise<ScenarioResponse> {
    const actor = actorFromToken(this.accessToken);
    if (actor !== "") {
      this.configuredHost.setActor(actor);
    }
    const rawBody = body === noScenarioBody ? "" : JSON.stringify(body);
    const response = request(
      this.requestExport,
      method,
      path,
      rawBody,
      `${method} ${path}`,
    );
    const json = response.body.trim() === ""
      ? {}
      : parseJSONRecord<JsonRecord>(response.body, `${method} ${path}`);
    if (
      method === "POST" && path === "/api/auth/register" &&
      body !== noScenarioBody
    ) {
      const email = body.email;
      const subjectID = json.subject_id;
      if (typeof email === "string" && typeof subjectID === "string") {
        this.configuredHost.rememberUser(email, subjectID);
      }
    }
    if (response.status >= 400 && response.error !== "") {
      json.error = response.error;
    }
    return Promise.resolve({ status: response.status, json });
  }

  withAccessToken(accessToken: string): ScenarioClient {
    return new WasmScenarioClient(
      this.requestExport,
      this.configuredHost,
      accessToken,
    );
  }
}

function actorFromToken(accessToken: string): string {
  const prefix = "wasm-access-";
  if (accessToken.startsWith(prefix)) {
    return accessToken.slice(prefix.length);
  }
  return "";
}

async function main(): Promise<void> {
  const wasmPath = parseArgs(Deno.args);
  const bytes = await Deno.readFile(wasmPath);
  const { runPromise } = await instantiateWasm(bytes);
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
    requiredNumber(responseBody(balance, "worker balance"), "amount") !== 130
  ) {
    throw new Error(
      "worker balance must include signup grant, payout, and tip",
    );
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
    "/api/not-implemented",
    "",
    "unsupported route",
  );
  assertStatus(unsupported, 404, "unsupported route");
  if (!unsupported.error.includes("request route is not implemented")) {
    throw new Error(`unsupported route error = ${unsupported.error}`);
  }

  configuredHost.setActor("user-requester");
  await runSharedScenarioParity(
    new WasmScenarioClient(
      requestExport,
      configuredHost,
      "wasm-access-user-requester",
    ),
  );

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
