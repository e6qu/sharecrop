import { expect, test } from "@playwright/test";
import { Buffer } from "node:buffer";
import process from "node:process";

// The demo serves the real compiled Elm client against the compiled Go/WASM
// backend path. It is hosted by the static webServer in playwright.config.ts
// because Browser.application needs a real HTTP origin.
const demoOrigin = process.env.SHARECROP_PLAYWRIGHT_DEMO_ORIGIN ??
  "http://127.0.0.1:29181";

test("demo boots the real Elm client against the Go/WASM backend with seeded tasks", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);

  // Boots straight into the seeded account (refresh auto-succeeds in the shim).
  await expect(page.getByText("1250 credits")).toBeVisible();
  // Ledger + My-tasks decode and populate (seed enum values must match the real
  // client's decoders, else Decode.list blanks the whole section).
  await expect(page.getByText("Signup grant")).toBeVisible();
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

test("demo uploads small task and submission attachments", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();

  await page.getByTestId("nav-create-task").click();
  const taskTitle = `Attachment demo ${Date.now()}`;
  await page.getByTestId("create-title").fill(taskTitle);
  await page.getByTestId("create-description").fill("Task with a small brief.");
  await page.getByTestId("create-advanced-options").click();
  const createChooser = page.waitForEvent("filechooser");
  await page.getByTestId("create-attachments-pick").click();
  await (await createChooser).setFiles({
    name: "brief.txt",
    mimeType: "text/plain",
    buffer: Buffer.from("brief"),
  });
  await expect(page.getByTestId("selected-attachment")).toContainText(
    "brief.txt",
  );
  await page.getByTestId("create-task").click();
  await expect(page.getByTestId("create-message")).toContainText(
    "Created task",
  );
  await page.getByTestId("nav-tasks").click();
  await page
    .getByTestId("task-row")
    .filter({ hasText: taskTitle })
    .getByTestId("view-task")
    .click();
  await expect(page.getByTestId("detail-attachments")).toContainText(
    "brief.txt",
  );

  await page.getByRole("link", { name: "Discovery" }).click();
  await page
    .getByTestId("discovery-task-row")
    .filter({ hasText: "Classify 8 support tickets by category" })
    .getByTestId("discovery-view")
    .click();
  await page.getByTestId("detail-submit-input").fill(
    '{"labels":["billing","bug","other","billing","account","feature_request","billing","other"]}',
  );
  const submitChooser = page.waitForEvent("filechooser");
  await page.getByTestId("submit-attachments-pick").click();
  await (await submitChooser).setFiles({
    name: "evidence.png",
    mimeType: "image/png",
    buffer: Buffer.from([0x89, 0x50, 0x4e, 0x47]),
  });
  await expect(page.getByTestId("selected-attachment")).toContainText(
    "evidence.png",
  );
  await page.getByTestId("detail-submit").click();
  await expect(page.getByTestId("detail-submit-message")).toContainText(
    "submitted",
  );
  await expect(page.getByTestId("my-submissions")).toContainText(
    "evidence.png",
  );
});

test("demo organization page shows a funded balance, not a stuck spinner", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();

  await page.getByTestId("nav-manage-menu").click();
  await page.getByRole("link", { name: "Organizations" }).click();
  await page.getByTestId("select-organization").first().click();

  // The WASM backend serves a per-organization balance, so the label resolves
  // to a real number instead of being stuck on "Loading…".
  await expect(page.getByText("Balance: 7200 credits")).toBeVisible();
  await expect(page.getByText("Balance: Loading…")).toHaveCount(0);
});

test("demo admin resolves privacy requests from the browser", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();

  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-profile").click();
  await page.getByTestId("account-privacy").click();
  await page.getByTestId("request-data-export").click();
  await expect(page.getByTestId("account-message")).toContainText(
    "Privacy request queued",
  );

  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-admin").click();
  await expect(page.getByTestId("admin-audit-page-offset")).toHaveText(
    "Offset 0",
  );
  await expect(page.getByTestId("admin-platform-admins-page-offset"))
    .toHaveText("Offset 0");
  await expect(page.getByTestId("admin-privacy-page-offset")).toHaveText(
    "Offset 0",
  );
  await expect(page.getByTestId("admin-moderation-page-offset")).toHaveText(
    "Offset 0",
  );
  await expect(page.getByTestId("admin-privacy-request")).toHaveCount(1);
  await page.getByTestId("admin-section-privacy").click();
  await page.getByTestId("admin-privacy-note").fill("Export generated");
  await page.getByTestId("admin-resolve-privacy").click();
  await expect(page.getByTestId("admin-message")).toContainText(
    "Privacy request resolved.",
  );
  await expect(page.getByTestId("admin-privacy-export")).toContainText(
    "user-mara",
  );
});

test("demo admin config grants and revokes platform admins", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();

  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-admin").click();
  await page.getByTestId("admin-section-platform-admins").click();
  await page.getByTestId("admin-platform-user").selectOption("user-jules");
  await page.getByTestId("admin-grant-platform-admin").click();
  await expect(page.getByTestId("admin-message")).toContainText(
    "Platform admin granted.",
  );
  await expect(page.getByTestId("admin-platform-admins")).toContainText(
    "user-jules",
  );

  await page
    .getByTestId("admin-platform-admin")
    .filter({ hasText: "user-jules" })
    .getByTestId("admin-revoke-platform-admin")
    .click();
  await expect(page.getByTestId("admin-message")).toContainText(
    "Platform admin revoked.",
  );
  await expect(page.getByTestId("admin-platform-admins")).not.toContainText(
    "user-jules",
  );
});

test("demo admin runs privacy retention from the browser", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();

  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-admin").click();
  await page.getByTestId("admin-section-privacy").click();
  await page.getByTestId("admin-run-privacy-retention").click();
  await expect(page.getByTestId("admin-message")).toContainText(
    "Privacy retention run finished.",
  );
  await expect(page.getByTestId("admin-retention-count")).toContainText(
    "Redacted fields:",
  );
});

test("demo task reports appear in the admin moderation panel", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();

  await page.getByRole("link", { name: "Discovery" }).click();
  await page.getByTestId("discovery-view").first().click();
  await page.getByTestId("moderation-reason-pii").click();
  await page.getByTestId("moderation-details").fill("Contains invoice PII.");
  await page.getByTestId("report-task").click();
  await expect(page.getByTestId("moderation-message")).toContainText(
    "Report submitted: pii",
  );

  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-admin").click();
  await expect(page.getByTestId("admin-moderation-report")).toHaveCount(1);
  await expect(page.getByTestId("admin-moderation-report")).toContainText(
    "task",
  );
  await expect(page.getByTestId("admin-moderation-details")).toContainText(
    "Contains invoice PII.",
  );
});

test("demo admin triages moderation reports from the browser", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();

  await page.getByRole("link", { name: "Discovery" }).click();
  await page.getByTestId("discovery-view").first().click();
  await page.getByTestId("moderation-reason-policy").click();
  await page.getByTestId("moderation-details").fill("Needs admin decision.");
  await page.getByTestId("report-task").click();
  await expect(page.getByTestId("moderation-message")).toContainText(
    "Report submitted: policy",
  );

  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-admin").click();
  await expect(page.getByTestId("admin-moderation-report")).toHaveCount(1);
  await expect(page.getByTestId("admin-moderation-subject-link")).toHaveCount(
    1,
  );
  await page.getByTestId("admin-section-moderation").click();
  await page.getByTestId("admin-moderation-note").fill("Handled by admin");
  await page.getByTestId("admin-moderation-resolve").click();
  await expect(page.getByTestId("admin-message")).toContainText(
    "Moderation report updated.",
  );
  await expect(page.getByTestId("admin-moderation-report")).toContainText(
    "resolved",
  );
  await expect(page.getByTestId("admin-moderation-resolution-note"))
    .toContainText(
      "Handled by admin",
    );

  await page.getByTestId("admin-moderation-state").selectOption("dismissed");
  await expect(page.getByTestId("admin-moderation-empty")).toBeVisible();
  await page.getByTestId("admin-moderation-state").selectOption("resolved");
  await expect(page.getByTestId("admin-moderation-report")).toHaveCount(1);
});

test("demo owner can refund a funded task they own", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();

  await page.getByRole("link", { name: "Tasks", exact: true }).click();
  await page
    .getByTestId("task-row")
    .filter({ hasText: "Verify 10 ledger transfers for fraud signals" })
    .getByTestId("view-task")
    .click();

  // The owner controls offer Refund; the WASM backend implements /refund and
  // returns the escrow shape the client decodes, so the action succeeds.
  await page.getByTestId("refund-task").click();
  await expect(page.getByTestId("task-action-message")).toContainText(
    "Task refunded and cancelled.",
  );
});

test("the collectibles catalog renders sprites, awards a default, and trades it", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();
  await page.getByTestId("nav-manage-menu").click();
  await page.getByRole("link", { name: "Collectibles", exact: true }).click();

  // The 25 default collectibles render as a gallery of pixel sprites.
  await expect(page.getByTestId("catalog-entry")).toHaveCount(25);

  // Award one to myself (the demo user id), then it appears in my holdings.
  await page.getByTestId("award-default-section").click();
  await page.getByTestId("award-recipient-id-query").fill("mara");
  await page.getByTestId("award-recipient-id").selectOption({
    label: "mara@sharecrop.demo",
  });
  await page.getByTestId("catalog-award").first().click();
  await expect(page.getByTestId("award-default-message")).toContainText(
    "Awarded",
  );
  // Open the newly held collectible and trade it to another user.
  await page.getByTestId("collectible-link").first().click();
  await expect(page.getByTestId("collectible-detail-name")).toBeVisible();
  await page.getByTestId("transfer-recipient-id-query").fill("jules");
  await page.getByTestId("transfer-recipient-id").selectOption({
    label: "jules@sharecrop.demo",
  });
  await page.getByTestId("transfer-collectible").click();
  await expect(page.getByTestId("transfer-message")).toContainText(
    "Transferred",
  );
});

test("demo creates and opens a task series", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();

  // A real bug found by hand-testing the demo: /api/task-series was entirely
  // unclassified in the WASM backend (a 404), so this whole flow was broken.
  await page.getByTestId("nav-work-menu").click();
  await page.getByRole("link", { name: "Series", exact: true }).click();
  await page.getByTestId("series-create-title").fill("Sprint 1");
  await page.getByTestId("series-create-description").fill(
    "A batch of related tasks.",
  );
  await page.getByTestId("create-series").click();
  await expect(page.getByTestId("series-message")).toContainText(
    "Series saved.",
  );
  await expect(page.getByTestId("series-row")).toContainText("Sprint 1");
  await expect(page.getByTestId("series-row")).toContainText("draft");

  await page.getByTestId("open-series").first().click();
  await expect(page.getByTestId("series-detail-title")).toContainText(
    "Sprint 1",
  );
});

test("the demo shows a Reset button and hash routing keeps a stable URL on refresh", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();

  // The demo-only Reset control is present.
  await page.getByTestId("nav-account-menu").click();
  await expect(page.getByTestId("reset-demo")).toBeVisible();

  // Navigation updates the fragment, and a hard refresh stays on the page.
  await page.getByTestId("nav-manage-menu").click();
  await page.getByRole("link", { name: "Collectibles", exact: true }).click();
  await expect(page).toHaveURL(/#\/collectibles$/);
  await page.reload();
  await expect(page).toHaveURL(/#\/collectibles$/);
  await expect(page.getByTestId("catalog")).toBeVisible();
});
