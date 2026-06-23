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

test("minting a collectible and awarding it to a task through the browser", async ({ page, request }) => {
  const email = uniqueEmail("ui-collectible");

  const registerResponse = await request.post("/api/auth/register", {
    data: { email, password },
  });
  expect(registerResponse.ok()).toBeTruthy();
  const registerBody = (await registerResponse.json()) as AuthBody;

  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${registerBody.access_token}` },
    data: taskRequest(
      "Collectible reward task",
      registerBody.subject_id,
      "default",
    ),
  });
  expect(taskResponse.ok()).toBeTruthy();
  const task = (await taskResponse.json()) as TaskBody;

  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await page.getByTestId("nav-collectibles").click();
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
  await page.getByTestId("award-task-id").selectOption(task.id);
  await awardRow.getByTestId("award-collectible").click();

  await expect(page.getByTestId("award-message")).toBeVisible();
  await expect(
    page.getByTestId("collectible-row").filter({ hasText: name }),
  ).toContainText("escrowed");
});

test("awarding multiple collectibles shows the count on the task", async ({ page, request }) => {
  const email = uniqueEmail("ui-multi-collectible");
  const registerResponse = await request.post("/api/auth/register", {
    data: { email, password },
  });
  expect(registerResponse.ok()).toBeTruthy();
  const registerBody = (await registerResponse.json()) as AuthBody;

  const title = `Multi collectible ${crypto.randomUUID()}`;
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

  await page.getByTestId("nav-collectibles").click();
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
