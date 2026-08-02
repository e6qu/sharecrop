import {
  type APIRequestContext,
  expect,
  type Page,
  test,
} from "@playwright/test";
import {
  type AuthBody,
  password,
  uniqueEmail,
  verifyAccountByApi,
  verifyAccountWithToken,
} from "./helpers.ts";

interface TaskBody {
  id: string;
}

interface CredentialCreatedBody {
  credential: { id: string };
  secret: string;
}

// registerVerified registers an account over the API and completes email
// verification, so its signup grant has landed and it can fund work.
async function registerVerified(
  request: APIRequestContext,
  prefix: string,
): Promise<{ email: string; body: AuthBody }> {
  const email = uniqueEmail(prefix);
  const response = await request.post("/api/auth/register", {
    data: { email, password },
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  const body = (await response.json()) as AuthBody;
  await verifyAccountWithToken(request, body.access_token);
  return { email, body };
}

// openReservableTask publishes a public, reward-carrying task that requires a
// reservation, which is the shape a work-seeking agent takes on.
async function openReservableTask(
  request: APIRequestContext,
  owner: AuthBody,
  title: string,
): Promise<TaskBody> {
  const headers = { Authorization: `Bearer ${owner.access_token}` };
  const created = await request.post("/api/tasks", {
    headers,
    data: {
      owner: {
        kind: "user",
        user_id: owner.subject_id,
        team_id: "",
        organization_id: "",
      },
      title,
      description: "Work an agent may take on within its budget.",
      task_type: "research",
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
      reward: { kind: "credit", credit_amount: 15 },
      response_schema_json: '{"kind":"freeform"}',
      payload: { kind: "none", json: "" },
    },
  });
  expect(created.ok(), await created.text()).toBeTruthy();
  const task = (await created.json()) as TaskBody;
  const funded = await request.post(`/api/tasks/${task.id}/funding`, {
    headers,
    data: { amount: 15, idempotency_key: `fund-${task.id}` },
  });
  expect(funded.ok(), await funded.text()).toBeTruthy();
  const opened = await request.post(`/api/tasks/${task.id}/open`, {
    headers,
    data: {},
  });
  expect(opened.ok(), await opened.text()).toBeTruthy();
  return task;
}

// signIn logs an already-registered account into the browser session.
async function signIn(page: Page, email: string): Promise<void> {
  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toBeVisible();
}

async function openAgentsPage(page: Page): Promise<void> {
  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-agents").click();
}

test("a new credential is not allowed to seek work until its human says so", async ({ page, request }) => {
  const email = uniqueEmail("work-budget-default");
  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("register").click();
  // Wait for the session to exist before verifying over the API: the API
  // login below needs the registration to have committed.
  await expect(page.getByTestId("balance")).toHaveText("0 credits");
  await verifyAccountByApi(request, email);
  await page.reload();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await openAgentsPage(page);
  await page.getByTestId("agent-label").fill("Kitchen table agent");
  await page.getByTestId("create-agent").click();
  await expect(page.getByTestId("agent-secret")).toContainText("scrop_agent_");

  // The resting state is a plain statement, not an error: the agent does the
  // work it is handed and nothing else.
  await expect(page.getByTestId("work-seeking-state")).toHaveText(
    "Not allowed to seek work",
  );
  await expect(page.getByTestId("work-policy-disabled")).toBeVisible();
  await expect(page.getByTestId("work-policy-enabled")).toHaveCount(0);
  // No consumption meters are shown for an agent that cannot consume.
  await expect(page.getByTestId("work-policy-usage")).toHaveCount(0);
});

test("allowing an agent to seek work records its daily cap and allowances", async ({ page, request }) => {
  const email = uniqueEmail("work-budget-enable");
  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("register").click();
  // Wait for the session to exist before verifying over the API: the API
  // login below needs the registration to have committed.
  await expect(page.getByTestId("balance")).toHaveText("0 credits");
  await verifyAccountByApi(request, email);
  await page.reload();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await openAgentsPage(page);
  await page.getByTestId("agent-label").fill("Overnight worker");
  await page.getByTestId("create-agent").click();
  await expect(page.getByTestId("agent-secret")).toContainText("scrop_agent_");

  await page.getByTestId("allow-work-seeking").click();
  // The daily task budget is prefilled with a sensible starting number.
  await expect(page.getByTestId("work-policy-tasks-per-day")).toHaveValue("10");
  // The token budget is labelled as advisory wherever it is offered.
  await expect(page.getByTestId("work-policy-token-advisory")).toContainText(
    "Sharecrop does not enforce this",
  );

  await page.getByTestId("work-policy-tasks-per-day").fill("4");
  await page.getByTestId("work-policy-max-concurrent").fill("2");
  await page.getByTestId("work-policy-max-credits").fill("60");
  await page.getByTestId("work-policy-min-reward").fill("5");
  await page.getByTestId("work-policy-type-research").check();
  await page.getByTestId("work-policy-type-data_extraction").check();
  await page.getByTestId("work-policy-token-budget-tokens").fill("250000");
  await page.getByTestId("work-policy-token-note").fill("Pace the day.");
  await page.getByTestId("save-work-policy").click();

  await expect(page.getByTestId("work-policy-message")).toContainText(
    "Overnight worker may seek work, up to 4 tasks a day.",
  );
  await expect(page.getByTestId("work-seeking-state")).toHaveText(
    "Allowed to seek work",
  );
  await expect(page.getByTestId("work-tasks-today")).toContainText(
    "0 of 4 tasks today",
  );
  await expect(page.getByTestId("work-reservations-now")).toContainText(
    "0 of 2 tasks held at once",
  );
  await expect(page.getByTestId("work-credits-today")).toContainText(
    "0 of 60 credits spent today",
  );
  await expect(page.getByTestId("work-policy-task-types")).toContainText(
    "Research, Data extraction",
  );
  await expect(page.getByTestId("work-policy-min-reward")).toContainText(
    "At least 5 credits per task",
  );
  await expect(page.getByTestId("work-policy-token-budget")).toContainText(
    "250000 tokens a day — advisory only",
  );
  await expect(page.getByTestId("work-policy-token-budget")).toContainText(
    "Pace the day.",
  );

  // Stopping puts the credential back to the default: no allowances at all.
  await page.getByTestId("stop-work-seeking").click();
  await expect(page.getByTestId("work-policy-message")).toContainText(
    "Overnight worker has stopped seeking work.",
  );
  await expect(page.getByTestId("work-seeking-state")).toHaveText(
    "Not allowed to seek work",
  );
});

test("a daily task cap below one is refused instead of sent as no limit", async ({ page, request }) => {
  const email = uniqueEmail("work-budget-invalid");
  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("register").click();
  // Wait for the session to exist before verifying over the API: the API
  // login below needs the registration to have committed.
  await expect(page.getByTestId("balance")).toHaveText("0 credits");
  await verifyAccountByApi(request, email);
  await page.reload();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");

  await openAgentsPage(page);
  await page.getByTestId("agent-label").fill("Careless setup");
  await page.getByTestId("create-agent").click();
  await expect(page.getByTestId("agent-secret")).toContainText("scrop_agent_");

  await page.getByTestId("allow-work-seeking").click();
  await page.getByTestId("work-policy-tasks-per-day").fill("0");
  await page.getByTestId("save-work-policy").click();
  await expect(page.getByTestId("work-policy-form-message")).toContainText(
    "at least 1",
  );
  await expect(page.getByTestId("work-policy-form")).toBeVisible();
});

test("an agent's consumption shows against its allowances after it takes work", async ({ page, request }) => {
  const owner = await registerVerified(request, "work-budget-owner");
  const worker = await registerVerified(request, "work-budget-worker");
  const task = await openReservableTask(
    request,
    owner.body,
    "Summarise the field notes for the week",
  );

  const credentialResponse = await request.post("/api/agent-credentials", {
    headers: { Authorization: `Bearer ${worker.body.access_token}` },
    data: {
      label: "Field research agent",
      scopes: ["tasks_read", "submissions_write"],
      expires_at: "",
    },
  });
  expect(credentialResponse.ok(), await credentialResponse.text())
    .toBeTruthy();
  const credential = (await credentialResponse.json()) as CredentialCreatedBody;

  const policyResponse = await request.put(
    `/api/agent-credentials/${credential.credential.id}/work-policy`,
    {
      headers: { Authorization: `Bearer ${worker.body.access_token}` },
      data: {
        work_seeking: "work_seeking_enabled",
        max_tasks_per_day: 3,
        max_concurrent_reservations: 2,
        max_credits_per_day: 0,
        task_types: [],
        min_reward_credits: 0,
        token_budget_tokens: 0,
        token_budget_note: "",
      },
    },
  );
  expect(policyResponse.ok(), await policyResponse.text()).toBeTruthy();

  // The agent takes the task on its own credential, which is what charges
  // the day counter and the concurrency allowance.
  const reserved = await request.post(`/api/tasks/${task.id}/reservations`, {
    headers: { Authorization: `Bearer ${credential.secret}` },
    data: {},
  });
  expect(reserved.ok(), await reserved.text()).toBeTruthy();

  await signIn(page, worker.email);
  await openAgentsPage(page);
  await expect(page.getByTestId("work-tasks-today")).toContainText(
    "1 of 3 tasks today",
  );
  await expect(page.getByTestId("work-reservations-now")).toContainText(
    "1 of 2 tasks held at once",
  );
  // Without a daily credit cap the number is stated as a fact, not as a
  // meter with no ceiling.
  await expect(page.getByTestId("work-credits-today")).toContainText(
    "0 credits spent today (no daily cap set)",
  );
});

test("the create-task form offers every task type, grouped", async ({ page, request }) => {
  const account = await registerVerified(request, "task-types");
  await signIn(page, account.email);

  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();

  const selector = page.getByTestId("create-task-type");
  await expect(selector.locator("option")).toHaveCount(16);
  await expect(selector.locator("optgroup")).toHaveCount(4);
  for (
    const taskType of [
      "research",
      "documentation_writing",
      "planning",
      "troubleshooting",
      "data_extraction",
      "threat_analysis",
      "architecture_review",
      "code_analysis",
      "diagram_writing",
      "document_review",
    ]
  ) {
    await expect(selector.locator(`option[value="${taskType}"]`)).toHaveCount(
      1,
    );
  }

  // The new high-value types carry starter templates, like the review types.
  await selector.selectOption("research");
  await expect(page.getByTestId("create-description")).toHaveValue(
    /Research the question stated above/,
  );
  await selector.selectOption("planning");
  await expect(page.getByTestId("create-description")).toHaveValue(
    /Break it into ordered steps/,
  );

  // The discovery filter offers the same full list plus "All types".
  await page.getByTestId("nav-tasks").click();
  await expect(page.getByTestId("tasks-type").locator("option")).toHaveCount(
    17,
  );
});

test("an unverified account is told why its balance is zero", async ({ page }) => {
  const email = uniqueEmail("verify-hint");
  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("register").click();

  await expect(page.getByTestId("balance")).toHaveText("0 credits");
  await expect(page.getByTestId("verify-email-hint")).toContainText(
    "Verify your email to receive your 100-credit signup grant.",
  );
  await expect(page.getByTestId("verify-email-hint-link")).toBeVisible();
});

test("a verified account is not told to verify anything", async ({ page, request }) => {
  const account = await registerVerified(request, "verify-hint-done");
  await signIn(page, account.email);

  await expect(page.getByTestId("balance")).toHaveText("100 credits");
  await expect(page.getByTestId("verify-email-hint")).toHaveCount(0);
});

test("an agent that has used up its daily allowance says so", async ({ page, request }) => {
  const owner = await registerVerified(request, "work-budget-cap-owner");
  const worker = await registerVerified(request, "work-budget-cap-worker");
  const task = await openReservableTask(
    request,
    owner.body,
    "Read the soil survey and pull out the pH figures",
  );

  const credentialResponse = await request.post("/api/agent-credentials", {
    headers: { Authorization: `Bearer ${worker.body.access_token}` },
    data: {
      label: "One task a day agent",
      scopes: ["tasks_read", "submissions_write"],
      expires_at: "",
    },
  });
  expect(credentialResponse.ok(), await credentialResponse.text()).toBeTruthy();
  const credential = (await credentialResponse.json()) as CredentialCreatedBody;

  const policyResponse = await request.put(
    `/api/agent-credentials/${credential.credential.id}/work-policy`,
    {
      headers: { Authorization: `Bearer ${worker.body.access_token}` },
      data: {
        work_seeking: "work_seeking_enabled",
        max_tasks_per_day: 1,
        max_concurrent_reservations: 1,
        max_credits_per_day: 0,
        task_types: [],
        min_reward_credits: 0,
        token_budget_tokens: 0,
        token_budget_note: "",
      },
    },
  );
  expect(policyResponse.ok(), await policyResponse.text()).toBeTruthy();

  const reserved = await request.post(`/api/tasks/${task.id}/reservations`, {
    headers: { Authorization: `Bearer ${credential.secret}` },
    data: {},
  });
  expect(reserved.ok(), await reserved.text()).toBeTruthy();

  await signIn(page, worker.email);
  await openAgentsPage(page);
  // Being out of allowance is stated in words next to the meter, not left to
  // the length of a coloured bar.
  await expect(page.getByTestId("work-tasks-today")).toContainText(
    "1 of 1 tasks today",
  );
  await expect(page.getByTestId("work-tasks-today")).toContainText(
    "cap reached",
  );
  await expect(page.getByTestId("work-reservations-now")).toContainText(
    "cap reached",
  );
});
