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

async function registerLoginAndCreateTask(
  page: import("@playwright/test").Page,
  request: import("@playwright/test").APIRequestContext,
  emailPrefix: string,
  title: string,
): Promise<TaskBody> {
  const email = uniqueEmail(emailPrefix);
  const registerResponse = await request.post("/api/auth/register", {
    data: { email, password },
  });
  expect(registerResponse.ok()).toBeTruthy();
  const registerBody = (await registerResponse.json()) as AuthBody;

  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${registerBody.access_token}` },
    data: taskRequest(title, registerBody.subject_id, "default"),
  });
  expect(taskResponse.ok()).toBeTruthy();
  const task = (await taskResponse.json()) as TaskBody;

  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  return task;
}

test("minting a collectible and awarding it to a task through the browser", async ({ page, request }) => {
  const task = await registerLoginAndCreateTask(
    page,
    request,
    "ui-collectible",
    "Collectible reward task",
  );

  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-collectibles").click();
  await page.getByTestId("collectibles-mint").click();
  const name = `Harvest badge ${crypto.randomUUID()}`;
  await page.getByTestId("collectible-name").fill(name);
  await page.getByTestId("mint-collectible").click();

  const row = page.getByTestId("collectible-row").filter({ hasText: name });
  await expect(row).toHaveCount(1);
  await expect(row).toContainText("minted");

  // The collectible has its own page reachable from its name link.
  await row.getByTestId("collectible-link").click();
  await expect(page).toHaveURL(/\/collectibles\/[0-9a-f-]+$/);
  await expect(page.getByTestId("collectible-detail-name")).toContainText(name);
  await page.getByTestId("back-collectibles").click();
  await expect(page).toHaveURL(/\/collectibles$/);

  const awardRow = page.getByTestId("collectible-row").filter({
    hasText: name,
  });
  await page.getByTestId("collectibles-award-task").click();
  await page.getByTestId("award-task-id").selectOption(task.id);
  await awardRow.getByTestId("award-collectible").click();

  await expect(page.getByTestId("award-message")).toBeVisible();
  await expect(
    page.getByTestId("collectible-row").filter({ hasText: name }),
  ).toContainText("escrowed");
});

test("awarding multiple collectibles shows the count on the task", async ({ page, request }) => {
  const title = `Multi collectible ${crypto.randomUUID()}`;
  const task = await registerLoginAndCreateTask(
    page,
    request,
    "ui-multi-collectible",
    title,
  );

  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-collectibles").click();
  await page.getByTestId("collectibles-mint").click();
  await page.getByTestId("collectibles-award-task").click();
  for (const label of ["First", "Second"]) {
    const name = `${label} medal ${crypto.randomUUID()}`;
    await page.getByTestId("collectible-name").fill(name);
    await page.getByTestId("mint-collectible").click();
    const row = page.getByTestId("collectible-row").filter({ hasText: name });
    await expect(row).toHaveCount(1);
    await page.getByTestId("award-task-id").selectOption(task.id);
    await row.getByTestId("award-collectible").click();
    await expect(page.getByTestId("award-message")).toBeVisible();
  }

  // The task row on the Tasks page now reflects both escrowed collectibles.
  await page.getByTestId("nav-tasks").click();
  await expect(
    page.getByTestId("task-row").filter({ hasText: title }),
  ).toContainText("2 collectibles");
});

test("a creator adds a collectible to a task's reward from the task detail page", async ({ page, request }) => {
  const task = await registerLoginAndCreateTask(
    page,
    request,
    "ui-detail-award",
    `Detail-page award ${crypto.randomUUID()}`,
  );

  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-collectibles").click();
  await page.getByTestId("collectibles-mint").click();
  const name = `Detail-page medal ${crypto.randomUUID()}`;
  await page.getByTestId("collectible-name").fill(name);
  await page.getByTestId("mint-collectible").click();
  await expect(page.getByTestId("collectible-row").filter({ hasText: name }))
    .toHaveCount(1);

  await page.goto(`/#/tasks/${task.id}`);
  await page.getByTestId("add-collectible-reward-panel").click();
  await page
    .getByTestId("collectible-row")
    .filter({ hasText: name })
    .getByTestId("award-collectible")
    .click();
  await expect(page.getByTestId("award-message")).toBeVisible();

  // The task's reward kind transitioned from none to collectible, so the
  // owner controls now offer the collectible-specific refund action.
  await expect(page.getByTestId("refund-collectible")).toBeVisible();
});
