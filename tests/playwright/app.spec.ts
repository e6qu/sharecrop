import { expect, test } from "npm:@playwright/test";

test("app shell loads", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Sharecrop" })).toBeVisible();
});
