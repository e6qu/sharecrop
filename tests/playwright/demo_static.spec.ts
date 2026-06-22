import { expect, test } from "@playwright/test";

const demoUrl = `file://${Deno.cwd()}/site/demo/index.html`;

test("static demo supports theme, user, local state, and reset flows", async ({ page }) => {
  await page.goto(demoUrl);

  await expect(page.getByRole("heading", {
    name: "Design tasks, reserve implementors, and pay fair rewards.",
  })).toBeVisible();

  await page.getByRole("button", { name: "Dark" }).click();
  await page.getByRole("button", { name: /Blocky/ }).click();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "dark");
  await expect(page.locator("body")).toHaveAttribute("data-theme", "blocky");

  await page.getByLabel("User").selectOption("jules");
  await expect(page.getByRole("heading", {
    name: "Find work, reserve it, and submit structured results.",
  })).toBeVisible();

  await page.getByLabel("Title").fill("Demo persistence task");
  await page.getByRole("button", { name: "Add demo task" }).click();
  await expect(page.getByRole("button", { name: /Demo persistence task/ }))
    .toBeVisible();

  await page.reload();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "dark");
  await expect(page.locator("body")).toHaveAttribute("data-theme", "blocky");
  await expect(page.getByRole("button", { name: /Demo persistence task/ }))
    .toBeVisible();

  await page.getByRole("button", { name: "Clear demo state" }).click();
  await expect(page.locator("body")).toHaveAttribute("data-mode", "light");
  await expect(page.locator("body")).toHaveAttribute("data-theme", "corporate");
  await expect(page.getByRole("button", { name: /Demo persistence task/ }))
    .toHaveCount(0);
});
