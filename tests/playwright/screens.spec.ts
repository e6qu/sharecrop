import { expect, type Page, test } from "@playwright/test";
import { Buffer } from "node:buffer";
import {
  type AuthBody,
  fillDetailResponse,
  password,
  taskRequest,
  uniqueEmail,
} from "./helpers.ts";

interface TaskBody {
  id: string;
}

interface SubmissionCreatedBody {
  submission: {
    id: string;
  };
}

async function registerViaApi(
  request: {
    post: (
      url: string,
      opts: { data: unknown },
    ) => Promise<{
      ok: () => boolean;
      json: () => Promise<unknown>;
      status: () => number;
      text: () => Promise<string>;
    }>;
  },
  prefix: string,
): Promise<{ email: string; body: AuthBody }> {
  const email = uniqueEmail(prefix);
  const response = await request.post("/api/auth/register", {
    data: { email, password },
  });
  const responseText = await response.text();
  expect(
    response.ok(),
    `register ${email} failed with ${response.status()}: ${responseText}`,
  ).toBeTruthy();
  return { email, body: JSON.parse(responseText) as AuthBody };
}

async function loginViaUi(page: Page, email: string): Promise<void> {
  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toBeVisible({ timeout: 15000 });
}

async function logoutViaUi(page: Page): Promise<void> {
  await page.getByTestId("logout").click();
  await expect(page.getByTestId("email")).toBeVisible();
}

async function openTaskFromDiscovery(
  page: Page,
  email: string,
  title: string,
): Promise<void> {
  await loginViaUi(page, email);
  // Wait for the post-login data load to settle before navigating; under load the
  // balance/nav can otherwise race and the discovery click times out.
  await page.waitForLoadState("networkidle");
  await expect(page.getByTestId("balance")).toBeVisible({ timeout: 15000 });
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("discovery-task-row").filter({ hasText: title })
    .getByTestId("discovery-view").click();
}

test("agents discover, submit to, and have a task accepted through the browser", async ({ page, request }) => {
  const owner = await registerViaApi(request, "screens-owner");
  const title = `Discoverable ${crypto.randomUUID()}`;

  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: taskRequest(title, owner.body.subject_id, "public", 20),
  });
  expect(taskResponse.ok()).toBeTruthy();
  const task = (await taskResponse.json()) as TaskBody;

  await request.post(`/api/tasks/${task.id}/funding`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: { amount: 20, idempotency_key: `fund:${task.id}` },
  });
  await request.post(`/api/tasks/${task.id}/open`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {},
  });

  // Worker discovers the public task and submits a response through the UI.
  const worker = await registerViaApi(request, "screens-worker");
  await loginViaUi(page, worker.email);
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("discovery-filters").click();
  await page.getByTestId("discovery-query").fill(title);
  const workerRow = page.getByTestId("discovery-task-row").filter({
    hasText: title,
  });
  await expect(workerRow).toHaveCount(1);
  await workerRow.getByTestId("discovery-view").click();
  await expect(page.getByTestId("detail-title")).toContainText(title);

  await fillDetailResponse(page, '{"answer":"from the browser"}');
  await page.getByTestId("detail-submit").click();
  await expect(page.getByTestId("detail-submit-message")).toBeVisible();
  await expect(page.getByTestId("my-submission-comments-toggle")).toHaveText(
    "Discussion open",
  );
  await expect(page.getByTestId("submission-comments-thread")).toBeVisible();

  // Owner reviews and accepts the submission through the UI.
  await page.getByTestId("nav-account-menu").click();
  await logoutViaUi(page);
  await loginViaUi(page, owner.email);
  await expect(page.getByTestId("balance")).toHaveText("80 credits");
  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-inbox").click();
  const submissionNotification = page.getByTestId("notification-row").filter({
    hasText: "submission_created",
  });
  await expect(submissionNotification.getByTestId("notification-task-link"))
    .toBeVisible();
  await submissionNotification.getByTestId("notification-task-link").click();
  await expect(page.getByTestId("detail-title")).toContainText(title);

  await page.getByTestId("nav-tasks").click();
  const ownerRow = page.getByTestId("discovery-task-row").filter({
    hasText: title,
  });
  await ownerRow.getByTestId("discovery-view").click();
  await expect(page.getByTestId("submission-row")).toHaveCount(1);
  await page.getByTestId("accept-submission").click();
  await expect(page.getByTestId("submission-comments-toggle")).toHaveText(
    "Discussion open",
  );
  await expect(page.getByTestId("submission-comments-thread")).toBeVisible();
  await expect(page.getByTestId("accept-submission")).toHaveCount(0);
});

test("requesters configure reservations and workers include reserved tasks", async ({ page, request }) => {
  const owner = await registerViaApi(request, "reservation-ui-owner");
  const title = `Reserved UI ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill(
    "Reservation required from the browser.",
  );
  await page.getByTestId("create-task-ownership").click();
  await page.getByTestId("create-participation-reservation_required").click();
  await page.getByTestId("create-visibility-public").click();
  await page.getByTestId("create-task").click();
  // Creating a task now opens it in the UI for further editing.
  await expect(page.getByTestId("detail-title")).toBeVisible();

  await page.getByTestId("nav-tasks").click();
  const ownerRow = page.getByTestId("task-row").filter({ hasText: title });
  await expect(ownerRow).toHaveCount(1);
  await ownerRow.getByTestId("view-task").click();
  await expect(page.getByTestId("toggle-integration")).toBeVisible();
  await page.getByTestId("open-task").click();
  await expect(page.getByTestId("task-action-message")).toContainText(
    "Task opened",
  );

  const otherTitle = `Reserved UI unrelated ${crypto.randomUUID()}`;
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  await page.getByTestId("create-title").fill(otherTitle);
  await page.getByTestId("create-description").fill(
    "An unrelated open task with no reservation requirement.",
  );
  await page.getByTestId("create-visibility-public").click();
  await page.getByTestId("create-task").click();
  // Creating a task now opens it in the UI for further editing.
  await expect(page.getByTestId("detail-title")).toBeVisible();
  await page.getByTestId("nav-tasks").click();
  const otherOwnerRow = page.getByTestId("task-row").filter({
    hasText: otherTitle,
  });
  await otherOwnerRow.getByTestId("view-task").click();
  await page.getByTestId("open-task").click();
  await expect(page.getByTestId("task-action-message")).toContainText(
    "Task opened",
  );

  const worker = await registerViaApi(request, "reservation-ui-worker");
  await page.getByTestId("nav-account-menu").click();
  await logoutViaUi(page);
  await loginViaUi(page, worker.email);
  await page.getByTestId("nav-tasks").click();
  const workerRow = page.getByTestId("discovery-task-row").filter({
    hasText: title,
  });
  await expect(workerRow).toHaveCount(1);
  await workerRow.getByTestId("discovery-view").click();
  await expect(page.getByTestId("reserve-task")).toBeVisible();
  await page.getByTestId("reserve-task").click();
  await expect(page.getByTestId("reservation-message")).toContainText(
    "active",
  );
  // Becoming active auto-issues a task-scoped agent credential, revealed once.
  await expect(page.getByTestId("reservation-agent-secret")).toContainText(
    "scrop_agent_",
  );

  // The worker who holds this reservation must see their own reservation row
  // (with a Cancel button) without being the task's owner - the list
  // endpoint used to reject anyone but the owner outright, so a worker's own
  // reservation was never visible to them at all, leaving no way to cancel it.
  await expect(page.getByTestId("reservation-row")).toHaveCount(1);

  // Navigating to an unrelated task must not leak this task's one-time secret.
  await page.getByTestId("nav-tasks").click();
  const otherWorkerRow = page.getByTestId("discovery-task-row").filter({
    hasText: otherTitle,
  });
  await otherWorkerRow.getByTestId("discovery-view").click();
  await expect(page.getByTestId("reservation-agent-secret")).toHaveCount(0);

  const other = await registerViaApi(request, "reservation-ui-other");
  await page.getByTestId("detail-back").click();
  await page.getByTestId("nav-account-menu").click();
  await logoutViaUi(page);
  await loginViaUi(page, other.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("discovery-filters").click();
  await page.getByTestId("discovery-query").fill(title);
  await expect(
    page.getByTestId("discovery-task-row").filter({ hasText: title }),
  ).toHaveCount(0);
  await page.getByTestId("include-reserved").click();
  await expect(
    page.getByTestId("discovery-task-row").filter({ hasText: title }),
  ).toHaveCount(1);
});

test("a worker cancels their own active reservation", async ({ page, request }) => {
  const owner = await registerViaApi(request, "reservation-cancel-owner");
  const title = `Reservation cancel ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill(
    "Reservation cancel from the browser.",
  );
  await page.getByTestId("create-task-ownership").click();
  await page.getByTestId("create-participation-reservation_required").click();
  await page.getByTestId("create-visibility-public").click();
  await page.getByTestId("create-task").click();
  await expect(page.getByTestId("detail-title")).toBeVisible();
  await page.getByTestId("open-task").click();
  await expect(page.getByTestId("task-action-message")).toContainText(
    "Task opened",
  );

  const worker = await registerViaApi(request, "reservation-cancel-worker");
  await page.getByTestId("nav-account-menu").click();
  await logoutViaUi(page);
  await loginViaUi(page, worker.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("discovery-task-row").filter({ hasText: title })
    .getByTestId("discovery-view").click();
  await page.getByTestId("reserve-task").click();
  await expect(page.getByTestId("reservation-message")).toContainText(
    "active",
  );

  // The worker sees their own reservation with a Cancel button, and cancelling
  // it releases the task back to "reserve" for themselves (and, per the
  // assertion above this test, anyone else).
  await expect(page.getByTestId("reservation-row")).toHaveCount(1);
  await page.getByTestId("cancel-reservation").click();
  await expect(page.getByTestId("reservation-message")).toContainText(
    "cancelled",
  );
  // The row stays (a cancelled reservation is still shown, as a record) but
  // its Cancel button goes away, and the task becomes reservable again.
  await expect(page.getByTestId("cancel-reservation")).toHaveCount(0);
  await expect(page.getByTestId("reserve-task")).toBeVisible();
});

test("the active implementor refunds a funded task they reserved", async ({ page, request }) => {
  const owner = await registerViaApi(request, "worker-refund-owner");
  const title = `Worker refund ${crypto.randomUUID()}`;

  // Owner creates a funded credit-reward task that requires a reservation, then
  // funds and opens it - all via API, so the test focuses on the worker's UI.
  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {
      owner: {
        kind: "user",
        user_id: owner.body.subject_id,
        team_id: "",
        organization_id: "",
      },
      title,
      description: "A funded task an implementor can refund.",
      reward: { kind: "credit", credit_amount: 15 },
      visibility: {
        kind: "public",
        user_id: "",
        team_id: "",
        organization_id: "",
      },
      placement: {
        kind: "standalone",
        series_id: "",
        series_title: "",
        series_position: 0,
      },
      participation: {
        policy: "reservation_required",
        assignee_scope: "user",
        reservation_expiry_hours: 48,
      },
      response_schema_json: '{"kind":"freeform"}',
      payload: { kind: "none", json: "" },
    },
  });
  expect(taskResponse.ok()).toBeTruthy();
  const task = (await taskResponse.json()) as TaskBody;
  await request.post(`/api/tasks/${task.id}/funding`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: { amount: 15, idempotency_key: `fund:${task.id}` },
  });
  await request.post(`/api/tasks/${task.id}/open`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {},
  });

  // A worker reserves the task, becoming the active implementor.
  const worker = await registerViaApi(request, "worker-refund-worker");
  await loginViaUi(page, worker.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("discovery-task-row").filter({ hasText: title })
    .getByTestId("discovery-view").click();
  await page.getByTestId("reserve-task").click();
  await expect(page.getByTestId("reservation-message")).toContainText("active");

  // The worker sees a "Refund reward" control - named a refund, not a reclaim -
  // with an info toggle explaining it. The owner's Reclaim is never shown to a
  // worker.
  await expect(page.getByTestId("worker-refund-task")).toHaveText(
    "Refund reward",
  );
  await expect(page.getByTestId("refund-task")).toHaveCount(0);
  await page.getByTestId("worker-refund-task-info").getByText("What this does")
    .click();
  await expect(page.getByTestId("worker-refund-task-info")).toContainText(
    "Refund returns the reward to the requester",
  );

  // Refunding returns the reward to the requester and cancels the task; the
  // confirmation is echoed into the reservation card the worker is looking at.
  await page.getByTestId("worker-refund-task").click();
  await expect(page.getByTestId("worker-task-action-message")).toContainText(
    "Reward returned and the task was cancelled.",
  );
});

test("an owner approves a worker's reservation request from the task detail page", async ({ page, request }) => {
  const owner = await registerViaApi(request, "reservation-approve-owner");
  const title = `Approval required ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill(
    "Approval required from the browser.",
  );
  await page.getByTestId("create-task-ownership").click();
  await page.getByTestId("create-participation-approval_required").click();
  await page.getByTestId("create-visibility-public").click();
  await page.getByTestId("create-task").click();
  // Creating a task now opens it in the UI for further editing.
  await expect(page.getByTestId("detail-title")).toBeVisible();
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("task-row").filter({ hasText: title }).getByTestId(
    "view-task",
  ).click();
  await page.getByTestId("open-task").click();
  await expect(page.getByTestId("task-action-message")).toContainText(
    "Task opened",
  );

  const worker = await registerViaApi(request, "reservation-approve-worker");
  await page.getByTestId("nav-account-menu").click();
  await logoutViaUi(page);
  await loginViaUi(page, worker.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("discovery-task-row").filter({ hasText: title })
    .getByTestId("discovery-view").click();
  await page.getByTestId("request-approval").click();
  await expect(page.getByTestId("reservation-message")).toContainText(
    "requested",
  );

  // The owner sees the pending request at the top of the task detail page
  // (the Reservation section, above owner controls) and can approve it —
  // previously there was no way to do this through the browser at all.
  await page.getByTestId("detail-back").click();
  await page.getByTestId("nav-account-menu").click();
  await logoutViaUi(page);
  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("task-row").filter({ hasText: title }).getByTestId(
    "view-task",
  ).click();
  const reservationRow = page.getByTestId("reservation-row").filter({
    hasText: worker.body.subject_id,
  });
  await expect(reservationRow).toHaveCount(1);
  await reservationRow.getByTestId("approve-reservation").click();
  await expect(page.getByTestId("reservation-message")).toContainText(
    "active",
  );
});

test("requesters upload small task attachments through the real backend", async ({ page, request }) => {
  const owner = await registerViaApi(request, "upload-ui-owner");
  const title = `Attached task ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill(
    "Task attachment through the DB-backed UI.",
  );
  await page.getByTestId("create-visibility-public").click();
  await page.getByTestId("create-advanced-options").click();

  const chooser = page.waitForEvent("filechooser");
  await page.getByTestId("create-attachments-pick").click();
  await (await chooser).setFiles({
    name: "brief.txt",
    mimeType: "text/plain",
    buffer: Buffer.from("hello"),
  });
  await expect(page.getByTestId("selected-attachment")).toContainText(
    "brief.txt",
  );

  await page.getByTestId("create-task").click();
  // Creating a task now opens it in the UI for further editing.
  await expect(page.getByTestId("detail-title")).toBeVisible();
  await page.getByTestId("nav-tasks").click();
  const row = page.getByTestId("task-row").filter({ hasText: title });
  await expect(row).toHaveCount(1);
  await row.getByTestId("view-task").click();
  await expect(page.getByTestId("detail-attachments")).toContainText(
    "brief.txt",
  );
});

test("requesters see attachment guardrails through the real backend UI", async ({ page, request }) => {
  const owner = await registerViaApi(request, "upload-edge-owner");
  const title = `Attachment guardrails ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill(
    "Task attachment guardrails through the DB-backed UI.",
  );
  await page.getByTestId("create-visibility-public").click();
  await page.getByTestId("create-advanced-options").click();

  async function pickAttachment(
    name: string,
    mimeType: string,
    buffer: Buffer,
  ): Promise<void> {
    const chooser = page.waitForEvent("filechooser");
    await page.getByTestId("create-attachments-pick").click();
    await (await chooser).setFiles({ name, mimeType, buffer });
  }

  await pickAttachment(
    "archive.bin",
    "application/octet-stream",
    Buffer.from("not allowed"),
  );
  await expect(page.getByTestId("create-message")).toContainText(
    "Attachment type is not allowed.",
  );

  await pickAttachment(
    "large.txt",
    "text/plain",
    Buffer.alloc(500 * 1024 + 1, "x"),
  );
  await expect(page.getByTestId("create-message")).toContainText(
    "Attachment must be under 500 KiB.",
  );

  for (let index = 0; index < 5; index += 1) {
    await pickAttachment(
      `brief-${index}.txt`,
      "text/plain",
      Buffer.from(`hello ${index}`),
    );
  }
  await expect(page.getByTestId("selected-attachment")).toHaveCount(5);
  await page.getByTestId("create-attachments-pick").click();
  await expect(page.getByTestId("create-message")).toContainText(
    "Attach up to 5 files.",
  );
});

test("users create and see their organizations", async ({ page, request }) => {
  const owner = await registerViaApi(request, "org-ui-owner");
  const name = `Org UI ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-organizations").click();
  await expect(page.getByTestId("organizations-empty")).toBeVisible();

  await page.getByTestId("create-org-name").fill(name);
  await page.getByTestId("create-org").click();
  await expect(page.getByTestId("org-message")).toContainText(
    "Created organization",
  );
  await expect(
    page.getByTestId("organization-row").filter({ hasText: name }),
  ).toHaveCount(1);
});

test("users open an organization and manage its teams and members", async ({ page, request }) => {
  const owner = await registerViaApi(request, "org-ctx-owner");
  const member = await registerViaApi(request, "org-ctx-member");
  const orgName = `Ctx Org ${crypto.randomUUID()}`;
  const teamName = `Crew ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-organizations").click();
  await page.getByTestId("create-org-name").fill(orgName);
  await page.getByTestId("create-org").click();
  await expect(page.getByTestId("org-message")).toContainText(
    "Created organization",
  );

  await page.getByTestId("select-organization").first().click();
  await expect(page).toHaveURL(/\/organizations\/[0-9a-f-]+$/);
  await expect(page.getByTestId("active-organization")).toBeVisible();
  await expect(page.getByTestId("org-operations-dashboard")).toBeVisible();
  await expect(page.getByTestId("org-ops-members-active")).toContainText("1");
  await expect(page.getByTestId("org-tasks-heading")).toContainText(
    "Organization tasks (0)",
  );
  await page.getByTestId("org-task-filters").click();
  await expect(page.getByTestId("org-task-query")).toBeVisible();
  await page.getByTestId("org-task-query").fill("missing task");
  await page.getByTestId("org-task-search").click();
  await expect(page.getByTestId("org-tasks-empty")).toBeVisible();
  await expect(page.getByTestId("org-tasks-page-offset")).toHaveText(
    "Offset 0",
  );
  await page.getByTestId("org-task-filter-open").click();
  await page.getByTestId("org-task-saved-view-name").fill("Open tasks");
  await page.getByTestId("org-task-save-view").click();
  await expect(page.getByTestId("org-task-saved-view")).toContainText(
    "Open tasks",
  );
  await expect(page.getByTestId("org-task-saved-view")).toContainText("Open");
  await expect(page.getByTestId("org-task-saved-view")).toContainText(
    "Newest",
  );
  await page.getByTestId("org-task-query").fill("");
  await page.getByTestId("org-task-saved-view").click();
  await expect(page.getByTestId("org-task-query")).toHaveValue("missing task");
  // The owner is a real member of the org they created.
  await expect(page.getByTestId("org-member-row")).toHaveCount(1);

  await page.getByTestId("org-teams-section").click();
  await page.getByTestId("create-org-team-name").fill(teamName);
  await page.getByTestId("create-org-team").click();
  await expect(
    page.getByTestId("org-team-row").filter({ hasText: teamName }),
  ).toHaveCount(1);

  await page.getByTestId("org-members-section").click();
  await page.getByTestId("provision-member-email").fill(member.email);
  await page.getByTestId("provision-member").click();
  await expect(page.getByTestId("provision-member-message")).toContainText(
    "provisioned",
  );
  // The provisioned member now appears in the real member list (owner + member).
  await expect(page.getByTestId("org-member-row")).toHaveCount(2);
  const provisionedMemberRow = page.getByTestId("org-member-row").filter({
    hasText: member.body.subject_id,
  });
  await expect(provisionedMemberRow).toContainText("member · active");
  await provisionedMemberRow.getByTestId("member-role-reviewer").click();
  await expect(provisionedMemberRow).toContainText("member, reviewer · active");
  await provisionedMemberRow.getByTestId("deactivate-member").click();
  await expect(provisionedMemberRow).toContainText(
    "member, reviewer · deactivated",
  );

  // The team row links to its own page.
  await page.getByTestId("org-team-row").filter({ hasText: teamName }).click();
  await expect(page).toHaveURL(/\/teams\/[0-9a-f-]+$/);
  await expect(page.getByTestId("team-detail-name")).toContainText(teamName);
  await expect(page.getByTestId("team-work-dashboard")).toBeVisible();
  await expect(page.getByTestId("team-review-queue-heading")).toContainText(
    "Review queue (0)",
  );
  await expect(page.getByTestId("team-ready-work-heading")).toContainText(
    "Ready for team (0)",
  );
  await expect(page.getByTestId("team-assigned-work-heading")).toContainText(
    "Assigned to team (0)",
  );
  await page.getByTestId("team-work-filters").click();
  await expect(page.getByTestId("team-work-query")).toBeVisible();
  await page.getByTestId("team-work-query").fill("missing task");
  await page.getByTestId("team-work-search").click();
  await expect(page.getByTestId("team-work-page-offset")).toHaveText(
    "Offset 0",
  );
  await page.getByTestId("team-work-filter-ready").click();
  await page.getByTestId("team-work-saved-view-name").fill("Ready work");
  await page.getByTestId("team-work-save-view").click();
  await expect(page.getByTestId("team-work-saved-view")).toContainText(
    "Ready work",
  );
  await expect(page.getByTestId("team-work-saved-view")).toContainText(
    "Ready",
  );
  await expect(page.getByTestId("team-work-saved-view")).toContainText(
    "Newest",
  );
  await page.getByTestId("team-work-query").fill("");
  await page.getByTestId("team-work-saved-view").click();
  await expect(page.getByTestId("team-work-query")).toHaveValue(
    "missing task",
  );
  await expect(page.getByTestId("team-review-queue-empty")).toBeVisible();
  await expect(page.getByTestId("team-ready-work-empty")).toBeVisible();
});

test("org admins mint and revoke an organization-wide credential", async ({ page, request }) => {
  const owner = await registerViaApi(request, "org-cred-owner");
  const orgName = `Cred Org ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-organizations").click();
  await page.getByTestId("create-org-name").fill(orgName);
  await page.getByTestId("create-org").click();
  await page.getByTestId("select-organization").first().click();
  await expect(page.getByTestId("active-organization")).toBeVisible();

  await page.getByTestId("org-credentials-section").click();
  await page.getByTestId("org-credential-label").fill("Org reporting bot");
  await page.getByTestId("org-scope-org_read").check();
  await page.getByTestId("org-credential-expires-hours").fill("48");
  await page.getByTestId("create-org-credential").click();

  await expect(page.getByTestId("org-credential-secret")).toContainText(
    "scrop_org_",
  );
  const credentialRow = page.getByTestId("org-credential-row").filter({
    hasText: "Org reporting bot",
  });
  await expect(credentialRow).toHaveCount(1);
  await expect(credentialRow).toContainText("expires");

  await credentialRow.getByTestId("revoke-org-credential").click();
  await expect(credentialRow).toContainText("revoked");
});

test("a task's reward shows as its own badge in the task list", async ({ page, request }) => {
  const owner = await registerViaApi(request, "reward-badge-owner");
  const title = `Reward badge ${crypto.randomUUID()}`;

  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: taskRequest(title, owner.body.subject_id, "default", 20),
  });
  expect(taskResponse.ok()).toBeTruthy();
  const task = (await taskResponse.json()) as TaskBody;
  await request.post(`/api/tasks/${task.id}/funding`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: { amount: 20, idempotency_key: `reward-badge:${task.id}` },
  });

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  const row = page.getByTestId("task-row").filter({ hasText: title });
  await expect(row).toHaveCount(1);
  await expect(row).toContainText("20 credits");
  // The reward renders as its own icon-prefixed badge (a rounded pill),
  // distinct from the plain trailing text it used to be.
  const rewardBadge = row.locator("span.rounded-full", {
    hasText: "20 credits",
  });
  await expect(rewardBadge).toHaveCount(1);
  await expect(rewardBadge).toContainText("◆");
});

test("requesters filter their task list by state", async ({ page, request }) => {
  const owner = await registerViaApi(request, "filter-ui-owner");
  const title = `Filter UI ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill("Filter from the browser.");
  await page.getByTestId("create-task").click();
  // Creating a task now opens it in the UI for further editing.
  await expect(page.getByTestId("detail-title")).toBeVisible();

  await page.getByTestId("nav-tasks").click();
  const row = page.getByTestId("task-row").filter({ hasText: title });
  await expect(row).toHaveCount(1);

  // The new task is a draft, so filtering to Open alone hides it.
  await page.getByTestId("tasks-filters").click();
  await page.getByTestId("task-filter-open").click();
  await expect(page.getByTestId("task-row").filter({ hasText: title }))
    .toHaveCount(0);

  // Adding Draft alongside Open (both active at once) shows it again - a
  // multi-select filter, not a single radio choice.
  await page.getByTestId("task-filter-draft").click();
  await expect(page.getByTestId("task-row").filter({ hasText: title }))
    .toHaveCount(1);

  // Deselecting Open, leaving only Draft active, still shows it.
  await page.getByTestId("task-filter-open").click();
  await expect(page.getByTestId("task-row").filter({ hasText: title }))
    .toHaveCount(1);

  await page.getByTestId("tasks-query").fill("not " + title);
  await expect(page.getByTestId("task-row").filter({ hasText: title }))
    .toHaveCount(0);
  await page.getByTestId("tasks-query").fill(title);
  await expect(page.getByTestId("task-row").filter({ hasText: title }))
    .toHaveCount(1);
});

test("a user profile page lists the user's public tasks", async ({ page, request }) => {
  const owner = await registerViaApi(request, "profile-page-owner");
  const title = `Public profile task ${crypto.randomUUID()}`;
  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: taskRequest(title, owner.body.subject_id, "public", 0),
  });
  expect(taskResponse.ok()).toBeTruthy();

  await loginViaUi(page, owner.email);
  // Wait for login to establish the session and refresh cookie before the
  // deep-link reload, otherwise the reload races the login response.
  await expect(page.getByTestId("balance")).toBeVisible();
  await page.goto(`/#/users/${owner.body.subject_id}`);
  await expect(page.getByTestId("user-id")).toContainText(
    owner.body.subject_id,
  );
  await expect(
    page.getByTestId("user-task-row").filter({ hasText: title }),
  ).toHaveCount(1);
  await page.getByTestId("account-privacy").click();
  await page.getByTestId("request-data-export").click();
  await expect(page.getByTestId("account-message")).toContainText(
    "Privacy request queued: data_export",
  );

  // The work and submissions sub-pages are their own linkable URLs.
  await page.getByTestId("user-work-link").click();
  await expect(page).toHaveURL(/\/users\/[^/]+\/work$/);
  await page.getByTestId("back-user").click();
  await page.getByTestId("user-submissions-link").click();
  await expect(page).toHaveURL(/\/users\/[^/]+\/submissions$/);
});

test("workers see requested revisions in their submission inbox", async ({ page, request }) => {
  const owner = await registerViaApi(request, "revision-owner");
  const worker = await registerViaApi(request, "revision-worker");
  const title = `Revision inbox ${crypto.randomUUID()}`;
  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: taskRequest(title, owner.body.subject_id, "public", 0),
  });
  expect(taskResponse.ok()).toBeTruthy();
  const task = (await taskResponse.json()) as TaskBody;
  const openResponse = await request.post(`/api/tasks/${task.id}/open`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {},
  });
  expect(openResponse.ok()).toBeTruthy();
  const submissionResponse = await request.post(
    `/api/tasks/${task.id}/submissions`,
    {
      headers: { Authorization: `Bearer ${worker.body.access_token}` },
      data: { response_json: '{"answer":"revise me"}' },
    },
  );
  const submissionBody = await submissionResponse.text();
  expect(submissionResponse.ok(), submissionBody).toBeTruthy();
  const submitted = JSON.parse(submissionBody) as SubmissionCreatedBody;
  const reviewNote = `Please revise ${crypto.randomUUID()}`;
  const requestChangesResponse = await request.post(
    `/api/tasks/${task.id}/submissions/${submitted.submission.id}/request-changes`,
    {
      headers: { Authorization: `Bearer ${owner.body.access_token}` },
      data: { review_note: reviewNote },
    },
  );
  expect(requestChangesResponse.ok()).toBeTruthy();

  await loginViaUi(page, worker.email);
  await expect(page.getByTestId("balance")).toBeVisible();
  await page.goto(`/#/users/${worker.body.subject_id}/submissions`);
  await expect(page.getByTestId("revision-inbox-heading")).toContainText(
    "Revision inbox (1)",
  );
  await expect(page.getByTestId("revision-inbox")).toBeVisible();
  await expect(page.getByTestId("revision-inbox")).toContainText(task.id);
  await expect(page.getByTestId("revision-inbox")).toContainText(reviewNote);
  await expect(page.getByTestId("revision-timeline")).toBeVisible();
  await expect(page.getByTestId("revision-timeline-heading")).toContainText(
    "Revision timeline (1)",
  );
  await expect(page.getByTestId("revision-timeline-row")).toContainText(
    reviewNote,
  );
  await page.getByTestId("revision-resubmit").click();
  await expect(page).toHaveURL(new RegExp(`/tasks/${task.id}$`));
  await expect
    .poll(async () =>
      JSON.parse(await page.getByTestId("detail-submit-input").inputValue())
    )
    .toEqual({ answer: "revise me" });
  await fillDetailResponse(page, '{"answer":"revised from inbox"}');
  await page.getByTestId("detail-submit").click();
  await expect(page.getByTestId("detail-submit-message")).toBeVisible();

  await page.goto(`/#/users/${worker.body.subject_id}/submissions`);
  await expect(page.getByTestId("revision-timeline-row")).toHaveCount(2);
  await expect(page.getByTestId("revision-timeline-heading")).toContainText(
    "Revision timeline (2)",
  );
  await page.getByTestId("user-submissions-all").click();
  await expect(page.getByTestId("user-submissions")).toContainText(
    "revised from inbox",
  );
});

test("requesters scope a task to a standalone team", async ({ page, request }) => {
  const owner = await registerViaApi(request, "team-scope-owner");
  const teamResponse = await request.post("/api/teams", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: { name: `Crew ${crypto.randomUUID()}` },
  });
  expect(teamResponse.ok()).toBeTruthy();
  const team = (await teamResponse.json()) as { id: string };

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  const title = `Team scoped ${crypto.randomUUID()}`;
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill("Scoped to a team.");
  await page.getByTestId("create-visibility-team").click();
  await page.getByTestId("create-scope-team").selectOption(team.id);
  await page.getByTestId("create-task").click();
  // Creating a task now opens it in the UI for further editing.
  await expect(page.getByTestId("detail-title")).toBeVisible();

  await page.getByTestId("nav-tasks").click();
  await expect(
    page.getByTestId("task-row").filter({ hasText: title }),
  ).toHaveCount(1);
});

test("requesters set a task's assignee scope to a team", async ({ page, request }) => {
  const owner = await registerViaApi(request, "assignee-owner");
  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  const title = `Team assignee ${crypto.randomUUID()}`;
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill("Assigned to a team.");
  await page.getByTestId("create-visibility-public").click();
  await page.getByTestId("create-task-ownership").click();
  await page.getByTestId("create-assignee-organization_team").click();
  await page.getByTestId("create-task").click();
  // Creating a task now opens it in the UI for further editing.
  await expect(page.getByTestId("detail-title")).toBeVisible();

  // Open it so a worker can discover and view it.
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("task-row").filter({ hasText: title }).getByTestId(
    "view-task",
  ).click();
  await page.getByTestId("open-task").click();
  await expect(page.getByTestId("task-action-message")).toContainText(
    "Task opened",
  );

  // A worker viewing the task sees the organization-team assignee scope.
  const worker = await registerViaApi(request, "assignee-worker");
  await page.getByTestId("nav-account-menu").click();
  await logoutViaUi(page);
  await loginViaUi(page, worker.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("discovery-task-row").filter({ hasText: title })
    .getByTestId("discovery-view").click();
  await expect(page.getByTestId("detail-title")).toContainText(title);
  await expect(page.getByText("organization team")).toBeVisible();
});

test("a team owner adds a member to a standalone team", async ({ page, request }) => {
  const owner = await registerViaApi(request, "teammember-owner");
  const member = await registerViaApi(request, "teammember-member");
  const teamResponse = await request.post("/api/teams", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: { name: `Crew ${crypto.randomUUID()}` },
  });
  expect(teamResponse.ok()).toBeTruthy();
  const team = (await teamResponse.json()) as { id: string };

  await loginViaUi(page, owner.email);
  await expect(page.getByTestId("balance")).toBeVisible();
  await page.goto(`/#/teams/${team.id}`);
  await expect(page.getByTestId("team-detail-name")).toBeVisible();
  await expect(page.getByTestId("team-work-dashboard")).toBeVisible();
  await expect(page.getByTestId("team-members-empty")).toBeVisible();

  await page.getByTestId("team-member-email").fill(member.email);
  await page.getByTestId("add-team-member").click();
  await expect(page.getByTestId("team-member-row")).toHaveCount(1);
});

test("pages have their own URLs and deep links load", async ({ page, request }) => {
  const owner = await registerViaApi(request, "routing-owner");
  await loginViaUi(page, owner.email);

  // Link navigation updates the address bar and shows the page.
  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-agents").click();
  await expect(page).toHaveURL(/\/agents$/);
  await expect(page.getByTestId("agent-label")).toBeVisible();

  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-collectibles").click();
  await expect(page).toHaveURL(/\/collectibles$/);
  await expect(page.getByTestId("collectibles-mint")).toBeVisible();
  // The admin-only award panel is hidden for a non-admin (role=member); the
  // catalog gallery is still browsable but without Award buttons.
  await expect(page.getByTestId("award-recipient-id")).toHaveCount(0);
  await expect(page.getByTestId("catalog-award")).toHaveCount(0);
  await expect(page.getByTestId("catalog-entry").first()).toBeVisible();

  // Deep-linking directly to a page loads the app on that page with the
  // session restored from the refresh cookie.
  await page.goto("/#/organizations");
  await expect(page).toHaveURL(/#\/organizations$/);
  await expect(page.getByTestId("create-org-name")).toBeVisible();

  // Hash routing: a hard refresh on a deep link stays put — no server fallback.
  await page.reload();
  await expect(page).toHaveURL(/#\/organizations$/);
  await expect(page.getByTestId("create-org-name")).toBeVisible();

  // Unknown routes get an explicit not-found page, not a silent redirect.
  await page.goto("/#/does-not-exist");
  await expect(page.getByTestId("not-found-home")).toBeVisible();

  // The demo-only Reset button is absent on the real app (demo flag is false).
  await expect(page.getByTestId("reset-demo")).toHaveCount(0);
});

test("requesters author a response schema and task input that the detail surfaces", async ({ page, request }) => {
  const owner = await registerViaApi(request, "schema-author");
  const title = `Authored ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill(
    "Review the PR in the task input and list your findings.",
  );
  // A human can now author a structured (non-freeform) response schema and an
  // embedded payload, both previously hardcoded in the client.
  await page.getByTestId("create-advanced-options").click();
  await page.getByTestId("create-response-schema").fill(
    '{"kind":"array","item":{"kind":"string"}}',
  );
  await page.getByTestId("create-payload").fill(
    '{"pr_url":"https://example.test/pr/7"}',
  );
  await page.getByTestId("create-visibility-public").click();
  await page.getByTestId("create-task").click();
  // Creating a task now opens it in the UI for further editing.
  await expect(page.getByTestId("detail-title")).toBeVisible();

  await page.getByTestId("nav-tasks").click();
  const ownerRow = page.getByTestId("task-row").filter({ hasText: title });
  await expect(ownerRow).toHaveCount(1);
  await ownerRow.getByTestId("view-task").click();

  // The authored payload renders as the Task input, and the authored schema is
  // shown back on the task detail.
  await expect(page.getByTestId("detail-input")).toContainText("pr_url");
  await expect(page.getByTestId("detail-schema")).toContainText("array");
});

test("a creator manages a first-class task series end to end", async ({ page, request }) => {
  const owner = await registerViaApi(request, "series-owner");
  const seriesTitle = `Series ${crypto.randomUUID()}`;

  // A task the owner created, to add to the series.
  const taskTitle = `Series task ${crypto.randomUUID()}`;
  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: taskRequest(taskTitle, owner.body.subject_id, "public", 0),
  });
  const seriesTask = (await taskResponse.json()) as TaskBody;

  await loginViaUi(page, owner.email);

  // Create a series from the Series section on the Tasks hub.
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("tasks-series").click();
  await page.getByTestId("series-create-title").fill(seriesTitle);
  await page.getByTestId("series-create-description").fill(
    "A demo onboarding series with review rounds.",
  );
  await page.getByTestId("create-series").click();
  const seriesRow = page.getByTestId("series-row").filter({
    hasText: seriesTitle,
  });
  await expect(seriesRow).toHaveCount(1);

  // Open it; the URL is stable and the detail renders.
  await seriesRow.getByTestId("open-series").click();
  await expect(page).toHaveURL(/\/series\/[0-9a-f-]+$/);
  await expect(page.getByTestId("series-detail-title")).toContainText(
    seriesTitle,
  );

  // Add the owner's task to the series, then publish it.
  await page.getByTestId("series-creator-controls").click();
  await page.getByTestId("series-add-task-id").selectOption(seriesTask.id);
  await page.getByTestId("series-add-task").click();
  await expect(
    page.getByTestId("series-task-row").filter({ hasText: taskTitle }),
  ).toHaveCount(1);

  await page.getByTestId("series-publish").click();
  await expect(page.getByTestId("series-state")).toContainText("ublished");

  // Comment on the series.
  await page.getByTestId("series-comments-section").click();
  await page.getByTestId("series-comment-body").fill(
    "Round one is ready for review.",
  );
  await page.getByTestId("add-series-comment").click();
  await expect(
    page.getByTestId("series-comment").filter({ hasText: "Round one" }),
  ).toHaveCount(1);
});

test("a requester uses a code-review template with a PR link and comments on the task", async ({ page, request }) => {
  const owner = await registerViaApi(request, "template-owner");
  const title = `Code review ${crypto.randomUUID()}`;
  const prURL = "https://github.com/example/repo/pull/7";

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  await page.getByTestId("create-title").fill(title);
  // Pick the code-review template (prefills description + response schema) and
  // point it at a specific pull request. Reference URL lives under Advanced
  // options, which is collapsed by default.
  await page.getByTestId("create-task-type").selectOption("code_review");
  await page.getByTestId("create-advanced-options").click();
  await page.getByTestId("create-reference-url").fill(prURL);
  await page.getByTestId("create-visibility-public").click();
  await page.getByTestId("create-task").click();
  // Creating a task now opens it in the UI for further editing.
  await expect(page.getByTestId("detail-title")).toBeVisible();

  await page.getByTestId("nav-tasks").click();
  const ownerRow = page.getByTestId("task-row").filter({ hasText: title });
  await ownerRow.getByTestId("view-task").click();

  // The task shows its type and the clickable PR reference.
  await expect(page.getByTestId("detail-type")).toBeVisible();
  await expect(page.getByTestId("detail-reference")).toHaveAttribute(
    "href",
    prURL,
  );

  // Comment on the task (clarifying question).
  await page.getByTestId("task-comment-body").fill(
    "Which branch should I diff against?",
  );
  await page.getByTestId("add-task-comment").click();
  await expect(
    page.getByTestId("task-comment").filter({ hasText: "Which branch" }),
  ).toHaveCount(1);
});

test("a requester builds a response schema with the structured designer", async ({ page, request }) => {
  const owner = await registerViaApi(request, "designer-owner");
  const title = `Designed ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill(
    "Summarize the linked material.",
  );

  // Build the schema from fields instead of writing JSON; the advanced textarea
  // reflects the generated schema.
  await page.getByTestId("schema-add-field").click();
  await page.getByTestId("schema-field-name").fill("summary");
  await expect(page.getByTestId("create-response-schema")).toHaveValue(
    /"name":"summary"/,
  );

  // A second field as an enum with allowed values flows into the schema JSON.
  await page.getByTestId("schema-add-field").click();
  await page.getByTestId("schema-field-name").nth(1).fill("rating");
  await page.getByTestId("schema-field-kind").nth(1).selectOption("enum");
  await page.getByTestId("schema-field-enum-values").fill("low, medium, high");
  await expect(page.getByTestId("create-response-schema")).toHaveValue(
    /"kind":"enum"/,
  );
  await expect(page.getByTestId("create-response-schema")).toHaveValue(
    /"high"/,
  );

  await page.getByTestId("create-visibility-public").click();
  await page.getByTestId("create-task").click();
  // Creating a task now opens it in the UI for further editing.
  await expect(page.getByTestId("detail-title")).toBeVisible();

  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("task-row").filter({ hasText: title }).getByTestId(
    "view-task",
  ).click();
  await expect(page.getByTestId("detail-schema")).toContainText("summary");
});

test("the task API & MCP panel mints a real token and shows placeholder-free commands", async ({ page, request }) => {
  const owner = await registerViaApi(request, "integration-owner");
  const title = `Integration ${crypto.randomUUID()}`;
  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: taskRequest(title, owner.body.subject_id, "public", 0),
  });
  const task = (await taskResponse.json()) as TaskBody;

  await loginViaUi(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("task-row").filter({ hasText: title }).getByTestId(
    "view-task",
  ).click();

  // The integration panel is collapsed by default.
  await expect(page.getByTestId("mint-task-token")).toBeHidden();
  await page.getByTestId("toggle-integration").click();

  // Mint an agent token; the commands then embed the real token (no <...>).
  await page.getByTestId("mint-task-token").click();
  await expect(page.getByTestId("integration-token")).toBeVisible();
  const restSubmit = page.getByTestId("integration-rest-submit");
  await expect(restSubmit).toContainText(`/api/tasks/${task.id}/submissions`);
  await expect(restSubmit).toContainText("Authorization: Bearer ");
  await expect(restSubmit).not.toContainText("<");
  await expect(page.getByTestId("integration-mcp-config")).not.toContainText(
    "<",
  );
  await expect(page.getByTestId("copy-command").first()).toBeVisible();
});

test("a user mints a personal agent token with MCP install commands on their own page", async ({ page, request }) => {
  const owner = await registerViaApi(request, "user-token");
  await loginViaUi(page, owner.email);
  await expect(page.getByTestId("balance")).toBeVisible();

  // The Profile nav link goes to the user's own page, where the agent-access
  // section is present.
  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-profile").click();
  await expect(page.getByTestId("mint-user-token")).toBeVisible();
  await page.getByTestId("mint-user-token").click();
  await expect(page.getByTestId("user-token")).toBeVisible();
  await expect(page.getByTestId("user-mcp-install")).toContainText(
    "claude mcp add",
  );
  await expect(page.getByTestId("user-mcp-install")).toContainText("/mcp");
  await expect(page.getByTestId("user-mcp-install")).not.toContainText("<");
  await expect(page.getByTestId("copy-command").first()).toBeVisible();

  // Another user's page does not expose the token section.
  const other = await registerViaApi(request, "user-token-other");
  await page.goto(`/#/users/${other.body.subject_id}`);
  await expect(page.getByTestId("user-id")).toBeVisible();
  await expect(page.getByTestId("mint-user-token")).toHaveCount(0);
});

test("the create-task template menu prefills the schema, and Freeform shows the designer", async ({ page, request }) => {
  const owner = await registerViaApi(request, "template-menu");
  await loginViaUi(page, owner.email);
  await expect(page.getByTestId("balance")).toBeVisible();
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();

  // Freeform (the default) shows the structured schema designer.
  await expect(page.getByTestId("schema-add-field")).toBeVisible();

  // Choosing a template prefills the description + schema and replaces the
  // designer with an explanatory note.
  await page.getByTestId("create-task-type").selectOption("code_review");
  await expect(page.getByTestId("template-schema-note")).toBeVisible();
  await expect(page.getByTestId("schema-add-field")).toHaveCount(0);
  await expect(page.getByTestId("create-response-schema")).toHaveValue(
    /"verdict"/,
  );
  await expect(page.getByTestId("create-description")).not.toHaveValue("");

  // Back to Freeform restores the designer and resets the schema.
  await page.getByTestId("create-task-type").selectOption("general");
  await expect(page.getByTestId("schema-add-field")).toBeVisible();
  await expect(page.getByTestId("create-response-schema")).toHaveValue(
    /freeform/,
  );
});

test("submitting the create-task form with missing fields highlights them, and typing clears it", async ({ page, request }) => {
  const owner = await registerViaApi(request, "create-validation");
  await loginViaUi(page, owner.email);
  await expect(page.getByTestId("balance")).toBeVisible();
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();

  // Neither field is flagged before a submit attempt.
  await expect(page.getByTestId("create-title")).not.toHaveClass(
    /border-red-400/,
  );
  await expect(page.getByTestId("create-description")).not.toHaveClass(
    /border-red-400/,
  );

  // Both empty: submitting flags both fields, each with its own message.
  await page.getByTestId("create-task").click();
  await expect(page.getByTestId("create-title")).toHaveClass(/border-red-400/);
  await expect(page.getByTestId("create-description")).toHaveClass(
    /border-red-400/,
  );
  await expect(page.getByText("Title is required")).toBeVisible();
  await expect(page.getByText("Description is required")).toBeVisible();

  // Filling in the title alone clears just that field's flag.
  await page.getByTestId("create-title").fill("A real title");
  await expect(page.getByTestId("create-title")).not.toHaveClass(
    /border-red-400/,
  );
  await expect(page.getByTestId("create-description")).toHaveClass(
    /border-red-400/,
  );

  // Filling in the description and submitting succeeds.
  await page.getByTestId("create-description").fill("A real description.");
  await page.getByTestId("create-task").click();
  await expect(page.getByTestId("detail-title")).toHaveText("A real title");
});

test("owner and worker exchange comments on a submission", async ({ page, request }) => {
  const owner = await registerViaApi(request, "subcomment-owner");
  const title = `Comment thread ${crypto.randomUUID()}`;
  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: taskRequest(title, owner.body.subject_id, "public", 20),
  });
  const task = (await taskResponse.json()) as TaskBody;
  await request.post(`/api/tasks/${task.id}/funding`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: { amount: 20, idempotency_key: `fund-${task.id}` },
  });
  await request.post(`/api/tasks/${task.id}/open`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {},
  });

  // Worker submits.
  const worker = await registerViaApi(request, "subcomment-worker");
  await loginViaUi(page, worker.email);
  await expect(page.getByTestId("balance")).toBeVisible();
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("discovery-task-row").filter({ hasText: title })
    .getByTestId("discovery-view").click();
  await fillDetailResponse(page, '{"answer":"done"}');
  await page.getByTestId("detail-submit").click();
  await expect(page.getByTestId("detail-submit-message")).toBeVisible();

  // Owner opens the submission and starts a comment thread on it.
  await page.getByTestId("nav-account-menu").click();
  await logoutViaUi(page);
  await loginViaUi(page, owner.email);
  await expect(page.getByTestId("balance")).toBeVisible();
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("discovery-task-row").filter({ hasText: title })
    .getByTestId("discovery-view").click();
  await expect(page.getByTestId("submission-row")).toHaveCount(1);
  await page.getByTestId("submission-comments-toggle").click();
  await expect(page.getByTestId("submission-comments-empty")).toBeVisible();
  await page.getByTestId("submission-comment-body").fill(
    "Could you clarify step 2?",
  );
  await page.getByTestId("add-submission-comment").click();
  await expect(page.getByTestId("submission-comment")).toContainText(
    "Could you clarify step 2?",
  );
});

test("an owner unpublishes an open task back to draft to fix its funding", async ({ page, request }) => {
  const owner = await registerViaApi(request, "unpublish-owner");
  const title = `Unpublish recovery ${crypto.randomUUID()}`;

  // A task that declares a credit reward at creation is not automatically
  // funded - opening it before funding must be rejected on the real backend.
  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: taskRequest(title, owner.body.subject_id, "default", 20),
  });
  expect(taskResponse.ok()).toBeTruthy();
  const task = (await taskResponse.json()) as TaskBody;
  const prematureOpen = await request.post(`/api/tasks/${task.id}/open`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {},
  });
  expect(prematureOpen.status()).toBe(409);

  // Fund it properly, then open it for real.
  const fundResponse = await request.post(`/api/tasks/${task.id}/funding`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: { amount: 20, idempotency_key: `unpublish-test:${task.id}` },
  });
  expect(fundResponse.ok()).toBeTruthy();
  const openResponse = await request.post(`/api/tasks/${task.id}/open`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {},
  });
  expect(openResponse.ok()).toBeTruthy();

  await loginViaUi(page, owner.email);
  await expect(page.getByTestId("balance")).toBeVisible();
  await page.goto(`/#/tasks/${task.id}`);
  await expect(page.getByTestId("detail-title")).toBeVisible();

  // Unpublish moves it back to draft - the escape hatch for a task that
  // somehow reached "open" without matching funding (e.g. a backend that
  // didn't enforce that invariant): back to draft, the normal funding
  // panel is available again, and it can be reopened.
  await page.getByTestId("unpublish-task").click();
  await expect(page.getByTestId("task-action-message")).toContainText(
    "back to draft",
  );
  await expect(page.getByTestId("open-task")).toBeVisible();
  await expect(page.getByTestId("fund-task-panel")).toBeVisible();
  await page.getByTestId("open-task").click();
  await expect(page.getByTestId("task-action-message")).toContainText(
    "Task opened",
  );
  await expect(page.getByTestId("unpublish-task")).toBeVisible();
});

test("an owner cancels a no-reward task through the owner controls", async ({ page, request }) => {
  const owner = await registerViaApi(request, "cancel-owner");
  const title = `Cancellable ${crypto.randomUUID()}`;

  // A no-reward public task can be opened without funding.
  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: taskRequest(title, owner.body.subject_id, "public", 0),
  });
  expect(taskResponse.ok()).toBeTruthy();
  const task = (await taskResponse.json()) as TaskBody;
  await request.post(`/api/tasks/${task.id}/open`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {},
  });

  await openTaskFromDiscovery(page, owner.email, title);

  // A no-reward open task exposes Cancel (Refund is hidden because there is no escrow).
  await expect(page.getByTestId("cancel-task")).toBeVisible();
  await expect(page.getByTestId("refund-task")).toHaveCount(0);
  await page.getByTestId("cancel-task").click();
  await expect(page.getByTestId("task-action-message")).toContainText(
    "cancelled",
  );
});

test("an owner tips a collectible on accept through the review form", async ({ page, request }) => {
  const owner = await registerViaApi(request, "ctip-owner");
  const worker = await registerViaApi(request, "ctip-worker");
  const title = `Tip-collectible ${crypto.randomUUID()}`;

  // A transferable collectible the owner will tip (non-transferable is refused).
  const mintResponse = await request.post("/api/collectibles", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {
      name: "Gratitude token",
      kind: "badge",
      transfer_policy: "transferable_between_users",
    },
  });
  expect(mintResponse.ok()).toBeTruthy();
  const tip = (await mintResponse.json()) as { id: string };

  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: taskRequest(title, owner.body.subject_id, "public", 20),
  });
  const task = (await taskResponse.json()) as TaskBody;
  await request.post(`/api/tasks/${task.id}/funding`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: { amount: 20, idempotency_key: `fund:${task.id}` },
  });
  await request.post(`/api/tasks/${task.id}/open`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {},
  });

  // Worker submits.
  await openTaskFromDiscovery(page, worker.email, title);
  await fillDetailResponse(page, '{"answer":"done"}');
  await page.getByTestId("detail-submit").click();
  await expect(page.getByTestId("detail-submit-message")).toBeVisible();

  // Owner accepts with the collectible selected as a tip.
  await page.getByTestId("nav-account-menu").click();
  await logoutViaUi(page);
  await openTaskFromDiscovery(page, owner.email, title);
  await expect(page.getByTestId("submission-row")).toHaveCount(1);
  await page.getByTestId("review-tip-collectible").selectOption(tip.id);
  await page.getByTestId("accept-submission").click();
  await expect(page.getByTestId("review-message")).toContainText(
    "Review saved.",
  );

  // The tipped collectible left the owner's holdings.
  const holdings = await request.get("/api/collectibles", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
  });
  const owned = (await holdings.json()) as { collectibles: { id: string }[] };
  expect(owned.collectibles.find((c) => c.id === tip.id)).toBeUndefined();
});

test("a bundle task refunds credits and collectible in one shot via the UI", async ({ page, request }) => {
  const owner = await registerViaApi(request, "bundle-refund-owner");
  const title = `Bundle-refund ${crypto.randomUUID()}`;

  // A collectible to escrow as the collectible portion of the bundle reward.
  const mintResponse = await request.post("/api/collectibles", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {
      name: "Bundle medal",
      kind: "badge",
      transfer_policy: "non_transferable_except_payout",
    },
  });
  expect(mintResponse.ok()).toBeTruthy();
  const medal = (await mintResponse.json()) as { id: string };

  // Create a bundle task (credits + collectible), fund the credits, escrow the collectible, open.
  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {
      owner: {
        kind: "user",
        user_id: owner.body.subject_id,
        team_id: "",
        organization_id: "",
      },
      title,
      description: "A bundle reward task.",
      reward: { kind: "bundle", credit_amount: 20 },
      visibility: {
        kind: "public",
        user_id: "",
        team_id: "",
        organization_id: "",
      },
      placement: {
        kind: "standalone",
        series_id: "",
        series_title: "",
        series_position: 0,
      },
      response_schema_json: '{"kind":"freeform"}',
      payload: { kind: "none", json: "" },
    },
  });
  const task = (await taskResponse.json()) as TaskBody;
  await request.post(`/api/tasks/${task.id}/funding`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: { amount: 20, idempotency_key: `fund:${task.id}` },
  });
  await request.post(`/api/tasks/${task.id}/collectible-reward`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: { collectible_id: medal.id },
  });
  await request.post(`/api/tasks/${task.id}/open`, {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
    data: {},
  });

  await openTaskFromDiscovery(page, owner.email, title);

  // A bundle task shows the owner's unified "Reclaim reward" button (which calls
  // /refund and returns credits + collectible together) and NOT a separate
  // collectible reclaim.
  await expect(page.getByTestId("refund-task")).toHaveText("Reclaim reward");
  await expect(page.getByTestId("refund-collectible")).toHaveCount(0);
  await page.getByTestId("refund-task").click();
  await expect(page.getByTestId("task-action-message")).toContainText(
    "returned",
  );

  // Both reward portions returned: the refunded credits are back in the
  // spendable balance and nothing remains allocated, and the collectible is
  // back in holdings.
  const balance = await request.get("/api/credits/balance", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
  });
  const wallet = (await balance.json()) as {
    spendable_credits: number;
    allocated_credits: number;
  };
  expect(wallet.spendable_credits).toBe(100);
  expect(wallet.allocated_credits).toBe(0);
  const holdings = await request.get("/api/collectibles", {
    headers: { Authorization: `Bearer ${owner.body.access_token}` },
  });
  const owned = (await holdings.json()) as { collectibles: { id: string }[] };
  expect(owned.collectibles.find((c) => c.id === medal.id)).toBeTruthy();
});
