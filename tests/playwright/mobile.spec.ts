/// <reference lib="dom" />
import { expect, test } from "@playwright/test";
import process from "node:process";

// The demo (real Elm client + compiled Go/WASM backend path) is served by the
// static webServer in playwright.config.ts. It boots into a seeded account.
const demoOrigin = process.env.SHARECROP_PLAYWRIGHT_DEMO_ORIGIN ??
  "http://127.0.0.1:29181";

test.use({ viewport: { width: 375, height: 667 } });

test("the demo renders without horizontal overflow across pages on a phone", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1250 credits")).toBeVisible();

  async function expectNoHorizontalOverflow(where: string) {
    // Wait for the pixel-art web fonts to load before measuring; font reflow
    // otherwise causes a transient width spike mid-navigation.
    await page.evaluate(() => document.fonts.ready);
    await expect
      .poll(
        () =>
          page.evaluate(
            () =>
              document.documentElement.scrollWidth -
              document.documentElement.clientWidth,
          ),
        { message: `horizontal overflow on ${where}`, timeout: 4000 },
      )
      .toBeLessThanOrEqual(1);
  }

  await expectNoHorizontalOverflow("overview");

  // The Manage/Account nav dropdowns float over page content on a narrow
  // phone viewport, so their open panels are their own overflow risk
  // distinct from the pages they link to.
  await page.getByTestId("nav-manage-menu").click();
  await expectNoHorizontalOverflow("nav manage menu open");
  await page.getByTestId("nav-manage-menu").click();
  await page.getByTestId("nav-account-menu").click();
  await expectNoHorizontalOverflow("nav account menu open");
  await page.getByTestId("nav-account-menu").click();

  // The Tasks hub: My tasks and Discover public tasks are always expanded;
  // My submissions and Series are collapsed disclosures, each its own
  // overflow risk once expanded (new content, not covered by the checks
  // above).
  await page.getByTestId("nav-tasks").click();
  await expectNoHorizontalOverflow("tasks hub");
  await page.getByTestId("tasks-submissions").click();
  await expectNoHorizontalOverflow("tasks hub with my-submissions expanded");
  await page.getByTestId("tasks-series").click();
  await expectNoHorizontalOverflow("tasks hub with series expanded");

  for (
    const [name, menu] of [
      ["Funding", "nav-manage-menu"],
      ["Agents", "nav-manage-menu"],
      ["Collectibles", "nav-manage-menu"],
      ["Organizations", "nav-manage-menu"],
    ] as const
  ) {
    await page.getByTestId(menu).click();
    await page.getByRole("link", { name, exact: true }).click();
    await expectNoHorizontalOverflow(name);
  }

  // A task detail page (instruction/code blocks are a mobile overflow risk).
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("discovery-view").first().click();
  await expect(page.getByTestId("detail-title")).toBeVisible();
  await expectNoHorizontalOverflow("task detail");

  // The expanded API & MCP panel with a minted token: long curl/JSON/.mcp.json
  // code blocks must stay contained on a phone.
  await page.getByTestId("toggle-integration").click();
  await page.getByTestId("mint-task-token").click();
  await expect(page.getByTestId("integration-token")).toBeVisible();
  await expectNoHorizontalOverflow("task detail with API & MCP panel");

  // The owned task's inline "Fund this task" panel (organization picker plus
  // an amount input) is new content and its own overflow risk. Create a
  // draft task fresh rather than relying on a seeded task's state; creating
  // a task now opens it directly in the UI for further editing.
  await page.getByTestId("nav-tasks").click();
  await page.getByTestId("new-task-button").click();
  await page.getByTestId("create-title").fill("Mobile overflow fund check");
  await page.getByTestId("create-description").fill(
    "Created to exercise the fund panel's mobile layout.",
  );
  await page.getByTestId("create-task").click();
  await expect(page.getByTestId("detail-title")).toHaveText(
    "Mobile overflow fund check",
  );
  await page.getByTestId("fund-task-panel").click();
  await expectNoHorizontalOverflow("task detail with fund panel expanded");

  // The user's own profile page mints a personal token with long MCP commands.
  await page.getByTestId("nav-account-menu").click();
  await page.getByTestId("nav-profile").click();
  await page.getByTestId("mint-user-token").click();
  await expect(page.getByTestId("user-token")).toBeVisible();
  await expectNoHorizontalOverflow("profile agent access");
});
