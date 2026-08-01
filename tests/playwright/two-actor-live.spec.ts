import { type BrowserContext, expect, type Page, test } from "@playwright/test";
import process from "node:process";
import { fillDetailResponse, password, uniqueEmail } from "./helpers.ts";

// Two live browser sessions - a requester and a worker signed in as
// different registered users at the same time - drive one task through its
// whole loop. The point over the single-page review specs is liveness: each
// side's UI must surface the other side's actions through the 15-second poll
// (activity feed, unread badge) without a reload, and each side must see the
// counterparty's display name.

const apiOrigin = `http://127.0.0.1:${
  process.env.SHARECROP_PLAYWRIGHT_API_PORT ?? "29180"
}`;

// The UI polls every 15 seconds; give each poll-driven expectation two poll
// cycles plus slack so one slow request cannot flake the test.
const pollTimeout = 40_000;

interface LiveActor {
  displayName: string;
  context: BrowserContext;
  page: Page;
}

test("two live sessions follow one task loop through the poll", async ({ browser, request }) => {
  // Two poll-driven waits of up to 40s dominate this test's budget.
  test.setTimeout(180_000);

  async function liveActor(
    prefix: string,
    displayName: string,
  ): Promise<LiveActor> {
    const email = uniqueEmail(prefix);
    const registered = await request.post("/api/auth/register", {
      data: { email, password, display_name: displayName },
    });
    expect(registered.ok(), `register ${email}`).toBeTruthy();
    const context = await browser.newContext({ baseURL: apiOrigin });
    const page = await context.newPage();
    await page.goto("/");
    await page.getByTestId("email").fill(email);
    await page.getByTestId("password").fill(password);
    await page.getByTestId("login").click();
    await expect(page.getByTestId("balance")).toBeVisible({ timeout: 15000 });
    return { displayName, context, page };
  }

  const requester = await liveActor("live-requester", "Rita Requester");
  const worker = await liveActor("live-worker", "Wren Worker");
  const title = `Live loop ${crypto.randomUUID()}`;

  // The requester creates, funds, and opens a public reservation-required
  // credit task entirely through their UI.
  const requesterPage = requester.page;
  await requesterPage.getByTestId("nav-tasks").click();
  await requesterPage.getByTestId("new-task-button").click();
  await requesterPage.getByTestId("create-title").fill(title);
  await requesterPage.getByTestId("create-description").fill(
    "Reserve this task, then submit a one-line answer to close the loop.",
  );
  await requesterPage.getByTestId("create-reward-kind-credit").click();
  await requesterPage.getByTestId("create-reward").fill("25");
  await requesterPage.getByTestId("create-task-ownership").click();
  await requesterPage.getByTestId("create-participation-reservation_required")
    .click();
  await requesterPage.getByTestId("create-visibility-public").click();
  await requesterPage.getByTestId("create-task").click();
  await expect(requesterPage.getByTestId("detail-title")).toContainText(title);
  await requesterPage.getByTestId("fund-amount").fill("25");
  await requesterPage.getByTestId("fund").click();
  // Successful funding refetches the detail, and the funded draft swaps the
  // open-by-default funding callout for the collapsed panel.
  await expect(requesterPage.getByTestId("fund-task-callout")).toHaveCount(0);
  await requesterPage.getByTestId("open-task").click();
  await expect(requesterPage.getByTestId("task-action-message")).toContainText(
    "Task opened",
  );
  // Park the requester on Overview: everything they learn from here on
  // arrives through the poll, never through a reload.
  await requesterPage.getByTestId("nav-overview").click();
  await expect(requesterPage.getByTestId("activity-row").first())
    .toBeVisible();

  // The worker discovers the task in the Discover section, reserves it, and
  // submits a response.
  const workerPage = worker.page;
  await workerPage.getByTestId("nav-tasks").click();
  await workerPage.getByTestId("discovery-filters").click();
  await workerPage.getByTestId("discovery-query").fill(title);
  const discovered = workerPage.getByTestId("discovery-task-row").filter({
    hasText: title,
  });
  await expect(discovered).toHaveCount(1);
  await discovered.getByTestId("discovery-view").click();
  await expect(workerPage.getByTestId("detail-title")).toContainText(title);
  await workerPage.getByTestId("reserve-task").click();
  await expect(workerPage.getByTestId("reservation-message")).toBeVisible();
  await fillDetailResponse(workerPage, '{"answer":"done, live"}');
  await workerPage.getByTestId("detail-submit").click();
  await expect(workerPage.getByTestId("detail-submit-message")).toBeVisible();

  // The requester's parked Overview picks the submission up from the poll:
  // the activity feed names the worker, and the unread badge lights up.
  const submittedRow = requesterPage.getByTestId("activity-row")
    .filter({ hasText: title })
    .filter({ hasText: "submitted a response to" });
  await expect(submittedRow).toHaveCount(1, { timeout: pollTimeout });
  await expect(submittedRow).toContainText(worker.displayName);
  await expect(requesterPage.getByTestId("nav-unread-count")).toBeVisible({
    timeout: pollTimeout,
  });

  // The inbox notification also names the worker, and its task link leads to
  // the review; the requester accepts.
  await requesterPage.getByTestId("nav-account-menu").click();
  await requesterPage.getByTestId("nav-inbox").click();
  const needsReview = requesterPage.getByTestId("notification-row")
    .filter({ hasText: "submitted a response to" })
    .filter({ hasText: worker.displayName });
  await expect(needsReview.first()).toBeVisible();
  await needsReview.first().getByTestId("notification-task-link").click();
  await expect(requesterPage.getByTestId("detail-title")).toContainText(title);
  await expect(requesterPage.getByTestId("submission-row")).toHaveCount(1);
  await requesterPage.getByTestId("accept-submission").click();
  await expect(requesterPage.getByTestId("accept-submission")).toHaveCount(0);

  // The worker's session learns of the acceptance from the poll (they were
  // left on the task detail page), and the inbox names the requester on the
  // acceptance and carries the payout row.
  await expect(workerPage.getByTestId("nav-unread-count")).toBeVisible({
    timeout: pollTimeout,
  });
  await workerPage.getByTestId("nav-account-menu").click();
  await workerPage.getByTestId("nav-inbox").click();
  const acceptedRow = workerPage.getByTestId("notification-row")
    .filter({ hasText: "accepted your submission to" });
  await expect(acceptedRow.first()).toBeVisible();
  await expect(acceptedRow.first()).toContainText(requester.displayName);
  const payoutRow = workerPage.getByTestId("notification-row")
    .filter({ hasText: "You received a payout for" });
  await expect(payoutRow.first()).toBeVisible();

  // Entering Overview refreshes the balance: signup grant plus the payout.
  await workerPage.getByTestId("nav-overview").click();
  await expect(workerPage.getByTestId("balance")).toHaveText("125 credits");

  await requester.context.close();
  await worker.context.close();
});
