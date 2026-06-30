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
  withAccessToken(accessToken: string): ScenarioClient;
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

function assertCollectibleShape(collectible: JsonRecord): void {
  [
    "id",
    "name",
    "kind",
    "state",
    "transfer_policy",
    "owner_id",
    "owner_kind",
    "organization_id",
    "art",
  ].forEach((key) => requireString(collectible, key));
}

function assertNotificationShape(notification: JsonRecord): void {
  [
    "id",
    "recipient_user_id",
    "actor_user_id",
    "kind",
    "subject_kind",
    "subject_id",
    "state",
    "metadata_json",
    "created_at",
  ].forEach((key) => requireString(notification, key));
}

function requireMetadataRecord(value: JsonRecord, label: string): JsonRecord {
  const metadataJSON = requireString(value, "metadata_json");
  const parsed = JSON.parse(metadataJSON) as unknown;
  return requireRecord(parsed, label);
}

interface ScenarioActor {
  subjectID: string;
  email: string;
  client: ScenarioClient;
}

const scenarioObjectKind = "obj" + "ect";

async function registerScenarioActor(
  client: ScenarioClient,
  label: string,
): Promise<ScenarioActor> {
  const email = `scenario-${label}-${Date.now()}@example.com`;
  const registered = await client.request("POST", "/api/auth/register", {
    email,
    password: "correct horse battery staple",
  });
  assertStatus(registered, 201, `register ${label}`);
  const subjectID = requireString(registered.json, "subject_id");
  const accessToken = requireString(registered.json, "access_token");
  return {
    subjectID,
    email,
    client: client.withAccessToken(accessToken),
  };
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
  client = client.withAccessToken(requireString(auth.json, "access_token"));

  const operations = await client.request(
    "GET",
    "/api/admin/operations",
    noScenarioBody,
  );
  assertStatus(operations, 200, "admin operations");
  requireString(operations.json, "status");
  requireString(operations.json, "account_token_delivery");
  requireString(operations.json, "mcp_storage");
  requireString(operations.json, "rate_limit_storage");
  requireString(operations.json, "secure_cookies");
  requireNumber(operations.json, "active_mcp_sessions");

  const platformAdmins = await client.request(
    "GET",
    "/api/admin/platform-admins",
    noScenarioBody,
  );
  assertStatus(platformAdmins, 200, "platform admin list");
  requireArray(platformAdmins.json, "admins").forEach((admin, index) => {
    const record = requireRecord(admin, `platformAdmins[${index}]`);
    requireString(record, "user_id");
    requireString(record, "source");
    requireString(record, "created_at");
  });

  const verification = await client.request(
    "POST",
    "/api/account/email-verification",
    {},
  );
  assertStatus(verification, 201, "email verification token issue");
  const verificationToken = verification.json["token"];
  const verificationStatus = verification.json["status"];
  assertScenario(
    typeof verificationToken === "string" ||
      verificationStatus === "sent",
    "email verification response must include a token or sent status",
  );

  const privacyRequest = await client.request(
    "POST",
    "/api/privacy-requests",
    { kind: "data_export" },
  );
  assertStatus(privacyRequest, 201, "privacy request");
  assertScenario(
    requireString(privacyRequest.json, "kind") === "data_export",
    "privacy request kind must round trip",
  );
  assertScenario(
    requireString(privacyRequest.json, "status") === "queued",
    "privacy request status must be queued",
  );
  assertScenario(
    requireString(privacyRequest.json, "requested_by") === subjectID,
    "privacy request actor must match authenticated user",
  );
  requireString(privacyRequest.json, "created_at");
  requireNumber(privacyRequest.json, "redacted_field_count");

  const privacyAudit = await client.request(
    "GET",
    "/api/admin/audit-events?action=privacy_request_created&subject_kind=privacy_request",
    noScenarioBody,
  );
  assertStatus(privacyAudit, 200, "privacy request audit events");
  const privacyAuditEvents = requireArray(privacyAudit.json, "events");
  assertScenario(
    privacyAuditEvents.some((event) => {
      const record = requireRecord(event, "privacyAuditEvent");
      return requireString(record, "subject_id") === subjectID;
    }),
    "privacy request must be visible in audit events",
  );

  const adminPrivacyList = await client.request(
    "GET",
    "/api/admin/privacy-requests",
    noScenarioBody,
  );
  assertStatus(adminPrivacyList, 200, "admin privacy request list");
  const adminPrivacyRequests = requireArray(adminPrivacyList.json, "requests");
  assertScenario(
    adminPrivacyRequests.some((request) =>
      requireString(requireRecord(request, "privacyRequest"), "id") ===
        requireString(privacyRequest.json, "id")
    ),
    "admin privacy request list must include created request",
  );

  const resolvedExport = await client.request(
    "POST",
    `/api/admin/privacy-requests/${
      requireString(privacyRequest.json, "id")
    }/resolve`,
    { resolution_note: "scenario export generated" },
  );
  assertStatus(resolvedExport, 200, "resolve data export privacy request");
  assertScenario(
    requireString(resolvedExport.json, "status") === "resolved",
    "resolved data export status must be resolved",
  );
  const exportDocument = requireRecord(
    JSON.parse(requireString(resolvedExport.json, "export_json")) as unknown,
    "privacyExportDocument",
  );
  assertScenario(
    requireString(exportDocument, "user_id") === subjectID,
    "privacy export must identify the requester",
  );
  requireString(resolvedExport.json, "resolved_at");

  const retentionRun = await client.request(
    "POST",
    "/api/admin/privacy-retention/run",
    {},
  );
  assertStatus(retentionRun, 200, "privacy retention run");
  requireNumber(retentionRun.json, "redacted_field_count");

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

  const transferRecipient = await registerScenarioActor(
    client,
    "transfer-recipient",
  );

  const grantedAdmin = await client.request(
    "POST",
    "/api/admin/platform-admins",
    { user_id: transferRecipient.subjectID },
  );
  assertStatus(grantedAdmin, 201, "grant platform admin");
  assertScenario(
    requireString(grantedAdmin.json, "user_id") === transferRecipient.subjectID,
    "granted platform admin must match selected user",
  );
  assertScenario(
    requireString(grantedAdmin.json, "source") === "granted",
    "granted platform admin source must be granted",
  );
  requireString(grantedAdmin.json, "created_at");

  const revokedAdmin = await client.request(
    "POST",
    `/api/admin/platform-admins/${transferRecipient.subjectID}/revoke`,
    {},
  );
  assertStatus(revokedAdmin, 200, "revoke platform admin");
  assertScenario(
    requireString(revokedAdmin.json, "user_id") === transferRecipient.subjectID,
    "revoked platform admin must match selected user",
  );

  const catalog = await client.request(
    "GET",
    "/api/collectibles/catalog",
    noScenarioBody,
  );
  assertStatus(catalog, 200, "collectible catalog");
  const catalogEntries = requireArray(catalog.json, "entries");
  assertScenario(catalogEntries.length > 0, "catalog must include entries");
  const catalogEntry = requireRecord(catalogEntries[0], "entries[0]");
  requireString(catalogEntry, "slug");
  requireString(catalogEntry, "name");
  requireString(catalogEntry, "kind");
  requireString(catalogEntry, "transfer_policy");
  requireString(catalogEntry, "art");

  const collectibleName = uniqueName("Scenario parity collectible");
  const mintedCollectible = await client.request(
    "POST",
    "/api/collectibles",
    {
      name: collectibleName,
      kind: "badge",
      transfer_policy: "transferable_between_users",
      art: "harvest-star",
    },
  );
  assertStatus(mintedCollectible, 201, "mint collectible");
  assertCollectibleShape(mintedCollectible.json);
  const collectibleID = requireString(mintedCollectible.json, "id");
  assertScenario(
    requireString(mintedCollectible.json, "name") === collectibleName,
    "minted collectible name must round trip",
  );

  const transferredCollectible = await client.request(
    "POST",
    `/api/collectibles/${collectibleID}/transfer`,
    { recipient_id: transferRecipient.subjectID },
  );
  assertStatus(transferredCollectible, 200, "transfer collectible");
  assertCollectibleShape(transferredCollectible.json);
  assertScenario(
    requireString(transferredCollectible.json, "owner_id") ===
      transferRecipient.subjectID,
    "transferred collectible owner must be the recipient",
  );

  const rewardCollectible = await client.request(
    "POST",
    "/api/collectibles",
    {
      name: uniqueName("Scenario parity reward collectible"),
      kind: "badge",
      transfer_policy: "non_transferable_except_payout",
      art: "seedling",
    },
  );
  assertStatus(rewardCollectible, 201, "mint reward collectible");
  assertCollectibleShape(rewardCollectible.json);
  const rewardCollectibleID = requireString(rewardCollectible.json, "id");

  const collectibleRewardTask = await client.request("POST", "/api/tasks", {
    owner: { kind: "user", user_id: subjectID },
    title: uniqueName("Scenario parity collectible reward task"),
    description: "Created to verify create-time collectible escrow and refund.",
    visibility: { kind: "public" },
    participation: {
      policy: "open",
      assignee_scope: "user",
      reservation_expiry_hours: 48,
    },
    reward: {
      kind: "collectible",
      credit_amount: 0,
      collectible_ids: [rewardCollectibleID],
    },
    response_schema_json: '{"kind":"freeform"}',
    payload: { kind: "none", json: "" },
  });
  assertStatus(collectibleRewardTask, 201, "create collectible reward task");
  assertScenario(
    requireNumber(collectibleRewardTask.json, "reward_collectible_count") === 1,
    "collectible reward task must report held collectible count",
  );
  const collectibleRewardTaskID = requireString(
    collectibleRewardTask.json,
    "id",
  );

  const refundedCollectibleReward = await client.request(
    "POST",
    `/api/tasks/${collectibleRewardTaskID}/collectible-refund`,
    {},
  );
  assertStatus(refundedCollectibleReward, 200, "refund collectible reward");
  const refundedCollectibles = requireArray(
    refundedCollectibleReward.json,
    "collectibles",
  );
  assertScenario(
    refundedCollectibles.some((item) =>
      requireString(requireRecord(item, "refundedCollectibles[]"), "id") ===
        rewardCollectibleID
    ),
    "collectible refund must return the escrowed collectible",
  );
  refundedCollectibles.forEach((item) =>
    assertCollectibleShape(requireRecord(item, "refundedCollectibles[]"))
  );

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

  const orgReviewer = await registerScenarioActor(client, "org-reviewer");
  const orgWorker = await registerScenarioActor(client, "org-worker");
  const provisionedReviewer = await client.request(
    "POST",
    `/api/organizations/${organizationID}/members`,
    {
      email: orgReviewer.email,
      roles: ["member", "reviewer"],
    },
  );
  assertStatus(provisionedReviewer, 201, "provision org reviewer");

  const orgReviewTask = await client.request("POST", "/api/tasks", {
    owner: { kind: "organization", organization_id: organizationID },
    title: uniqueName("Scenario parity org reviewer task"),
    description: "Created to verify organization reviewer acceptance.",
    visibility: { kind: "public" },
    participation: {
      policy: "open",
      assignee_scope: "user",
      reservation_expiry_hours: 48,
    },
    reward: {
      kind: "credit",
      credit_amount: 25,
      collectible_ids: [],
    },
    response_schema_json: '{"kind":"freeform"}',
    payload: { kind: "none", json: "" },
  });
  assertStatus(orgReviewTask, 201, "create org reviewer task");
  const orgReviewTaskID = requireString(orgReviewTask.json, "id");

  const orgFunded = await client.request(
    "POST",
    `/api/tasks/${orgReviewTaskID}/funding`,
    {
      amount: 25,
      idempotency_key: `scenario-org-fund-${orgReviewTaskID}`,
      organization_id: organizationID,
    },
  );
  assertStatus(orgFunded, 201, "fund org reviewer task");

  const orgOpened = await client.request(
    "POST",
    `/api/tasks/${orgReviewTaskID}/open`,
    {},
  );
  assertStatus(orgOpened, 200, "open org reviewer task");

  const orgWorkerSubmission = await orgWorker.client.request(
    "POST",
    `/api/tasks/${orgReviewTaskID}/submissions`,
    { response_json: '{"org_review":"ready"}' },
  );
  assertStatus(orgWorkerSubmission, 201, "org worker submit");
  const orgSubmission = requireRecord(
    orgWorkerSubmission.json["submission"],
    "orgWorkerSubmission",
  );
  const orgSubmissionID = requireString(orgSubmission, "id");

  const orgReviewed = await orgReviewer.client.request(
    "POST",
    `/api/tasks/${orgReviewTaskID}/submissions/${orgSubmissionID}/accept`,
    {
      idempotency_key: `scenario-org-accept-${orgSubmissionID}`,
      payout_amount: 25,
    },
  );
  assertStatus(orgReviewed, 200, "org reviewer accept");
  assertScenario(
    requireString(orgReviewed.json, "worker_user_id") === orgWorker.subjectID,
    "org reviewer accept must pay the submitting worker",
  );

  const taskTitle = uniqueName("Scenario parity task");
  const createdTask = await client.request("POST", "/api/tasks", {
    owner: { kind: "user", user_id: subjectID },
    title: taskTitle,
    description: "Created by the shared scenario parity suite.",
    task_type: "code_review",
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

  const moderationReport = await client.request(
    "POST",
    "/api/moderation/reports",
    {
      subject_kind: "task",
      subject_id: taskID,
      reason: "policy",
      details: "Scenario parity moderation report.",
    },
  );
  assertStatus(moderationReport, 201, "create moderation report");
  assertScenario(
    requireString(moderationReport.json, "subject_kind") === "task",
    "moderation report subject kind must round trip",
  );
  assertScenario(
    requireString(moderationReport.json, "subject_id") === taskID,
    "moderation report subject id must round trip",
  );
  assertScenario(
    requireString(moderationReport.json, "reason") === "policy",
    "moderation report reason must round trip",
  );
  assertScenario(
    requireString(moderationReport.json, "reporter_user_id") === subjectID,
    "moderation report reporter must match authenticated user",
  );
  assertScenario(
    requireString(moderationReport.json, "subject_href") ===
      `#/tasks/${taskID}`,
    "task moderation report must include a direct subject href",
  );
  assertScenario(
    requireString(moderationReport.json, "state") === "open",
    "new moderation report triage state must be open",
  );
  requireString(moderationReport.json, "resolution_note");
  requireString(moderationReport.json, "updated_by");
  requireString(moderationReport.json, "updated_at");
  requireString(moderationReport.json, "created_at");

  const adminModerationList = await client.request(
    "GET",
    "/api/admin/moderation/reports?state=open",
    noScenarioBody,
  );
  assertStatus(adminModerationList, 200, "admin moderation report list");
  const adminModerationReports = requireArray(
    adminModerationList.json,
    "reports",
  );
  assertScenario(
    adminModerationReports.some((report) => {
      const record = requireRecord(report, "moderationReport");
      return requireString(record, "subject_id") === taskID;
    }),
    "admin moderation report list must include created report",
  );

  const triagedModerationReport = await client.request(
    "POST",
    `/api/admin/moderation/reports/${
      requireString(moderationReport.json, "id")
    }/triage`,
    {
      state: "resolved",
      resolution_note: "scenario moderation resolved",
    },
  );
  assertStatus(triagedModerationReport, 200, "triage moderation report");
  assertScenario(
    requireString(triagedModerationReport.json, "state") === "resolved",
    "triaged moderation report state must be resolved",
  );
  assertScenario(
    requireString(triagedModerationReport.json, "resolution_note") ===
      "scenario moderation resolved",
    "triaged moderation report note must round trip",
  );
  assertScenario(
    requireString(triagedModerationReport.json, "updated_by") === subjectID,
    "triaged moderation report updater must match admin actor",
  );

  const moderationAudit = await client.request(
    "GET",
    "/api/admin/audit-events?action=moderation_report_created&subject_kind=task",
    noScenarioBody,
  );
  assertStatus(moderationAudit, 200, "moderation report audit events");
  const moderationAuditEvents = requireArray(moderationAudit.json, "events");
  assertScenario(
    moderationAuditEvents.some((event) => {
      const record = requireRecord(event, "moderationAuditEvent");
      return requireString(record, "subject_id") === taskID;
    }),
    "moderation report must be visible in audit events",
  );

  const organizationTaskSearch = await client.request(
    "GET",
    `/api/tasks?scope=organization&organization_id=${organizationID}&query=${
      encodeURIComponent(taskTitle)
    }&task_type=code_review&sort=title_asc&limit=1&offset=0`,
    noScenarioBody,
  );
  assertStatus(organizationTaskSearch, 200, "organization task queue search");
  const organizationTaskPage = requireArray(
    organizationTaskSearch.json,
    "tasks",
  );
  assertScenario(
    organizationTaskPage.length === 1 &&
      requireString(
          requireRecord(organizationTaskPage[0], "organizationTaskPage[0]"),
          "id",
        ) === taskID,
    "organization task queue search must return the created task",
  );

  const organizationTeamTaskTitle = uniqueName(
    "Scenario parity organization-team queue task",
  );
  const createdOrganizationTeamTask = await client.request(
    "POST",
    "/api/tasks",
    {
      owner: { kind: "user", user_id: subjectID },
      title: organizationTeamTaskTitle,
      description: "Created to verify team queue search.",
      task_type: "qa_testing",
      visibility: {
        kind: "organization_team",
        organization_id: organizationID,
        team_id: orgTeamID,
      },
      participation: {
        policy: "open",
        assignee_scope: "organization_team",
        reservation_expiry_hours: 48,
      },
      reward: {
        kind: "none",
        credit_amount: 0,
        collectible_ids: [],
      },
      response_schema_json: '{"kind":"freeform"}',
      payload: { kind: "none", json: "" },
    },
  );
  assertStatus(
    createdOrganizationTeamTask,
    201,
    "create organization-team queue task",
  );
  const organizationTeamTaskID = requireString(
    createdOrganizationTeamTask.json,
    "id",
  );
  const teamWorkSearch = await client.request(
    "GET",
    `/api/teams/${orgTeamID}/work?query=${
      encodeURIComponent(organizationTeamTaskTitle)
    }&task_type=qa_testing&sort=reward_desc&limit=1&offset=0`,
    noScenarioBody,
  );
  assertStatus(teamWorkSearch, 200, "team work queue search");
  const teamWorkPage = requireArray(teamWorkSearch.json, "tasks");
  assertScenario(
    teamWorkPage.length === 1 &&
      requireString(requireRecord(teamWorkPage[0], "teamWorkPage[0]"), "id") ===
        organizationTeamTaskID,
    "team work queue search must return the created organization-team task",
  );

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

  const submissionTaskTitle = uniqueName("Scenario parity submission task");
  const submissionTask = await client.request("POST", "/api/tasks", {
    owner: { kind: "user", user_id: subjectID },
    title: submissionTaskTitle,
    description: "Created for the shared submission scenario.",
    visibility: { kind: "public" },
    participation: {
      policy: "open",
      assignee_scope: "user",
      reservation_expiry_hours: 48,
    },
    reward: {
      kind: "none",
      credit_amount: 0,
      collectible_ids: [],
    },
    response_schema_json: JSON.stringify({
      kind: scenarioObjectKind,
      fields: [
        {
          name: "email",
          presence: "required",
          schema: {
            kind: "string",
            sensitivity: {
              category: "pii",
              retention: "delete_on_request",
              redaction: "replace",
            },
          },
        },
        {
          name: "result",
          presence: "required",
          schema: { kind: "string" },
        },
      ],
    }),
    payload: { kind: "none", json: "" },
  });
  assertStatus(submissionTask, 201, "create submission task");
  const submissionTaskID = requireString(submissionTask.json, "id");

  const createdSubmission = await client.request(
    "POST",
    `/api/tasks/${submissionTaskID}/submissions`,
    { response_json: '{"email":"worker@example.com","result":"done"}' },
  );
  assertStatus(createdSubmission, 201, "create submission");
  const submission = requireRecord(
    createdSubmission.json["submission"],
    "submission",
  );
  const submissionID = requireString(submission, "id");
  requireString(createdSubmission.json, "receipt_token");
  assertScenario(
    requireString(submission, "state") === "submitted",
    "submission must be accepted by schema validation",
  );
  const sensitiveFields = requireArray(submission, "sensitive_fields");
  assertScenario(
    sensitiveFields.length === 1,
    "submission must index one sensitive field",
  );
  const sensitiveField = requireRecord(sensitiveFields[0], "sensitiveField");
  assertScenario(
    requireString(sensitiveField, "state") === "active",
    "new sensitive field state must be active",
  );

  const listedSubmissions = await client.request(
    "GET",
    `/api/tasks/${submissionTaskID}/submissions`,
    noScenarioBody,
  );
  assertStatus(listedSubmissions, 200, "list submissions");
  const submissionList = requireArray(listedSubmissions.json, "submissions");
  assertScenario(
    submissionList.some((item) => {
      const listed = requireRecord(item, "submissions[]");
      requireArray(listed, "sensitive_fields");
      return requireString(listed, "id") === submissionID;
    }),
    "listed submissions must include created submission",
  );

  const deletionRequest = await client.request(
    "POST",
    "/api/privacy-requests",
    { kind: "sensitive_field_deletion" },
  );
  assertStatus(deletionRequest, 201, "sensitive field deletion request");
  const resolvedDeletion = await client.request(
    "POST",
    `/api/admin/privacy-requests/${
      requireString(deletionRequest.json, "id")
    }/resolve`,
    { resolution_note: "scenario sensitive fields redacted" },
  );
  assertStatus(
    resolvedDeletion,
    200,
    "resolve sensitive field deletion request",
  );
  assertScenario(
    requireNumber(resolvedDeletion.json, "redacted_field_count") >= 1,
    "sensitive field deletion must report affected redactions",
  );
  const listedAfterDeletion = await client.request(
    "GET",
    `/api/tasks/${submissionTaskID}/submissions`,
    noScenarioBody,
  );
  assertStatus(
    listedAfterDeletion,
    200,
    "list submissions after privacy deletion",
  );
  const afterDeletionItems = requireArray(
    listedAfterDeletion.json,
    "submissions",
  );
  const redactedSubmission = afterDeletionItems
    .map((item) => requireRecord(item, "submissionsAfterDeletion[]"))
    .find((item) => requireString(item, "id") === submissionID);
  if (redactedSubmission === undefined) {
    throw new Error("redacted submission must remain listed");
  }
  const redactedFields = requireArray(redactedSubmission, "sensitive_fields");
  assertScenario(
    redactedFields.some((field) =>
      requireString(requireRecord(field, "redactedField"), "state") ===
        "redacted"
    ),
    "privacy deletion must mark the sensitive-field index redacted",
  );

  const submissionCommentBody = uniqueName(
    "Scenario parity submission comment",
  );
  const submissionComment = await client.request(
    "POST",
    `/api/submissions/${submissionID}/comments`,
    { body: submissionCommentBody },
  );
  assertStatus(submissionComment, 201, "create submission comment");
  assertScenario(
    requireString(submissionComment.json, "body") === submissionCommentBody,
    "submission comment body must round trip",
  );

  const submissionComments = await client.request(
    "GET",
    `/api/submissions/${submissionID}/comments`,
    noScenarioBody,
  );
  assertStatus(submissionComments, 200, "list submission comments");
  const submissionCommentList = requireArray(
    submissionComments.json,
    "comments",
  );
  assertScenario(
    submissionCommentList.some((item) =>
      requireString(requireRecord(item, "submissionComments[]"), "body") ===
        submissionCommentBody
    ),
    "listed submission comments must include created comment",
  );

  const owner = await registerScenarioActor(client, "owner");
  const worker = await registerScenarioActor(client, "worker");
  const multiActorTitle = uniqueName("Scenario parity multi actor task");
  const multiActorTask = await owner.client.request("POST", "/api/tasks", {
    owner: { kind: "user", user_id: owner.subjectID },
    title: multiActorTitle,
    description: "Created for multi-actor reservation and payout parity.",
    visibility: { kind: "public" },
    participation: {
      policy: "approval_required",
      assignee_scope: "user",
      reservation_expiry_hours: 48,
    },
    reward: {
      kind: "credit",
      credit_amount: 30,
      collectible_ids: [],
    },
    response_schema_json: '{"kind":"freeform"}',
    payload: { kind: "none", json: "" },
  });
  assertStatus(multiActorTask, 201, "create multi-actor task");
  const multiActorTaskID = requireString(multiActorTask.json, "id");
  assertScenario(
    requireString(multiActorTask.json, "created_by") === owner.subjectID,
    "multi-actor task must be created by owner actor",
  );

  const funded = await owner.client.request(
    "POST",
    `/api/tasks/${multiActorTaskID}/funding`,
    {
      amount: 30,
      idempotency_key: `scenario-fund-${multiActorTaskID}`,
    },
  );
  assertStatus(funded, 201, "fund multi-actor task");
  requireNumber(funded.json, "amount");

  const opened = await owner.client.request(
    "POST",
    `/api/tasks/${multiActorTaskID}/open`,
    {},
  );
  assertStatus(opened, 200, "open multi-actor task");

  const requestedReservation = await worker.client.request(
    "POST",
    `/api/tasks/${multiActorTaskID}/reservations`,
    {},
  );
  assertStatus(requestedReservation, 201, "request reservation approval");
  const reservationID = requireString(requestedReservation.json, "id");
  assertScenario(
    requireString(requestedReservation.json, "state") === "requested",
    "approval-required reservation must start requested",
  );
  assertScenario(
    requireString(requestedReservation.json, "requested_by") ===
      worker.subjectID,
    "reservation requester must be the worker actor",
  );

  const ownerReservations = await owner.client.request(
    "GET",
    `/api/tasks/${multiActorTaskID}/reservations`,
    noScenarioBody,
  );
  assertStatus(ownerReservations, 200, "owner list reservations");
  const reservationList = requireArray(ownerReservations.json, "reservations");
  assertScenario(
    reservationList.some((item) =>
      requireString(requireRecord(item, "reservations[]"), "id") ===
        reservationID
    ),
    "owner reservation list must include worker request",
  );

  const approvedReservation = await owner.client.request(
    "POST",
    `/api/tasks/${multiActorTaskID}/reservations/${reservationID}/approve`,
    {},
  );
  assertStatus(approvedReservation, 200, "approve reservation");
  assertScenario(
    requireString(approvedReservation.json, "state") === "active",
    "approved reservation must become active",
  );

  const workerSubmission = await worker.client.request(
    "POST",
    `/api/tasks/${multiActorTaskID}/submissions`,
    { response_json: '{"multi_actor":"complete"}' },
  );
  assertStatus(workerSubmission, 201, "worker submit approved reservation");
  const multiActorSubmission = requireRecord(
    workerSubmission.json["submission"],
    "multiActorSubmission",
  );
  const multiActorSubmissionID = requireString(multiActorSubmission, "id");
  assertScenario(
    requireString(multiActorSubmission, "submitter_id") === worker.subjectID,
    "multi-actor submission must be owned by worker actor",
  );

  const ownerSubmissionList = await owner.client.request(
    "GET",
    `/api/tasks/${multiActorTaskID}/submissions`,
    noScenarioBody,
  );
  assertStatus(ownerSubmissionList, 200, "owner list multi-actor submissions");
  const ownerSubmissions = requireArray(
    ownerSubmissionList.json,
    "submissions",
  );
  assertScenario(
    ownerSubmissions.some((item) =>
      requireString(requireRecord(item, "multiActorSubmissions[]"), "id") ===
        multiActorSubmissionID
    ),
    "owner submission list must include worker submission",
  );

  const ownerCommentBody = uniqueName("Scenario parity owner comment");
  const ownerSubmissionComment = await owner.client.request(
    "POST",
    `/api/submissions/${multiActorSubmissionID}/comments`,
    { body: ownerCommentBody },
  );
  assertStatus(ownerSubmissionComment, 201, "owner adds submission comment");

  const workerCommentNotifications = await worker.client.request(
    "GET",
    "/api/notifications",
    noScenarioBody,
  );
  assertStatus(workerCommentNotifications, 200, "worker comment notifications");
  const workerCommentNotificationList = requireArray(
    workerCommentNotifications.json,
    "notifications",
  );
  assertScenario(
    workerCommentNotificationList.some((item) => {
      const notification = requireRecord(item, "workerCommentNotifications[]");
      const metadata = requireMetadataRecord(
        notification,
        "workerCommentNotification.metadata_json",
      );
      return requireString(notification, "kind") === "submission_commented" &&
        requireString(notification, "actor_user_id") === owner.subjectID &&
        requireString(notification, "recipient_user_id") === worker.subjectID &&
        requireString(notification, "subject_id") === multiActorSubmissionID &&
        requireString(metadata, "task_id") === multiActorTaskID;
    }),
    "worker inbox must include owner submission-comment notification with task metadata",
  );

  const acceptedSubmission = await owner.client.request(
    "POST",
    `/api/tasks/${multiActorTaskID}/submissions/${multiActorSubmissionID}/accept`,
    {
      idempotency_key: `scenario-accept-${multiActorSubmissionID}`,
      payout_amount: 30,
      tip_amount: 5,
    },
  );
  assertStatus(acceptedSubmission, 200, "accept worker submission");
  assertScenario(
    requireString(acceptedSubmission.json, "payout_kind") === "credit",
    "accepted submission payout must be credit",
  );
  assertScenario(
    requireNumber(acceptedSubmission.json, "payout_amount") === 30,
    "accepted submission payout amount must match funded reward",
  );
  assertScenario(
    requireNumber(acceptedSubmission.json, "tip_amount") === 5,
    "accepted submission tip amount must round trip",
  );
  assertScenario(
    requireString(acceptedSubmission.json, "worker_user_id") ===
      worker.subjectID,
    "accepted submission worker must be the worker actor",
  );

  const workerBalance = await worker.client.request(
    "GET",
    "/api/credits/balance",
    noScenarioBody,
  );
  assertStatus(workerBalance, 200, "worker balance after accept");
  assertScenario(
    requireNumber(workerBalance.json, "amount") === 135,
    "worker balance must include payout and tip",
  );

  const ownerNotifications = await owner.client.request(
    "GET",
    "/api/notifications",
    noScenarioBody,
  );
  assertStatus(ownerNotifications, 200, "owner notifications");
  const ownerNotificationList = requireArray(
    ownerNotifications.json,
    "notifications",
  );
  assertScenario(
    ownerNotificationList.some((item) => {
      const notification = requireRecord(item, "ownerNotifications[]");
      return requireString(notification, "kind") === "submission_created" &&
        requireString(notification, "actor_user_id") === worker.subjectID &&
        requireString(notification, "recipient_user_id") === owner.subjectID;
    }),
    "owner inbox must include worker submission notification",
  );

  const workerNotifications = await worker.client.request(
    "GET",
    "/api/notifications",
    noScenarioBody,
  );
  assertStatus(workerNotifications, 200, "worker notifications");
  const workerNotificationList = requireArray(
    workerNotifications.json,
    "notifications",
  );
  const acceptedNotification = workerNotificationList.find((item) => {
    const notification = requireRecord(item, "workerNotifications[]");
    return requireString(notification, "kind") === "submission_accepted" &&
      requireString(notification, "actor_user_id") === owner.subjectID &&
      requireString(notification, "recipient_user_id") === worker.subjectID;
  });
  assertScenario(
    acceptedNotification !== undefined,
    "worker inbox must include owner acceptance notification",
  );
  const readAcceptedNotification = await worker.client.request(
    "POST",
    `/api/notifications/${
      requireString(
        requireRecord(acceptedNotification, "acceptedNotification"),
        "id",
      )
    }/read`,
    noScenarioBody,
  );
  assertStatus(readAcceptedNotification, 200, "worker mark acceptance read");
  assertNotificationShape(readAcceptedNotification.json);
  assertScenario(
    requireString(readAcceptedNotification.json, "state") === "read",
    "worker acceptance notification state must change to read",
  );

  const notifications = await client.request(
    "GET",
    "/api/notifications",
    noScenarioBody,
  );
  assertStatus(notifications, 200, "list notifications");
  const notificationList = requireArray(notifications.json, "notifications");
  if (notificationList.length > 0) {
    const firstNotification = requireRecord(
      notificationList[0],
      "notifications[0]",
    );
    assertNotificationShape(firstNotification);
    const readNotification = await client.request(
      "POST",
      `/api/notifications/${requireString(firstNotification, "id")}/read`,
      noScenarioBody,
    );
    assertStatus(readNotification, 200, "mark notification read");
    assertNotificationShape(readNotification.json);
    assertScenario(
      requireString(readNotification.json, "state") === "read",
      "notification state must change to read",
    );
  }
}
