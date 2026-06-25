import { expect, test } from "@playwright/test";

// The demo serves the REAL compiled Elm client (site/demo) against an in-browser
// fake backend (backend.js). It is hosted by the static webServer in
// playwright.config.ts (Browser.application needs a real HTTP origin).
const demoOrigin = "http://127.0.0.1:18081";

test("demo boots the real Elm client against the fake backend with seeded tasks", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);

  // Boots straight into the seeded account (refresh auto-succeeds in the shim).
  await expect(page.getByText("1240 credits")).toBeVisible();
  // Ledger + My-tasks decode and populate (seed enum values must match the real
  // client's decoders, else Decode.list blanks the whole section).
  await expect(page.getByText("signup_grant")).toBeVisible();
  await page.getByRole("link", { name: "Tasks", exact: true }).click();
  await expect(page.getByText("Verify 10 ledger transfers for fraud signals"))
    .toBeVisible();

  // The real client's Discovery page lists the realistic seeded tasks.
  await page.getByRole("link", { name: "Discovery" }).click();
  await expect(
    page.getByText("Extract line items from 6 vendor invoices"),
  ).toBeVisible();
  await expect(page.getByText("Classify 8 support tickets by category"))
    .toBeVisible();
  await expect(page.getByText("Write release notes for 5 changelog entries"))
    .toBeVisible();

  // Opening a task shows the real detail view with its instructions, the
  // self-contained Task input (all data embedded), and the response schema.
  await page.getByTestId("discovery-view").first().click();
  await expect(
    page.getByText("OCR'd text of 6 vendor invoices", { exact: false }),
  )
    .toBeVisible();
  // The Task input block embeds the actual data needed to solve the task.
  await expect(page.getByTestId("detail-input")).toBeVisible();
  await expect(page.getByText("Birch Supply Co", { exact: false }))
    .toBeVisible();
  await expect(page.getByText('"invoice_id"', { exact: false })).toBeVisible();

  // Reserve-then-submit, with the response validated against the task schema
  // (the demo enforces the schema like the real backend): a malformed response
  // is recorded "invalid", a schema-correct one "submitted".
  await page.getByTestId("reserve-task").click();
  await page.getByTestId("detail-submit-input").fill("{}");
  await page.getByTestId("detail-submit").click();
  await expect(page.getByTestId("detail-submit-message")).toContainText(
    "invalid",
  );
  await page.getByTestId("detail-submit-input").fill(
    '{"invoices":[{"invoice_id":"INV-1041","vendor":"Birch Supply Co","total":"1240.55","due_date":"2026-07-12"}]}',
  );
  await page.getByTestId("detail-submit").click();
  await expect(page.getByTestId("detail-submit-message")).toContainText(
    "submitted",
  );
});

test("demo organization page shows a funded balance, not a stuck spinner", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1240 credits")).toBeVisible();

  await page.getByRole("link", { name: "Organizations" }).click();
  await page.getByTestId("select-organization").first().click();

  // The fake backend serves a per-organization balance, so the label resolves
  // to a real number instead of being stuck on "Loading…".
  await expect(page.getByText("Balance: 7200 credits")).toBeVisible();
  await expect(page.getByText("Balance: Loading…")).toHaveCount(0);
});

test("demo owner can refund a funded task they own", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1240 credits")).toBeVisible();

  await page.getByRole("link", { name: "Tasks", exact: true }).click();
  await page
    .getByTestId("task-row")
    .filter({ hasText: "Verify 10 ledger transfers for fraud signals" })
    .getByTestId("view-task")
    .click();

  // The owner controls offer Refund; the fake backend implements /refund and
  // returns the escrow shape the client decodes, so the action succeeds.
  await page.getByTestId("refund-task").click();
  await expect(page.getByTestId("create-message")).toContainText(
    "Task refunded and cancelled.",
  );
});
