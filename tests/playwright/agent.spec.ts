import { expect, test } from "@playwright/test";
import {
  type AuthBody,
  password,
  taskRequest,
  uniqueEmail,
} from "./helpers.ts";

test("creating an agent credential shows the token and MCP config", async ({ page }) => {
  await page.goto("/");
  await page.getByTestId("email").fill(uniqueEmail("ui-agent"));
  await page.getByTestId("password").fill(password);
  await page.getByTestId("register").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await page.getByTestId("agent-label").fill("Local workstation agent");
  await page.getByTestId("create-agent").click();

  await expect(page.getByTestId("agent-secret")).toContainText("scrop_agent_");
  await expect(page.getByTestId("mcp-config")).toContainText("mcpServers");
  await expect(page.getByTestId("mcp-config")).toContainText("/mcp");
  await expect(page.getByTestId("credential-row")).toContainText(
    "Local workstation agent",
  );
});

test("tasks panel lists user tasks and shows agent curl examples", async ({ page, request }) => {
  const email = uniqueEmail("ui-tasks");

  const registerResponse = await request.post("/api/auth/register", {
    data: { email, password },
  });
  expect(registerResponse.ok()).toBeTruthy();
  const registerBody = (await registerResponse.json()) as AuthBody;

  const taskResponse = await request.post("/api/tasks", {
    headers: { Authorization: `Bearer ${registerBody.access_token}` },
    data: taskRequest(
      "Agent task from the browser",
      registerBody.subject_id,
      "default",
    ),
  });
  expect(taskResponse.ok()).toBeTruthy();

  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await expect(page.getByTestId("task-row")).toContainText(
    "Agent task from the browser",
  );
  await page.getByTestId("view-task").click();
  await expect(page.getByTestId("task-mcp-submit")).toContainText("/mcp");
  await expect(page.getByTestId("task-mcp-submit")).toContainText(
    "sharecrop.submit_response",
  );
});
