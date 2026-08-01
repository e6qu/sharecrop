import {
  type APIRequestContext,
  expect,
  type Page,
  test,
} from "@playwright/test";
import { password, taskRequest, uniqueEmail } from "./helpers.ts";

// Browser coverage for the agent-loop UI items: forced sign-out on an
// unauthenticated response, the persistent overview feed cursor, superseded
// submissions and dispute filing, pager totals, and funded/holder facts on
// task rows. The admin-queue half of the dispute flow lives in demo.spec.ts,
// where the signed-in demo account is a platform admin.

interface Account {
  email: string;
  displayName: string;
  accessToken: string;
  subjectId: string;
}

async function signUp(
  request: APIRequestContext,
  prefix: string,
  displayName: string,
): Promise<Account> {
  const email = uniqueEmail(prefix);
  const response = await request.post("/api/auth/register", {
    data: { email, password, display_name: displayName },
  });
  const body = await response.text();
  expect(response.ok(), `register ${email}: ${body}`).toBeTruthy();
  const parsed = JSON.parse(body) as {
    access_token: string;
    subject_id: string;
  };
  return {
    email,
    displayName,
    accessToken: parsed.access_token,
    subjectId: parsed.subject_id,
  };
}

async function signIn(page: Page, email: string): Promise<void> {
  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toBeVisible({ timeout: 15000 });
}

function bearer(account: Account): { Authorization: string } {
  return { Authorization: `Bearer ${account.accessToken}` };
}

async function postAsUser(
  request: APIRequestContext,
  account: Account,
  path: string,
  data: Record<string, unknown>,
): Promise<string> {
  const response = await request.post(path, { headers: bearer(account), data });
  const body = await response.text();
  expect(response.ok(), `POST ${path}: ${body}`).toBeTruthy();
  return body;
}

// Creates a funded, open, public credit-reward task owned by the owner.
async function fundedOpenTask(
  request: APIRequestContext,
  owner: Account,
  title: string,
  amount: number,
  participationPolicy: string,
): Promise<string> {
  const creation = {
    ...taskRequest(title, owner.subjectId, "public", amount),
    participation: {
      policy: participationPolicy,
      assignee_scope: "user",
      reservation_expiry_hours: 48,
    },
  };
  const created = JSON.parse(
    await postAsUser(request, owner, "/api/tasks", creation),
  ) as { id: string };
  await postAsUser(request, owner, `/api/tasks/${created.id}/funding`, {
    amount,
    idempotency_key: `fund:${created.id}`,
  });
  await postAsUser(request, owner, `/api/tasks/${created.id}/open`, {});
  return created.id;
}

async function submitAnswer(
  request: APIRequestContext,
  worker: Account,
  taskId: string,
  answer: string,
): Promise<string> {
  const body = JSON.parse(
    await postAsUser(request, worker, `/api/tasks/${taskId}/submissions`, {
      response_json: JSON.stringify({ answer }),
    }),
  ) as { submission: { id: string } };
  return body.submission.id;
}

test("an unauthenticated API response mid-session forces sign-out with a notice", async ({ page, request }) => {
  const user = await signUp(request, "session-end", "Session Ender");
  await signIn(page, user.email);

  // Simulate the token dying mid-session: the next task-list request comes
  // back with the unauthenticated error code.
  await page.route("**/api/tasks**", (route) =>
    route.fulfill({
      status: 401,
      contentType: "application/json",
      body: '{"code":"unauthenticated","error":"access token is invalid"}',
    }));
  await page.getByTestId("nav-tasks").click();

  // The session is cleared centrally: the auth screen appears with the
  // factual notice, without a hand-patched handler on the tasks page.
  await expect(page.getByTestId("session-ended-notice")).toHaveText(
    "Your session ended. Sign in again.",
  );
  await expect(page.getByTestId("email")).toBeVisible();
  await expect(page.getByTestId("nav-tasks")).toHaveCount(0);
});

test("the overview activity feed resumes from its cursor on revisit", async ({ page, request }) => {
  const owner = await signUp(request, "feed-owner", "Feed Owner");
  const worker = await signUp(request, "feed-worker", "Feed Worker");
  const title = `Feed cursor ${crypto.randomUUID()}`;
  const taskId = await fundedOpenTask(request, owner, title, 20, "open");
  await submitAnswer(request, worker, taskId, "first");

  // First visit reads the stream from the start and shows the submission
  // event (funding and opening the task emitted rows of their own too).
  await signIn(page, owner.email);
  const submissionRows = page
    .getByTestId("activity-row")
    .filter({ hasText: title })
    .filter({ hasText: "submitted a response to" });
  await expect(submissionRows).toHaveCount(1);

  // Leave the feed, let a second event happen elsewhere.
  await page.getByTestId("nav-tasks").click();
  await expect(page.getByTestId("tasks")).toBeVisible();
  await submitAnswer(request, worker, taskId, "second");

  // The revisit resumes after the cursor (the request carries after=) and
  // appends the new event without duplicating the old one.
  const resumed = page.waitForRequest((candidate) =>
    candidate.url().includes("/api/events?") &&
    candidate.url().includes("after=")
  );
  await page.getByTestId("nav-overview").click();
  await resumed;
  await expect(submissionRows).toHaveCount(2);
});

test("a competing accept marks the losing submission superseded in the UI", async ({ page, request }) => {
  const owner = await signUp(request, "supersede-owner", "Supersede Owner");
  const winner = await signUp(request, "supersede-winner", "Winning Worker");
  const loser = await signUp(request, "supersede-loser", "Losing Worker");
  const title = `Superseded ${crypto.randomUUID()}`;
  const taskId = await fundedOpenTask(request, owner, title, 20, "open");
  const winningSubmission = await submitAnswer(
    request,
    winner,
    taskId,
    "the accepted answer",
  );
  await submitAnswer(request, loser, taskId, "the competing answer");
  await postAsUser(
    request,
    owner,
    `/api/tasks/${taskId}/submissions/${winningSubmission}/accept`,
    { idempotency_key: `accept:${winningSubmission}` },
  );

  // The losing worker's own submission row shows the neutral superseded
  // state with a one-line explanation.
  await signIn(page, loser.email);
  await page.goto(`/#/tasks/${taskId}`);
  const row = page.getByTestId("my-submission-row");
  await expect(row).toHaveCount(1);
  await expect(row).toContainText("superseded");
  await expect(row.getByTestId("superseded-note")).toHaveText(
    "Another submission was accepted.",
  );
});

test("a worker files a dispute from their rejected submission", async ({ page, request }) => {
  const owner = await signUp(request, "dispute-owner", "Dispute Owner");
  const worker = await signUp(request, "dispute-worker", "Dispute Worker");
  const title = `Dispute ${crypto.randomUUID()}`;
  const taskId = await fundedOpenTask(request, owner, title, 20, "open");
  const submissionId = await submitAnswer(request, worker, taskId, "rejected");
  await postAsUser(
    request,
    owner,
    `/api/tasks/${taskId}/submissions/${submissionId}/reject`,
    {
      idempotency_key: `reject:${submissionId}`,
      review_note: "Does not follow the brief.",
      ban_selection: "none",
    },
  );

  await signIn(page, worker.email);
  await page.goto(`/#/tasks/${taskId}`);
  const row = page.getByTestId("my-submission-row");
  await expect(row).toContainText("rejected");

  // "File a dispute" opens the report form preset to reason=dispute with the
  // submission as its subject.
  await row.getByTestId("file-dispute").click();
  await expect(page.getByTestId("moderation-subject")).toContainText(
    `disputes the review of submission ${submissionId}`,
  );
  await page.getByTestId("moderation-details").fill(
    "The response matches the brief; the rejection is not justified.",
  );
  await page.getByTestId("report-task").click();
  await expect(page.getByTestId("moderation-message")).toContainText(
    "Report submitted: dispute",
  );
});

test("pagers show the total page count where the response carries one", async ({ page, request }) => {
  const user = await signUp(request, "pager-total", "Pager Totals");
  await signIn(page, user.email);

  // The signup grant is the one ledger entry, so the counted pager reads
  // "Page 1 of 1" instead of a bare "Page 1".
  await expect(page.getByTestId("ledger-page-offset")).toHaveText(
    "Page 1 of 1",
  );
  await page.getByTestId("nav-tasks").click();
  await expect(page.getByTestId("tasks-page-offset")).toHaveText(
    "Page 1 of 1",
  );
  await expect(page.getByTestId("discovery-page-offset")).toContainText(
    "Page 1 of",
  );
  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-inbox").click();
  await expect(page.getByTestId("inbox-page-offset")).toHaveText(
    "Page 1 of 1",
  );
});

test("task rows show funded state and the reservation holder's name", async ({ page, request }) => {
  const owner = await signUp(request, "funded-owner", "Funded Owner");
  const holder = await signUp(request, "funded-holder", "Ren Holder");
  const fundedTitle = `Funded task ${crypto.randomUUID()}`;
  const draftTitle = `Unfunded draft ${crypto.randomUUID()}`;

  const fundedTaskId = await fundedOpenTask(
    request,
    owner,
    fundedTitle,
    20,
    "reservation_required",
  );
  await postAsUser(
    request,
    holder,
    `/api/tasks/${fundedTaskId}/reservations`,
    {},
  );
  // A declared-but-unfunded credit reward stays a draft and shows the
  // warning-toned unfunded badge in My tasks.
  await postAsUser(
    request,
    owner,
    "/api/tasks",
    taskRequest(draftTitle, owner.subjectId, "public", 15),
  );

  await signIn(page, owner.email);
  await page.getByTestId("nav-tasks").click();
  const fundedRow = page.getByTestId("task-row").filter({
    hasText: fundedTitle,
  });
  await expect(fundedRow.getByTestId("task-funded")).toContainText("funded");
  await expect(fundedRow.getByTestId("task-reserved")).toHaveText(
    "· reserved by Ren Holder",
  );
  const draftRow = page.getByTestId("task-row").filter({
    hasText: draftTitle,
  });
  await expect(draftRow.getByTestId("task-unfunded")).toContainText(
    "unfunded",
  );

  // Discovery offers a funded-only filter that queries funded=reward_funded;
  // the reserved-but-funded task reappears with its badge and holder once
  // reserved tasks are included.
  await page.getByTestId("discovery-filters").click();
  await page.getByTestId("include-reserved").click();
  const filtered = page.waitForRequest((candidate) =>
    candidate.url().includes("/api/tasks?scope=public") &&
    candidate.url().includes("funded=reward_funded")
  );
  await page.getByTestId("discovery-funded-only").click();
  await filtered;
  await page.getByTestId("discovery-query").fill(fundedTitle);
  const discoveryRow = page.getByTestId("discovery-task-row").filter({
    hasText: fundedTitle,
  });
  await expect(discoveryRow).toHaveCount(1);
  await expect(discoveryRow.getByTestId("task-funded")).toContainText(
    "funded",
  );
  await expect(discoveryRow.getByTestId("task-reserved")).toHaveText(
    "· reserved by Ren Holder",
  );
});
