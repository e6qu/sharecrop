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
  await expect(page.getByText("Classify 20 support tickets by category"))
    .toBeVisible();

  // Opening a task shows the real detail view with its instructions and the
  // typed response schema, served by the fake backend.
  await page.getByTestId("discovery-view").first().click();
  await expect(
    page.getByText("Read the 6 attached invoice scans", { exact: false }),
  )
    .toBeVisible();
  await expect(page.getByText('"invoice_id"', { exact: false })).toBeVisible();
});
