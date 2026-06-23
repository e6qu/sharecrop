import { expect, test } from "@playwright/test";
import {
  type AuthBody,
  password,
  taskRequest,
  uniqueEmail,
} from "./helpers.ts";

interface TaskBody {
  id: string;
}

async function registerViaApi(
  request: {
    post: (
      url: string,
      opts: { data: unknown },
    ) => Promise<{ ok: () => boolean; json: () => Promise<unknown> }>;
  },
  prefix: string,
): Promise<{ email: string; body: AuthBody }> {
  const email = uniqueEmail(prefix);
  const response = await request.post("/api/auth/register", {
    data: { email, password },
  });
  expect(response.ok()).toBeTruthy();
  return { email, body: (await response.json()) as AuthBody };
}

async function loginViaUi(
  page: {
    goto: (u: string) => Promise<unknown>;
    getByTestId: (
      id: string,
    ) => { fill: (v: string) => Promise<void>; click: () => Promise<void> };
  },
  email: string,
): Promise<void> {
  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
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

  await page.getByTestId("nav-discovery").click();
  const workerRow = page.getByTestId("discovery-task-row").filter({
    hasText: title,
  });
  await expect(workerRow).toHaveCount(1);
  await workerRow.getByTestId("discovery-view").click();
  await expect(page.getByTestId("detail-title")).toContainText(title);

  await page.getByTestId("detail-submit-input").fill(
    '{"answer":"from the browser"}',
  );
  await page.getByTestId("detail-submit").click();
  await expect(page.getByTestId("detail-submit-message")).toBeVisible();

  // Owner reviews and accepts the submission through the UI.
  await page.getByTestId("nav-dashboard").click();
  await page.getByTestId("logout").click();
  await loginViaUi(page, owner.email);
  await expect(page.getByTestId("balance")).toHaveText("80 credits");

  await page.getByTestId("nav-discovery").click();
  const ownerRow = page.getByTestId("discovery-task-row").filter({
    hasText: title,
  });
  await ownerRow.getByTestId("discovery-view").click();
  await expect(page.getByTestId("submission-row")).toHaveCount(1);
  await page.getByTestId("accept-submission").click();
  await expect(page.getByTestId("accept-submission")).toHaveCount(0);
});

test("requesters configure reservations and workers include reserved tasks", async ({ page, request }) => {
  const owner = await registerViaApi(request, "reservation-ui-owner");
  const title = `Reserved UI ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill(
    "Reservation required from the browser.",
  );
  await page.getByTestId("create-participation-reservation_required").click();
  await page.getByTestId("create-visibility-public").click();
  await page.getByTestId("create-task").click();
  await expect(page.getByTestId("create-message")).toContainText(
    "Created task",
  );

  const ownerRow = page.getByTestId("task-row").filter({ hasText: title });
  await expect(ownerRow).toHaveCount(1);
  await ownerRow.getByTestId("view-task").click();
  await expect(page.getByTestId("task-instructions")).toContainText(
    "sharecrop.get_task_schema",
  );
  await page.getByTestId("open-task").click();
  await expect(page.getByTestId("create-message")).toContainText("Task opened");

  const worker = await registerViaApi(request, "reservation-ui-worker");
  await page.getByTestId("logout").click();
  await loginViaUi(page, worker.email);
  await page.getByTestId("nav-discovery").click();
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

  const other = await registerViaApi(request, "reservation-ui-other");
  await page.getByTestId("detail-back").click();
  await page.getByTestId("nav-dashboard").click();
  await page.getByTestId("logout").click();
  await loginViaUi(page, other.email);
  await page.getByTestId("nav-discovery").click();
  await expect(
    page.getByTestId("discovery-task-row").filter({ hasText: title }),
  ).toHaveCount(0);
  await page.getByTestId("include-reserved").click();
  await expect(
    page.getByTestId("discovery-task-row").filter({ hasText: title }),
  ).toHaveCount(1);
});

test("users create and see their organizations", async ({ page, request }) => {
  const owner = await registerViaApi(request, "org-ui-owner");
  const name = `Org UI ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
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
  await page.getByTestId("create-org-name").fill(orgName);
  await page.getByTestId("create-org").click();
  await expect(page.getByTestId("org-message")).toContainText(
    "Created organization",
  );

  await page.getByTestId("select-organization").first().click();
  await expect(page.getByTestId("active-organization")).toBeVisible();

  await page.getByTestId("create-org-team-name").fill(teamName);
  await page.getByTestId("create-org-team").click();
  await expect(
    page.getByTestId("org-team-row").filter({ hasText: teamName }),
  ).toHaveCount(1);

  await page.getByTestId("provision-member-email").fill(member.email);
  await page.getByTestId("provision-member").click();
  await expect(page.getByTestId("provision-member-message")).toContainText(
    "provisioned",
  );
});

test("requesters filter their task list by state", async ({ page, request }) => {
  const owner = await registerViaApi(request, "filter-ui-owner");
  const title = `Filter UI ${crypto.randomUUID()}`;

  await loginViaUi(page, owner.email);
  await page.getByTestId("create-title").fill(title);
  await page.getByTestId("create-description").fill("Filter from the browser.");
  await page.getByTestId("create-task").click();
  await expect(page.getByTestId("create-message")).toContainText(
    "Created task",
  );

  const row = page.getByTestId("task-row").filter({ hasText: title });
  await expect(row).toHaveCount(1);

  // The new task is a draft, so filtering to Open hides it and Draft shows it.
  await page.getByTestId("task-filter-open").click();
  await expect(page.getByTestId("task-row").filter({ hasText: title }))
    .toHaveCount(0);
  await page.getByTestId("task-filter-draft").click();
  await expect(page.getByTestId("task-row").filter({ hasText: title }))
    .toHaveCount(1);
});
