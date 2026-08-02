import { expect, test } from "@playwright/test";
import {
  type AuthBody,
  password,
  uniqueEmail,
  verifyAccountWithToken,
} from "./helpers.ts";

// Browser flows for the peer economy against the real backend: sending
// credits, sending collectibles, the mint confirmation, per-page document
// titles, and the load-error state that keeps an API failure from
// masquerading as an empty list.

async function registerViaAPI(
  request: import("@playwright/test").APIRequestContext,
  emailPrefix: string,
): Promise<{ email: string; body: AuthBody }> {
  const email = uniqueEmail(emailPrefix);
  const response = await request.post("/api/auth/register", {
    data: { email, password },
  });
  expect(response.ok()).toBeTruthy();
  const body = (await response.json()) as AuthBody;
  await verifyAccountWithToken(request, body.access_token);
  return { email, body };
}

async function loginViaUI(
  page: import("@playwright/test").Page,
  email: string,
): Promise<void> {
  await page.goto("/");
  await page.getByTestId("email").fill(email);
  await page.getByTestId("password").fill(password);
  await page.getByTestId("login").click();
  await expect(page.getByTestId("balance")).toHaveText("100 credits");
}

test("sending credits moves the balance and notes both ledgers", async ({ page, request }) => {
  const sender = await registerViaAPI(request, "ui-send-a");
  const recipient = await registerViaAPI(request, "ui-send-b");

  await loginViaUI(page, sender.email);

  await page.getByTestId("send-credits-panel").click();
  await page.getByTestId("send-recipient-id-query").fill(recipient.email);
  await page.getByTestId("send-recipient-id").selectOption({
    label: recipient.email,
  });
  await page.getByTestId("send-amount").fill("25");
  await page.getByTestId("send-note").fill("for the harvest");
  await page.getByTestId("send-credits").click();

  await expect(page.getByTestId("send-message")).toHaveText(
    `Sent 25 credits to ${recipient.email}.`,
  );
  await expect(page.getByTestId("balance")).toHaveText("75 credits");
  // The sender-side ledger row carries the note.
  await expect(page.getByTestId("ledger-entry-note").first()).toHaveText(
    "for the harvest",
  );

  // The recipient's account received the credits.
  const recipientBalance = await request.get("/api/credits/balance", {
    headers: { Authorization: `Bearer ${recipient.body.access_token}` },
  });
  expect(recipientBalance.ok()).toBeTruthy();
  expect(await recipientBalance.json()).toEqual({
    spendable_credits: 125,
    allocated_credits: 0,
  });
});

test("sending credits without a recipient is blocked with a reason", async ({ page, request }) => {
  const sender = await registerViaAPI(request, "ui-send-guard");
  await loginViaUI(page, sender.email);

  await page.getByTestId("send-credits-panel").click();
  await page.getByTestId("send-amount").fill("10");
  await page.getByTestId("send-credits").click();
  await expect(page.getByTestId("send-message")).toHaveText(
    "Choose a recipient first.",
  );
});

test("a transferable collectible row offers Send and a non-transferable one explains why not", async ({ page, request }) => {
  const owner = await registerViaAPI(request, "ui-row-send-a");
  const recipient = await registerViaAPI(request, "ui-row-send-b");
  await loginViaUI(page, owner.email);

  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-collectibles").click();
  await page.getByTestId("collectibles-mint").click();

  // A non-transferable mint: Send is disabled and the reason is spelled out.
  const lockedName = `Locked badge ${crypto.randomUUID()}`;
  await page.getByTestId("collectible-name").fill(lockedName);
  await page.getByTestId("mint-collectible").click();
  // The mint confirmation is visible right by the mint button.
  await expect(page.getByTestId("collectible-message")).toContainText(
    `Minted ${lockedName}`,
  );
  const lockedRow = page.getByTestId("collectible-row").filter({
    hasText: lockedName,
  });
  await expect(lockedRow.getByTestId("send-collectible-toggle")).toBeDisabled();
  await expect(
    lockedRow.getByTestId("send-collectible-unavailable"),
  ).toContainText("only moves as a task payout");

  // A transferable mint: Send opens the row panel and moves the collectible.
  const name = `Traveling badge ${crypto.randomUUID()}`;
  await page.getByTestId("collectible-name").fill(name);
  await page.getByTestId("collectible-policy-transferable_between_users")
    .click();
  await page.getByTestId("mint-collectible").click();
  const row = page.getByTestId("collectible-row").filter({ hasText: name });
  await expect(row).toHaveCount(1);
  await row.getByTestId("send-collectible-toggle").click();
  await page.getByTestId("send-collectible-recipient-query").fill(
    recipient.email,
  );
  await page.getByTestId("send-collectible-recipient").selectOption({
    label: recipient.email,
  });
  await page.getByTestId("send-collectible").click();
  await expect(page.getByTestId("transfer-message")).toHaveText(
    `Sent ${name}.`,
  );
  // The collectible left the sender's holdings.
  await expect(page.getByTestId("collectible-row").filter({ hasText: name }))
    .toHaveCount(0);
});

test("document titles name the page", async ({ page, request }) => {
  const account = await registerViaAPI(request, "ui-titles");
  await page.goto("/");
  await expect(page).toHaveTitle("Sharecrop");
  await loginViaUI(page, account.email);
  await expect(page).toHaveTitle("Sharecrop — Overview");
  await page.getByTestId("nav-tasks").click();
  await expect(page).toHaveTitle("Sharecrop — Tasks");
});

test("a failed list load shows an error state, not a fake empty state", async ({ page, request }) => {
  const account = await registerViaAPI(request, "ui-load-error");
  await loginViaUI(page, account.email);

  // Genuine emptiness first: a fresh account's Tasks page shows the
  // empty-state, not an error.
  await page.getByTestId("nav-tasks").click();
  await expect(page.getByTestId("tasks-empty")).toBeVisible();
  await expect(page.getByTestId("tasks-load-error")).toHaveCount(0);

  // Now the same list with the API failing: the error state replaces the
  // empty state and carries the server's message.
  await page.route(
    (url) => url.pathname === "/api/tasks" && url.search.length > 0,
    (route) => {
      if (route.request().method() !== "GET") {
        return route.fallback();
      }
      return route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({
          code: "internal",
          error: "the database is unavailable",
        }),
      });
    },
  );
  await page.getByTestId("nav-overview").click();
  await page.getByTestId("nav-tasks").click();
  await expect(page.getByTestId("tasks-load-error")).toContainText(
    "the database is unavailable",
  );
  await expect(page.getByTestId("tasks-load-error")).toContainText(
    "Retrying on the next refresh.",
  );
  await expect(page.getByTestId("tasks-empty")).toHaveCount(0);
  // The public-discovery section hit the same route, so it reports the same
  // failure rather than "No public tasks available."
  await expect(page.getByTestId("discovery-load-error")).toBeVisible();
  await expect(page.getByTestId("discovery-empty")).toHaveCount(0);
});
