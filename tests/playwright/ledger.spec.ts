import { expect, test } from "@playwright/test";

interface AuthBody {
  access_token: string;
  subject_id: string;
}

interface TaskBody {
  id: string;
}

const password = "correct horse battery staple";

function uniqueEmail(prefix: string): string {
  return `${prefix}-${crypto.randomUUID()}@example.com`;
}

function userTaskRequest(userId: string): Record<string, unknown> {
  return {
    owner: { kind: "user", user_id: userId, team_id: "", organization_id: "" },
    title: "Fund from the browser",
    description: "A task funded through the browser interface.",
    visibility: {
      kind: "default",
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
  };
}

test("registering shows the signup grant balance and ledger entry", async ({ page }) => {
  await page.goto("/");
  await page.getByTestId("email").fill(uniqueEmail("ui-signup"));
  await page.getByTestId("password").fill(password);
  await page.getByTestId("register").click();

  await expect(page.getByTestId("balance")).toHaveText("100 credits");
  await expect(page.getByTestId("ledger")).toContainText("signup_grant");
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
    data: userTaskRequest(registerBody.subject_id),
  });
  expect(taskResponse.ok()).toBeTruthy();
  const taskBody = (await taskResponse.json()) as TaskBody;

  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await page.getByTestId("fund-task-id").fill(taskBody.id);
  await page.getByTestId("fund-amount").fill("40");
  await page.getByTestId("fund").click();

  await expect(page.getByTestId("fund-message")).toContainText(
    "Escrowed 40 credits",
  );
  await expect(page.getByTestId("balance")).toHaveText("60 credits");
});
