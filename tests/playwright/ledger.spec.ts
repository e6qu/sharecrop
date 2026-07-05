import { expect, test } from "@playwright/test";
import {
  type AuthBody,
  password,
  type TaskBody,
  taskRequest,
  uniqueEmail,
} from "./helpers.ts";

test("registering shows the signup grant balance and ledger entry", async ({ page }) => {
  await page.goto("/");
  await page.getByTestId("email").fill(uniqueEmail("ui-signup"));
  await page.getByTestId("password").fill(password);
  await page.getByTestId("register").click();

  await expect(page.getByTestId("balance")).toHaveText("100 credits");
  await expect(page.getByTestId("ledger")).toContainText("Signup grant");
});

test("funding a task escrows credits and lowers the balance", async ({ page, request }) => {
  const email = uniqueEmail("ui-fund");

  const registerResponse = await request.post("/api/auth/register", {
    data: { email, password },
  });
  expect(registerResponse.ok()).toBeTruthy();
  const registerBody = (await registerResponse.json()) as AuthBody;

  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${registerBody.access_token}` },
    data: taskRequest(
      "Fund from the browser",
      registerBody.subject_id,
      "default",
      40,
    ),
  });
  expect(taskResponse.ok()).toBeTruthy();
  const taskBody = (await taskResponse.json()) as TaskBody;

  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-funding").click();
  await page.getByTestId("fund-task-id").selectOption(taskBody.id);
  await page.getByTestId("fund-amount").fill("40");
  await page.getByTestId("fund").click();

  await expect(page.getByTestId("fund-message")).toContainText(
    "Escrowed 40 credits",
  );

  await page.getByTestId("nav-overview").click();
  await expect(page.getByTestId("balance")).toHaveText("60 credits");
});

test("the fund panel does not appear on an already-funded, open task", async ({ page, request }) => {
  const email = uniqueEmail("ui-fund-open");

  const registerResponse = await request.post("/api/auth/register", {
    data: { email, password },
  });
  expect(registerResponse.ok()).toBeTruthy();
  const registerBody = (await registerResponse.json()) as AuthBody;

  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${registerBody.access_token}` },
    data: taskRequest(
      "Fund from task list, already open",
      registerBody.subject_id,
      "default",
      40,
    ),
  });
  expect(taskResponse.ok()).toBeTruthy();
  const taskBody = (await taskResponse.json()) as TaskBody;

  const preFundResponse = await request.post(
    `/api/tasks/${taskBody.id}/funding`,
    {
      headers: { Authorization: `Bearer ${registerBody.access_token}` },
      data: {
        amount: 40,
        idempotency_key: `pre-fund-setup-${taskBody.id}`,
        organization_id: "",
      },
    },
  );
  expect(preFundResponse.ok()).toBeTruthy();

  const openResponse = await request.post(`/api/tasks/${taskBody.id}/open`, {
    headers: { Authorization: `Bearer ${registerBody.access_token}` },
    data: {},
  });
  expect(openResponse.ok()).toBeTruthy();

  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toHaveText("60 credits");

  await page.goto(`/#/tasks/${taskBody.id}`);
  await expect(page.getByTestId("detail-title")).toBeVisible();
  await expect(page.getByTestId("fund-task-panel")).toHaveCount(0);
});
