/// <reference lib="dom" />
import { expect, test } from "@playwright/test";

// The demo (real Elm client + in-browser fake backend) is served at :18081 by
// the static webServer in playwright.config.ts. It boots into a seeded account.
const demoOrigin = "http://127.0.0.1:18081";

test.use({ viewport: { width: 375, height: 667 } });

test("the demo renders without horizontal overflow across pages on a phone", async ({ page }) => {
  await page.goto(`${demoOrigin}/index.html`);
  await expect(page.getByText("1240 credits")).toBeVisible();

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
});
