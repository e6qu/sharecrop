export type JsonRecord = Record<string, unknown>;

export const noScenarioBody = "__sharecrop_no_scenario_body__";

export type ScenarioBody = JsonRecord | typeof noScenarioBody;

export interface ScenarioResponse {
  status: number;
  json: JsonRecord;
}

export interface ScenarioClient {
  request(
    method: string,
    path: string,
    body: ScenarioBody,
  ): Promise<ScenarioResponse>;
}

export function assertScenario(
  condition: boolean,
  message: string,
): asserts condition {
  if (!condition) {
    throw new Error(message);
  }
}

export function requireString(value: JsonRecord, key: string): string {
  const found = value[key];
  assertScenario(typeof found === "string", `${key} must be a string`);
  return found;
}

export function requireNumber(value: JsonRecord, key: string): number {
  const found = value[key];
  assertScenario(typeof found === "number", `${key} must be a number`);
  return found;
}

export function requireArray(value: JsonRecord, key: string): unknown[] {
  const found = value[key];
  assertScenario(Array.isArray(found), `${key} must be an array`);
  return found;
}

export function requireRecord(value: unknown, path: string): JsonRecord {
  assertScenario(isRecord(value), `${path} must be a record`);
  return value as JsonRecord;
}

function isRecord(value: unknown): value is JsonRecord {
  if (value == undefined || Array.isArray(value)) {
    return false;
  }
  const kind = typeof value;
  return kind !== "string" && kind !== "number" && kind !== "boolean" &&
    kind !== "function" && kind !== "symbol" && kind !== "bigint";
}

function assertStatus(
  response: ScenarioResponse,
  expected: number,
  label: string,
): void {
  assertScenario(
    response.status === expected,
    `${label} status ${response.status}, want ${expected}`,
  );
}

function uniqueName(prefix: string): string {
  return `${prefix} ${Date.now()}`;
}

function assertTaskSummaryShape(task: JsonRecord): void {
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

export async function runSharedScenarioParity(
  client: ScenarioClient,
): Promise<void> {
  const auth = await client.request(
    "POST",
    "/api/auth/refresh",
    noScenarioBody,
  );
  assertStatus(auth, 200, "refresh");
  const subjectID = requireString(auth.json, "subject_id");

  const users = await client.request(
    "GET",
    "/api/users?query=user&limit=2&offset=0",
    noScenarioBody,
  );
  assertStatus(users, 200, "user selector");
  const firstUserPage = requireArray(users.json, "users");
  assertScenario(
    firstUserPage.length <= 2,
    "user selector must honor limit",
  );
  if (firstUserPage.length > 0) {
    const user = requireRecord(firstUserPage[0], "users[0]");
    requireString(user, "id");
    requireString(user, "email");
    requireString(user, "status");
  }

  const organizationName = uniqueName("Scenario parity org");
  const createdOrganization = await client.request(
    "POST",
    "/api/organizations",
    {
      name: organizationName,
    },
  );
  assertStatus(createdOrganization, 201, "create organization");
  const organizationID = requireString(createdOrganization.json, "id");
  assertScenario(
    requireString(createdOrganization.json, "name") === organizationName,
    "created organization name must round trip",
  );

  const organizations = await client.request(
    "GET",
    `/api/organizations?query=${
      encodeURIComponent(organizationName)
    }&limit=1&offset=0`,
    noScenarioBody,
  );
  assertStatus(organizations, 200, "organization selector");
  const organizationPage = requireArray(organizations.json, "organizations");
  assertScenario(
    organizationPage.length === 1,
    "organization selector must return the created organization",
  );
  assertScenario(
    requireString(
      requireRecord(organizationPage[0], "organizations[0]"),
      "id",
    ) ===
      organizationID,
    "organization selector returned a different organization",
  );

  const teamName = uniqueName("Scenario parity org team");
  const createdOrgTeam = await client.request(
    "POST",
    `/api/organizations/${organizationID}/teams`,
    { name: teamName },
  );
  assertStatus(createdOrgTeam, 201, "create organization team");
  const orgTeamID = requireString(createdOrgTeam.json, "id");

  const orgTeams = await client.request(
    "GET",
    `/api/organizations/${organizationID}/teams?query=${
      encodeURIComponent(teamName)
    }&limit=1&offset=0`,
    noScenarioBody,
  );
  assertStatus(orgTeams, 200, "organization team selector");
  const orgTeamPage = requireArray(orgTeams.json, "teams");
  assertScenario(
    orgTeamPage.length === 1 &&
      requireString(requireRecord(orgTeamPage[0], "teams[0]"), "id") ===
        orgTeamID,
    "organization team selector must return the created team",
  );

  const standaloneName = uniqueName("Scenario parity standalone team");
  const createdStandaloneTeam = await client.request("POST", "/api/teams", {
    name: standaloneName,
  });
  assertStatus(createdStandaloneTeam, 201, "create standalone team");
  const standaloneTeamID = requireString(createdStandaloneTeam.json, "id");

  const standaloneTeams = await client.request(
    "GET",
    `/api/teams?query=${encodeURIComponent(standaloneName)}&limit=1&offset=0`,
    noScenarioBody,
  );
  assertStatus(standaloneTeams, 200, "standalone team selector");
  const standaloneTeamPage = requireArray(standaloneTeams.json, "teams");
  assertScenario(
    standaloneTeamPage.length === 1 &&
      requireString(requireRecord(standaloneTeamPage[0], "teams[0]"), "id") ===
        standaloneTeamID,
    "standalone team selector must return the created team",
  );

  const taskTitle = uniqueName("Scenario parity task");
  const createdTask = await client.request("POST", "/api/tasks", {
    owner: { kind: "user", user_id: subjectID },
    title: taskTitle,
    description: "Created by the shared scenario parity suite.",
    visibility: { kind: "organization", organization_id: organizationID },
    participation: {
      policy: "reservation_required",
      assignee_scope: "organization_team",
      organization_team_id: orgTeamID,
      reservation_expiry_hours: 48,
    },
    reward: {
      kind: "none",
      credit_amount: 0,
      collectible_ids: [],
    },
    response_schema_json: '{"kind":"freeform"}',
    payload: { kind: "none", json: "" },
  });
  assertStatus(createdTask, 201, "create task");
  const taskID = requireString(createdTask.json, "id");
  assertTaskSummaryShape(createdTask.json);

  const taskDetail = await client.request(
    "GET",
    `/api/tasks/${taskID}`,
    noScenarioBody,
  );
  assertStatus(taskDetail, 200, "task detail");
  assertTaskSummaryShape(taskDetail.json);
  assertScenario(
    requireString(taskDetail.json, "title") === taskTitle,
    "task detail title must round trip",
  );

  const commentBody = uniqueName("Scenario parity comment");
  const createdComment = await client.request(
    "POST",
    `/api/tasks/${taskID}/comments`,
    { body: commentBody },
  );
  assertStatus(createdComment, 201, "create task comment");
  assertScenario(
    requireString(createdComment.json, "body") === commentBody,
    "task comment body must round trip",
  );

  const comments = await client.request(
    "GET",
    `/api/tasks/${taskID}/comments`,
    noScenarioBody,
  );
  assertStatus(comments, 200, "task comments");
  const taskComments = requireArray(comments.json, "comments");
  assertScenario(
    taskComments.some((comment) =>
      requireString(requireRecord(comment, "comments[]"), "body") ===
        commentBody
    ),
    "task comments must include created comment",
  );
}
