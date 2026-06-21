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
    data: taskRequest(title, owner.body.subject_id, "public"),
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
