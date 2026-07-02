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
  for (
    const name of [
      "Discovery",
      "Tasks",
      "New task",
      "Series",
      "Funding",
      "Agents",
      "Collectibles",
      "Organizations",
    ]
  ) {
    await page.getByRole("link", { name, exact: true }).click();
    await expectNoHorizontalOverflow(name);
  }

  // A task detail page (instruction/code blocks are a mobile overflow risk).
  await page.getByRole("link", { name: "Discovery", exact: true }).click();
  await page.getByTestId("discovery-view").first().click();
  await expect(page.getByTestId("detail-title")).toBeVisible();
  await expectNoHorizontalOverflow("task detail");

  // The expanded API & MCP panel with a minted token: long curl/JSON/.mcp.json
  // code blocks must stay contained on a phone.
  await page.getByTestId("toggle-integration").click();
  await page.getByTestId("mint-task-token").click();
  await expect(page.getByTestId("integration-token")).toBeVisible();
  await expectNoHorizontalOverflow("task detail with API & MCP panel");

  // The user's own profile page mints a personal token with long MCP commands.
  await page.getByTestId("nav-profile").click();
  await page.getByTestId("mint-user-token").click();
  await expect(page.getByTestId("user-token")).toBeVisible();
  await expectNoHorizontalOverflow("profile agent access");
});
